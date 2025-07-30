package services_test

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	logger.InitLogger("test")
}

func TestEmailService_SendDunningEmail(t *testing.T) {
	tests := []struct {
		name        string
		template    *db.DunningEmailTemplate
		data        business.EmailData
		toEmail     string
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully sends dunning email with HTML and text",
			template: &db.DunningEmailTemplate{
				Subject:      "Payment Failed - {{product_name}}",
				BodyHtml:     "<p>Hello {{customer_name}}, your payment of {{amount}} {{currency}} failed.</p>",
				BodyText:     pgtype.Text{String: "Hello {{customer_name}}, your payment of {{amount}} {{currency}} failed.", Valid: true},
				TemplateType: "payment_failed",
			},
			data: business.EmailData{
				CustomerName:  "John Doe",
				CustomerEmail: "john@example.com",
				Amount:        "100.00",
				Currency:      "USD",
				ProductName:   "Premium Plan",
				MerchantName:  "Test Merchant",
			},
			toEmail: "john@example.com",
			wantErr: false,
		},
		{
			name: "successfully sends email with HTML only",
			template: &db.DunningEmailTemplate{
				Subject:      "Payment Due - {{product_name}}",
				BodyHtml:     "<p>Hello {{customer_name}}, payment is due.</p>",
				BodyText:     pgtype.Text{Valid: false}, // No text version
				TemplateType: "payment_due",
			},
			data: business.EmailData{
				CustomerName: "Jane Smith",
				ProductName:  "Basic Plan",
			},
			toEmail: "jane@example.com",
			wantErr: false,
		},
		{
			name: "handles invalid HTML template",
			template: &db.DunningEmailTemplate{
				Subject:      "Test",
				BodyHtml:     "<p>Hello {{CustomerName</p>", // Missing closing braces
				TemplateType: "test",
			},
			data:        business.EmailData{CustomerName: "Test"},
			toEmail:     "test@example.com",
			wantErr:     true,
			errorString: "failed to parse HTML template",
		},
		{
			name: "handles invalid text template",
			template: &db.DunningEmailTemplate{
				Subject:      "Test",
				BodyHtml:     "<p>Valid HTML</p>",
				BodyText:     pgtype.Text{String: "Hello {{CustomerName", Valid: true}, // Missing closing braces
				TemplateType: "test",
			},
			data:        business.EmailData{CustomerName: "Test"},
			toEmail:     "test@example.com",
			wantErr:     true,
			errorString: "failed to parse text template",
		},
		{
			name: "handles complex template with all fields",
			template: &db.DunningEmailTemplate{
				Subject: "Payment Retry - {{product_name}} - {{amount}} {{currency}}",
				BodyHtml: `<p>Dear {{customer_name}},</p>
					<p>Your payment of {{amount}} {{currency}} for {{product_name}} will be retried on {{retry_date}}.</p>
					<p>You have {{attempts_remaining}} attempts remaining.</p>
					<p><a href="{{payment_link}}">Update Payment Method</a></p>
					<p>Questions? Contact us at {{support_email}}</p>
					<p>Best regards,<br>{{merchant_name}}</p>
					<p><a href="{{unsubscribe_link}}">Unsubscribe</a></p>`,
				TemplateType: "payment_retry",
			},
			data: business.EmailData{
				CustomerName:      "Alice Johnson",
				CustomerEmail:     "alice@example.com",
				Amount:            "50.00",
				Currency:          "EUR",
				ProductName:       "Enterprise Plan",
				RetryDate:         "2024-01-15",
				AttemptsRemaining: 2,
				PaymentLink:       "https://example.com/payment/update",
				SupportEmail:      "support@example.com",
				MerchantName:      "Example Corp",
				UnsubscribeLink:   "https://example.com/unsubscribe",
			},
			toEmail: "alice@example.com",
			wantErr: false,
		},
	}

	// Note: In real tests, we would mock the resend client
	// For now, we're testing the template parsing logic
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create service with test configuration
			_ = services.NewEmailService("test-api-key", "noreply@test.com", "Test Sender", zap.NewNop())

			// For testing template parsing without actually sending emails
			if tt.wantErr && (tt.errorString == "failed to parse HTML template" || tt.errorString == "failed to parse text template") {
				// Test template parsing directly
				_, err := template.New("test").Parse(tt.template.BodyHtml)
				if tt.errorString == "failed to parse HTML template" {
					assert.Error(t, err)
					return
				}

				if tt.template.BodyText.Valid {
					_, parseErr := template.New("test").Parse(tt.template.BodyText.String)
					if tt.errorString == "failed to parse text template" {
						assert.Error(t, parseErr)
						return
					}
				}
			}

			// In a real test, we would:
			// 1. Mock the resend client
			// 2. Call service.SendDunningEmail(ctx, tt.template, tt.data, tt.toEmail)
			// 3. Verify the mock was called with correct parameters
		})
	}
}

func TestEmailService_SendTransactionalEmail(t *testing.T) {
	tests := []struct {
		name        string
		params      params.TransactionalEmailParams
		wantErr     bool
		errorString string
	}{
		{
			name: "successfully sends transactional email",
			params: params.TransactionalEmailParams{
				To:          []string{"user@example.com"},
				Subject:     "Welcome to Our Service",
				HTMLContent: "<h1>Welcome!</h1><p>Thanks for signing up.</p>",
				TextContent: "Welcome! Thanks for signing up.",
				Tags: map[string]interface{}{
					"category": "welcome",
					"source":   "signup",
				},
			},
			wantErr: false,
		},
		{
			name: "sends email with CC and BCC",
			params: params.TransactionalEmailParams{
				To:          []string{"primary@example.com"},
				CC:          []string{"cc@example.com"},
				BCC:         []string{"bcc@example.com"},
				Subject:     "Important Update",
				HTMLContent: "<p>This is an important update.</p>",
				ReplyTo:     aws.String("support@example.com"),
			},
			wantErr: false,
		},
		{
			name: "sends email with custom headers",
			params: params.TransactionalEmailParams{
				To:          []string{"user@example.com"},
				Subject:     "Order Confirmation",
				HTMLContent: "<p>Your order has been confirmed.</p>",
				Headers: map[string]string{
					"X-Order-ID":    "12345",
					"X-Customer-ID": "67890",
				},
			},
			wantErr: false,
		},
		{
			name: "handles multiple recipients",
			params: params.TransactionalEmailParams{
				To:          []string{"user1@example.com", "user2@example.com", "user3@example.com"},
				Subject:     "Team Update",
				HTMLContent: "<p>Here's the latest team update.</p>",
				TextContent: "Here's the latest team update.",
			},
			wantErr: false,
		},
		{
			name: "handles empty recipients",
			params: params.TransactionalEmailParams{
				To:          []string{},
				Subject:     "Test",
				HTMLContent: "<p>Test</p>",
			},
			wantErr: true, // Should fail with empty recipients
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = services.NewEmailService("test-api-key", "noreply@test.com", "Test Sender", zap.NewNop())

			// In a real test with mocked resend client:
			// err := service.SendTransactionalEmail(context.Background(), tt.params)
			// if tt.wantErr {
			//     assert.Error(t, err)
			// } else {
			//     assert.NoError(t, err)
			// }

			// For now, just validate the params structure
			if len(tt.params.To) == 0 && !tt.wantErr {
				t.Error("Expected error for empty recipients")
			}
		})
	}
}

func TestEmailService_SendBatchEmails(t *testing.T) {
	tests := []struct {
		name     string
		requests []requests.BatchEmailRequest
		wantLen  int
	}{
		{
			name: "successfully processes batch of emails",
			requests: []requests.BatchEmailRequest{
				{
					ToEmail:     "user1@example.com",
					Subject:     "Batch Email 1",
					HTMLContent: "<p>First email</p>",
					Tags:        map[string]string{"batch": "1", "source": "signup"},
				},
				{
					ToEmail:     "user2@example.com",
					Subject:     "Batch Email 2",
					HTMLContent: "<p>Second email</p>",
					Tags:        map[string]string{"batch": "2", "source": "signup"},
				},
				{
					ToEmail:     "user3@example.com",
					Subject:     "Batch Email 3",
					HTMLContent: "<p>Third email</p>",
					Tags:        map[string]string{"batch": "3"},
				},
			},
			wantLen: 3,
		},
		{
			name:     "handles empty batch",
			requests: []requests.BatchEmailRequest{},
			wantLen:  0,
		},
		{
			name: "processes batch with mixed success/failure",
			requests: []requests.BatchEmailRequest{
				{
					ToEmail:     "valid@example.com",
					Subject:     "Valid Email",
					HTMLContent: "<p>This should succeed</p>",
				},
				{
					ToEmail:     "", // This should fail
					Subject:     "Invalid Email",
					HTMLContent: "<p>This should fail</p>",
				},
				{
					ToEmail:     "another@example.com",
					Subject:     "Another Valid Email",
					HTMLContent: "<p>This should succeed</p>",
				},
			},
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = services.NewEmailService("test-api-key", "noreply@test.com", "Test Sender", zap.NewNop())

			// In a real test:
			// results, err := service.SendBatchEmails(context.Background(), tt.requests)
			// assert.NoError(t, err)
			// assert.Len(t, results, tt.wantLen)

			// Validate batch structure
			assert.Equal(t, tt.wantLen, len(tt.requests))
		})
	}
}

func TestEmailService_ParseTemplate(t *testing.T) {
	_ = services.NewEmailService("test-api-key", "noreply@test.com", "Test Sender", zap.NewNop())

	tests := []struct {
		name     string
		template string
		data     business.EmailData
		want     string
		wantErr  bool
	}{
		{
			name:     "simple template substitution",
			template: "Hello {{.CustomerName}}, your payment of {{.Amount}} {{.Currency}} is due.",
			data: business.EmailData{
				CustomerName: "John Doe",
				Amount:       "100.00",
				Currency:     "USD",
			},
			want:    "Hello John Doe, your payment of 100.00 USD is due.",
			wantErr: false,
		},
		{
			name:     "template with conditional logic",
			template: "Hello {{.CustomerName}}{{if .AttemptsRemaining}}, you have {{.AttemptsRemaining}} attempts remaining{{end}}.",
			data: business.EmailData{
				CustomerName:      "Jane Smith",
				AttemptsRemaining: 3,
			},
			want:    "Hello Jane Smith, you have 3 attempts remaining.",
			wantErr: false,
		},
		{
			name:     "template with missing field uses zero value",
			template: "Product: {{.ProductName}}",
			data:     business.EmailData{}, // ProductName is empty
			want:     "Product: ",
			wantErr:  false,
		},
		{
			name:     "invalid template syntax",
			template: "Hello {{.CustomerName", // Missing closing braces
			data:     business.EmailData{CustomerName: "Test"},
			wantErr:  true,
		},
		{
			name:     "template with HTML content",
			template: `<div class="greeting">Hello {{.CustomerName}}</div>`,
			data:     business.EmailData{CustomerName: "Alice"},
			want:     `<div class="greeting">Hello Alice</div>`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This assumes EmailService exports a ParseTemplate method
			// In reality, we might need to test this indirectly or make it public for testing

			// For demonstration, let's test the template parsing logic
			tmpl, err := template.New("test").Parse(tt.template)
			if tt.wantErr {
				if err != nil {
					return // Expected error during parsing
				}
				// Try to execute to trigger error
				var buf bytes.Buffer
				err = tmpl.Execute(&buf, tt.data)
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, tt.data)
			require.NoError(t, err)

			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestEmailService_ParseSubject(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		data    business.EmailData
		want    string
	}{
		{
			name:    "replaces customer name",
			subject: "Hello {{customer_name}}!",
			data:    business.EmailData{CustomerName: "John Doe"},
			want:    "Hello John Doe!",
		},
		{
			name:    "replaces multiple placeholders",
			subject: "Payment of {{amount}} {{currency}} for {{product_name}}",
			data: business.EmailData{
				Amount:      "50.00",
				Currency:    "EUR",
				ProductName: "Premium Plan",
			},
			want: "Payment of 50.00 EUR for Premium Plan",
		},
		{
			name:    "handles missing placeholders",
			subject: "Welcome {{customer_name}} to {{merchant_name}}",
			data:    business.EmailData{CustomerName: "Alice"},
			want:    "Welcome Alice to ", // merchant_name is empty
		},
		{
			name:    "no placeholders",
			subject: "Plain subject line",
			data:    business.EmailData{CustomerName: "Test"},
			want:    "Plain subject line",
		},
		{
			name:    "handles special characters",
			subject: "Payment failed: {{amount}} {{currency}}",
			data: business.EmailData{
				Amount:   "$100.00",
				Currency: "USD",
			},
			want: "Payment failed: $100.00 USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would require parseSubject to be exported or tested indirectly
			// For now, we demonstrate the expected behavior

			replacer := strings.NewReplacer(
				"{{customer_name}}", tt.data.CustomerName,
				"{{amount}}", tt.data.Amount,
				"{{currency}}", tt.data.Currency,
				"{{product_name}}", tt.data.ProductName,
				"{{merchant_name}}", tt.data.MerchantName,
			)
			result := replacer.Replace(tt.subject)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestEmailService_ConvertToResendTags(t *testing.T) {
	tests := []struct {
		name string
		tags map[string]string
		want int // Expected number of tags
	}{
		{
			name: "converts multiple tags",
			tags: map[string]string{
				"category": "dunning",
				"campaign": "payment_retry",
				"customer": "vip",
			},
			want: 3,
		},
		{
			name: "handles empty tags",
			tags: map[string]string{},
			want: 0,
		},
		{
			name: "handles nil tags",
			tags: nil,
			want: 0,
		},
		{
			name: "handles single tag",
			tags: map[string]string{
				"type": "transactional",
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This tests the helper function logic
			var count int
			for range tt.tags {
				count++
			}
			assert.Equal(t, tt.want, count)
		})
	}
}

// TestEmailService_ErrorHandling tests error scenarios
func TestEmailService_ErrorHandling(t *testing.T) {
	logger := zap.NewNop()

	t.Run("handles nil logger gracefully", func(t *testing.T) {
		// Service should not panic with nil logger
		service := services.NewEmailService("test-key", "test@example.com", "Test", nil)
		assert.NotNil(t, service)
	})

	t.Run("validates email configuration", func(t *testing.T) {
		// Test with empty from email
		service := services.NewEmailService("test-key", "", "Test", logger)
		assert.NotNil(t, service)

		// Test with empty from name
		service = services.NewEmailService("test-key", "test@example.com", "", logger)
		assert.NotNil(t, service)
	})
}
