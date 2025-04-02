package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cyphera-api/internal/client"
	"cyphera-api/internal/db"
	"cyphera-api/internal/handlers"
	"cyphera-api/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

const (
	usageText = `Cyphera Subscription Processor

Usage:
  subscription-processor [options]

Options:
  --interval=DURATION   Time between subscription checks (default: 5m)
                        Valid time units: s (seconds), m (minutes), h (hours)
                        Examples: 30s, 5m, 1h, 2h30m
  
  --once               Run once and exit (don't continue running)
  
  --help               Show this help message

Examples:
  subscription-processor --interval=15m
  subscription-processor --once
  subscription-processor --interval=1h30m

Environment variables required:
  DATABASE_URL                  PostgreSQL connection string
  CYPHERA_SMART_WALLET_ADDRESS  Address of Cyphera smart wallet contract
`
)

func main() {
	err := godotenv.Load()
	if err != nil {
		// It's often okay if the .env file is missing, especially in production
		// where variables might be set directly in the environment.
		// Log it but don't necessarily stop the application.
		log.Printf("Warning: Error loading .env file: %v", err)
	}
	// Set custom usage message
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usageText)
	}

	// Parse command line arguments
	interval := flag.String("interval", "5m", "Check interval (e.g., 1h, 30m, 1m)")
	oneTime := flag.Bool("once", false, "Run once and exit")
	flag.Parse()

	// Initialize logger
	logger.InitLogger()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	// Parse and validate interval duration
	checkDuration, err := time.ParseDuration(*interval)
	if err != nil {
		log.Fatalf("Invalid interval format: %v\nValid formats examples: 30s, 5m, 1h, 2h30m", err)
	}

	// Provide feedback on the selected interval
	log.Printf("Using check interval: %s", formatDuration(checkDuration))

	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("DATABASE_URL environment variable is required")
	}

	// Create a connection pool
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse database connection string: %v", err)
	}

	// Configure the connection pool
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 15

	// Create the connection pool
	connPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v", err)
	}
	defer connPool.Close()

	// Create queries instance with the connection pool
	dbQueries := db.New(connPool)

	// Initialize the delegation client
	delegationClient, err := client.NewDelegationClient()
	if err != nil {
		log.Fatalf("Failed to initialize delegation client: %v", err)
	}
	defer delegationClient.Close()

	// Get the Cyphera smart wallet address
	cypheraSmartWalletAddress := os.Getenv("CYPHERA_SMART_WALLET_ADDRESS")
	if cypheraSmartWalletAddress == "" {
		log.Fatalf("CYPHERA_SMART_WALLET_ADDRESS environment variable is required")
	}

	// Create common services
	commonServices := handlers.NewCommonServices(dbQueries, cypheraSmartWalletAddress)

	// Create subscription handler
	subscriptionHandler := handlers.NewSubscriptionHandler(commonServices, delegationClient)

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for termination signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Run immediately the first time
	log.Printf("Starting subscription processor, checking for due subscriptions...")
	processSubscriptions(ctx, subscriptionHandler)

	// If one-time mode is requested, exit after the first run
	if *oneTime {
		log.Printf("One-time execution completed, exiting")
		return
	}

	// Setup ticker for periodic execution
	log.Printf("Will check for subscriptions every %s", formatDuration(checkDuration))
	ticker := time.NewTicker(checkDuration)
	defer ticker.Stop()

	// Main loop
	for {
		select {
		case <-ticker.C:
			log.Printf("Checking for due subscriptions...")
			processSubscriptions(ctx, subscriptionHandler)
		case sig := <-signalChan:
			log.Printf("Received signal %v, shutting down...", sig)
			return
		}
	}
}

// processSubscriptions runs the subscription processor and logs the results
func processSubscriptions(ctx context.Context, handler *handlers.SubscriptionHandler) {
	results, err := handler.ProcessDueSubscriptions(ctx)
	if err != nil {
		log.Printf("Error processing subscriptions: %v", err)
		return
	}

	log.Printf("Processed %d subscriptions: %d succeeded, %d failed, %d completed",
		results.Total, results.Succeeded, results.Failed, results.Completed)
}

// formatDuration formats a duration in a more human-friendly way
func formatDuration(d time.Duration) string {
	// Round duration to make it more readable
	d = d.Round(time.Second)

	// Handle hours
	h := d / time.Hour
	d -= h * time.Hour

	// Handle minutes
	m := d / time.Minute
	d -= m * time.Minute

	// Handle seconds
	s := d / time.Second

	// Build string representation
	parts := []string{}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 && h == 0 { // Only show seconds if less than an hour
		parts = append(parts, fmt.Sprintf("%ds", s))
	}

	return strings.Join(parts, " ")
}
