package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/resend/resend-go/v2"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/db"
)

type EmailService struct {
	client *resend.Client
	logger *zap.Logger
	fromEmail string
	fromName  string
}

func NewEmailService(apiKey string, fromEmail string, fromName string, logger *zap.Logger) *EmailService {
	client := resend.NewClient(apiKey)
	
	return &EmailService{
		client:    client,
		logger:    logger,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// EmailData contains the data for email templates
type EmailData struct {
	CustomerName       string
	CustomerEmail      string
	Amount             string
	Currency           string
	ProductName        string
	RetryDate          string
	AttemptsRemaining  int
	PaymentLink        string
	SupportEmail       string
	MerchantName       string
	UnsubscribeLink    string
}

// SendDunningEmail sends a dunning email using a template
func (s *EmailService) SendDunningEmail(ctx context.Context, template *db.DunningEmailTemplate, data EmailData, toEmail string) error {
	// Parse and execute HTML template
	htmlContent, err := s.parseTemplate(template.BodyHtml, data)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Parse and execute text template if available
	var textContent string
	if template.BodyText.Valid && template.BodyText.String != "" {
		textContent, err = s.parseTemplate(template.BodyText.String, data)
		if err != nil {
			return fmt.Errorf("failed to parse text template: %w", err)
		}
	}

	// Prepare email parameters
	from := fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{toEmail},
		Subject: s.parseSubject(template.Subject, data),
		Html:    htmlContent,
		Text:    textContent,
		Headers: map[string]string{
			"X-Entity-Ref-ID": uuid.New().String(),
			"X-Campaign-Type": "dunning",
		},
		Tags: []resend.Tag{
			{Name: "category", Value: "dunning"},
			{Name: "template_type", Value: template.TemplateType},
		},
	}

	// Send email
	sent, err := s.client.Emails.Send(params)
	if err != nil {
		s.logger.Error("failed to send dunning email", 
			zap.Error(err),
			zap.String("to", toEmail),
			zap.String("template_type", template.TemplateType))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("dunning email sent successfully",
		zap.String("email_id", sent.Id),
		zap.String("to", toEmail),
		zap.String("template_type", template.TemplateType))

	return nil
}

// SendTransactionalEmail sends a general transactional email
func (s *EmailService) SendTransactionalEmail(ctx context.Context, params TransactionalEmailParams) error {
	from := fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail)
	
	emailParams := &resend.SendEmailRequest{
		From:    from,
		To:      params.To,
		Subject: params.Subject,
		Html:    params.HTMLBody,
		Text:    params.TextBody,
		Cc:      params.Cc,
		Bcc:     params.Bcc,
		ReplyTo: params.ReplyTo,
		Headers: params.Headers,
		Tags:    convertToResendTags(params.Tags),
	}

	sent, err := s.client.Emails.Send(emailParams)
	if err != nil {
		s.logger.Error("failed to send transactional email", 
			zap.Error(err),
			zap.Strings("to", params.To),
			zap.String("subject", params.Subject))
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("transactional email sent successfully",
		zap.String("email_id", sent.Id),
		zap.Strings("to", params.To),
		zap.String("subject", params.Subject))

	return nil
}

// SendBatchEmails sends multiple emails in a batch
func (s *EmailService) SendBatchEmails(ctx context.Context, requests []BatchEmailRequest) ([]BatchEmailResult, error) {
	results := make([]BatchEmailResult, len(requests))
	
	// Process each email
	for i, req := range requests {
		params := TransactionalEmailParams{
			To:       req.To,
			Subject:  req.Subject,
			HTMLBody: req.HTMLBody,
			TextBody: req.TextBody,
			Tags:     req.Tags,
		}
		
		err := s.SendTransactionalEmail(ctx, params)
		results[i] = BatchEmailResult{
			Index:   i,
			Success: err == nil,
			Error:   err,
		}
		
		// Add small delay to avoid rate limiting
		if i < len(requests)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	return results, nil
}

// parseTemplate parses and executes a template with the given data
func (s *EmailService) parseTemplate(templateStr string, data EmailData) (string, error) {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// parseSubject replaces simple placeholders in the subject line
func (s *EmailService) parseSubject(subject string, data EmailData) string {
	replacer := strings.NewReplacer(
		"{{customer_name}}", data.CustomerName,
		"{{amount}}", data.Amount,
		"{{currency}}", data.Currency,
		"{{product_name}}", data.ProductName,
		"{{merchant_name}}", data.MerchantName,
	)
	return replacer.Replace(subject)
}

// Helper types

type TransactionalEmailParams struct {
	To       []string
	Subject  string
	HTMLBody string
	TextBody string
	Cc       []string
	Bcc      []string
	ReplyTo  string
	Headers  map[string]string
	Tags     map[string]string
}

type BatchEmailRequest struct {
	To       []string
	Subject  string
	HTMLBody string
	TextBody string
	Tags     map[string]string
}

type BatchEmailResult struct {
	Index   int
	Success bool
	Error   error
}

func convertToResendTags(tags map[string]string) []resend.Tag {
	var resendTags []resend.Tag
	for name, value := range tags {
		resendTags = append(resendTags, resend.Tag{
			Name:  name,
			Value: value,
		})
	}
	return resendTags
}

// Email templates for common scenarios

func GetDefaultDunningTemplates() map[string]DunningEmailTemplate {
	return map[string]DunningEmailTemplate{
		"pre_dunning": {
			Subject: "Payment Due Soon - {{product_name}}",
			BodyHTML: `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f4f4f4; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .button { display: inline-block; padding: 10px 20px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Payment Reminder</h2>
        </div>
        <div class="content">
            <p>Hi {{.CustomerName}},</p>
            <p>This is a friendly reminder that your payment of <strong>{{.Amount}} {{.Currency}}</strong> for {{.ProductName}} is due soon.</p>
            <p>Please ensure your payment method is up to date to avoid any interruption to your service.</p>
            <p><a href="{{.PaymentLink}}" class="button">Update Payment Method</a></p>
            <p>If you have any questions, please contact us at {{.SupportEmail}}.</p>
            <p>Best regards,<br>{{.MerchantName}} Team</p>
        </div>
        <div class="footer">
            <p><a href="{{.UnsubscribeLink}}">Unsubscribe</a> from these notifications</p>
        </div>
    </div>
</body>
</html>`,
			BodyText: `Hi {{.CustomerName}},

This is a friendly reminder that your payment of {{.Amount}} {{.Currency}} for {{.ProductName}} is due soon.

Please ensure your payment method is up to date to avoid any interruption to your service.

Update your payment method: {{.PaymentLink}}

If you have any questions, please contact us at {{.SupportEmail}}.

Best regards,
{{.MerchantName}} Team

Unsubscribe from these notifications: {{.UnsubscribeLink}}`,
		},
		"attempt_1": {
			Subject: "Payment Failed - Action Required for {{product_name}}",
			BodyHTML: `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .button { display: inline-block; padding: 10px 20px; background-color: #28a745; color: white; text-decoration: none; border-radius: 5px; }
        .warning { background-color: #fff3cd; border: 1px solid #ffeaa7; padding: 10px; margin: 10px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Payment Failed</h2>
        </div>
        <div class="content">
            <p>Hi {{.CustomerName}},</p>
            <p>We were unable to process your payment of <strong>{{.Amount}} {{.Currency}}</strong> for {{.ProductName}}.</p>
            <div class="warning">
                <p><strong>Important:</strong> We'll retry your payment on {{.RetryDate}}. You have {{.AttemptsRemaining}} automatic retry attempts remaining.</p>
            </div>
            <p>To avoid service interruption, please update your payment method now:</p>
            <p style="text-align: center;"><a href="{{.PaymentLink}}" class="button">Update Payment Method</a></p>
            <p>If you believe this is an error or need assistance, please contact us immediately at {{.SupportEmail}}.</p>
            <p>Best regards,<br>{{.MerchantName}} Team</p>
        </div>
        <div class="footer">
            <p><a href="{{.UnsubscribeLink}}">Unsubscribe</a> from these notifications</p>
        </div>
    </div>
</body>
</html>`,
		},
		"final_notice": {
			Subject: "Final Notice - {{product_name}} Subscription at Risk",
			BodyHTML: `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .button { display: inline-block; padding: 15px 30px; background-color: #dc3545; color: white; text-decoration: none; border-radius: 5px; font-weight: bold; }
        .urgent { background-color: #f8d7da; border: 2px solid #dc3545; padding: 15px; margin: 15px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>FINAL NOTICE - Urgent Action Required</h2>
        </div>
        <div class="content">
            <p>Hi {{.CustomerName}},</p>
            <div class="urgent">
                <p><strong>This is your final notice.</strong> We have been unable to process your payment of <strong>{{.Amount}} {{.Currency}}</strong> for {{.ProductName}}.</p>
                <p><strong>Your subscription will be cancelled if payment is not received within 24 hours.</strong></p>
            </div>
            <p>This is your last opportunity to maintain uninterrupted service. Please update your payment method immediately:</p>
            <p style="text-align: center;"><a href="{{.PaymentLink}}" class="button">UPDATE PAYMENT NOW</a></p>
            <p>If you're experiencing difficulties or have questions, please contact our support team urgently at {{.SupportEmail}}. We're here to help.</p>
            <p>We value you as a customer and hope to continue serving you.</p>
            <p>Best regards,<br>{{.MerchantName}} Team</p>
        </div>
        <div class="footer">
            <p><a href="{{.UnsubscribeLink}}">Unsubscribe</a> from these notifications</p>
        </div>
    </div>
</body>
</html>`,
		},
		"recovery_success": {
			Subject: "Payment Successful - Thank You!",
			BodyHTML: `
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #28a745; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .success { background-color: #d4edda; border: 1px solid #28a745; padding: 10px; margin: 10px 0; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h2>Payment Successful!</h2>
        </div>
        <div class="content">
            <p>Hi {{.CustomerName}},</p>
            <div class="success">
                <p><strong>Good news!</strong> We've successfully processed your payment of <strong>{{.Amount}} {{.Currency}}</strong> for {{.ProductName}}.</p>
            </div>
            <p>Your subscription is now active and you can continue enjoying uninterrupted service.</p>
            <p>Thank you for your prompt action in resolving this matter.</p>
            <p>If you have any questions, feel free to reach out to us at {{.SupportEmail}}.</p>
            <p>Best regards,<br>{{.MerchantName}} Team</p>
        </div>
        <div class="footer">
            <p><a href="{{.UnsubscribeLink}}">Unsubscribe</a> from these notifications</p>
        </div>
    </div>
</body>
</html>`,
		},
	}
}

type DunningEmailTemplate struct {
	Subject  string
	BodyHTML string
	BodyText string
}