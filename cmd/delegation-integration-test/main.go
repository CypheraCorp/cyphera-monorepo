package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"cyphera-api/internal/client"
)

// Sample delegation JSON structure based on the expected format by the Node.js server
type Delegation struct {
	Delegator string   `json:"delegator"`
	Delegate  string   `json:"delegate"`
	Signature string   `json:"signature"`
	Expiry    int64    `json:"expiry"`
	Caveats   []string `json:"caveats"`
	Salt      string   `json:"salt"`
	Authority struct {
		Scheme    string `json:"scheme"`
		Signature string `json:"signature"`
		Signer    string `json:"signer"`
	} `json:"authority"`
}

func main() {
	// Parse command line flags
	delegatorFlag := flag.String("delegator", "0x1234567890123456789012345678901234567890", "Delegator address")
	delegateFlag := flag.String("delegate", "0x0987654321098765432109876543210987654321", "Delegate address")
	signatureFlag := flag.String("signature", "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "Delegation signature")
	expiryFlag := flag.Int64("expiry", time.Now().Unix()+3600, "Expiry timestamp (default: 1 hour from now)")
	saltFlag := flag.String("salt", "0x123456789", "Delegation salt")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	// Create a sample delegation
	delegation := Delegation{
		Delegator: *delegatorFlag,
		Delegate:  *delegateFlag,
		Signature: *signatureFlag,
		Expiry:    *expiryFlag,
		Caveats:   []string{},
		Salt:      *saltFlag,
		Authority: struct {
			Scheme    string `json:"scheme"`
			Signature string `json:"signature"`
			Signer    string `json:"signer"`
		}{
			Scheme:    "0x00",
			Signature: "0xsig",
			Signer:    "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		},
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the delegation client
	delegationClient, err := client.NewDelegationClient()
	if err != nil {
		log.Fatalf("Failed to create delegation client: %v", err)
	}
	defer delegationClient.Close()

	// Notify about test start
	log.Println("Starting delegation integration test...")
	log.Printf("Using delegator: %s", delegation.Delegator)
	log.Printf("Using delegate: %s", delegation.Delegate)
	log.Printf("Delegation expires at: %d (%s)", delegation.Expiry, time.Unix(delegation.Expiry, 0).Format(time.RFC3339))

	// Convert the delegation to JSON
	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		log.Fatalf("Failed to marshal delegation to JSON: %v", err)
	}

	if *verboseFlag {
		log.Printf("Delegation JSON: %s", string(delegationJSON))
	}

	// Call the gRPC service to redeem the delegation
	log.Println("Sending delegation to gRPC service...")
	txHash, err := delegationClient.RedeemDelegation(ctx, delegationJSON)
	if err != nil {
		log.Fatalf("Delegation redemption failed: %v", err)
	}

	// Print the result
	log.Printf("Delegation successfully redeemed! Transaction hash: %s", txHash)

	// Exit successfully
	os.Exit(0)
}
