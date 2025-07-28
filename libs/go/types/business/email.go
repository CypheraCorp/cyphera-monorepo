package business

// EmailData represents data for email templates
type EmailData struct {
	CustomerName      string
	CustomerEmail     string
	Amount            string
	Currency          string
	ProductName       string
	RetryDate         string
	AttemptsRemaining int
	PaymentLink       string
	SupportEmail      string
	MerchantName      string
	UnsubscribeLink   string
}
