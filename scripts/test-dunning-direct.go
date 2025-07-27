package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/resend/resend-go/v2"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/services"
)

func main() {
	// Get configuration from environment
	resendAPIKey := os.Getenv("RESEND_API_KEY")
	if resendAPIKey == "" {
		log.Fatal("RESEND_API_KEY environment variable is required")
	}

	testEmail := os.Getenv("TEST_EMAIL")
	if testEmail == "" {
		testEmail = "natefikru@gmail.com"
	}

	// Test direct email sending with Resend
	fmt.Println("Testing direct email send with Resend...")
	fmt.Printf("Sending to: %s\n", testEmail)

	// Create Resend client
	client := resend.NewClient(resendAPIKey)

	// Send a simple test email
	params := &resend.SendEmailRequest{
		From:    "Cyphera <noreply@cypherapay.com>",
		To:      []string{testEmail},
		Subject: "Test Dunning Email - Direct Send",
		Html: `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .button { display: inline-block; padding: 10px 20px; background-color: #28a745; color: white; text-decoration: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Test Dunning Email</h2>
        </div>
        <div class="content">
            <p>Hi Test User,</p>
            <p>This is a <strong>test dunning email</strong> sent directly via Resend API.</p>
            <p>If you received this email, the integration is working correctly!</p>
            <p>Test details:</p>
            <ul>
                <li>Amount: $99.99 USD</li>
                <li>Next retry: ` + time.Now().Add(24*time.Hour).Format("January 2, 2006") + `</li>
                <li>Attempts remaining: 3</li>
            </ul>
            <p style="text-align: center;"><a href="https://app.cyphera.com" class="button">Update Payment Method</a></p>
            <p>Best regards,<br>Cyphera Team</p>
        </div>
    </div>
</body>
</html>`,
		Text: `Test Dunning Email

Hi Test User,

This is a test dunning email sent directly via Resend API.

If you received this email, the integration is working correctly!

Test details:
- Amount: $99.99 USD
- Next retry: ` + time.Now().Add(24*time.Hour).Format("January 2, 2006") + `
- Attempts remaining: 3

Update your payment method: https://app.cyphera.com

Best regards,
Cyphera Team`,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	fmt.Printf("✅ Email sent successfully!\n")
	fmt.Printf("Email ID: %s\n", sent.Id)
	fmt.Printf("Check your inbox at: %s\n", testEmail)

	// Now test with the email service
	fmt.Println("\nTesting email service integration...")

	logger, _ := zap.NewDevelopment()
	emailService := services.NewEmailService(resendAPIKey, "noreply@cypherapay.com", "Cyphera", logger)

	// Create a test template
	template := &db.DunningEmailTemplate{
		ID:           uuid.New(),
		WorkspaceID:  uuid.New(),
		Name:         "Test Template",
		TemplateType: "attempt_1",
		Subject:      "Payment Failed - {{.CustomerName}}",
		BodyHtml: `<h1>Payment Failed</h1>
<p>Hi {{.CustomerName}},</p>
<p>Your payment of {{.Amount}} {{.Currency}} has failed.</p>
<p>Next retry: {{.RetryDate}}</p>
<p>Attempts remaining: {{.AttemptsRemaining}}</p>`,
		BodyText: pgtype.Text{
			String: `Payment Failed

Hi {{.CustomerName}},
Your payment of {{.Amount}} {{.Currency}} has failed.
Next retry: {{.RetryDate}}
Attempts remaining: {{.AttemptsRemaining}}`,
			Valid: true,
		},
	}

	// Send using the service
	emailData := services.EmailData{
		CustomerName:      "Test User",
		CustomerEmail:     testEmail,
		Amount:            "$99.99",
		Currency:          "USD",
		ProductName:       "Premium Subscription",
		RetryDate:         time.Now().Add(24 * time.Hour).Format("January 2, 2006"),
		AttemptsRemaining: 3,
		PaymentLink:       "https://app.cyphera.com/retry",
		SupportEmail:      "support@cyphera.com",
		MerchantName:      "Cyphera",
		UnsubscribeLink:   "https://app.cyphera.com/unsubscribe",
	}

	err = emailService.SendDunningEmail(context.Background(), template, emailData, testEmail)
	if err != nil {
		log.Printf("Failed to send via email service: %v", err)
	} else {
		fmt.Println("✅ Email sent via email service successfully!")
	}

	// Test with full dunning system
	fmt.Println("\nTo test the full dunning system:")
	fmt.Println("1. Start the API server")
	fmt.Println("2. Run: ./scripts/test-dunning-email.sh")
	fmt.Println("3. Create a test campaign in the database")
	fmt.Println("4. Call POST /api/v1/dunning/process to trigger processing")
}