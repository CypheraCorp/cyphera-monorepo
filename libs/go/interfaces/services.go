package interfaces

import (
	"context"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PaymentService handles payment processing operations
type PaymentService interface {
	CreatePaymentFromSubscriptionEvent(ctx context.Context, params params.CreatePaymentFromSubscriptionEventParams) (*db.Payment, error)
	CreateComprehensivePayment(ctx context.Context, params params.CreateComprehensivePaymentParams) (*db.Payment, error)
	GetPayment(ctx context.Context, params params.GetPaymentParams) (*db.Payment, error)
	GetPaymentByTransactionHash(ctx context.Context, txHash string) (*db.Payment, error)
	ListPayments(ctx context.Context, params params.ListPaymentsParams) ([]db.Payment, error)
	UpdatePaymentStatus(ctx context.Context, params params.UpdatePaymentStatusParams) (*db.Payment, error)
	GetPaymentMetrics(ctx context.Context, workspaceID uuid.UUID, startTime, endTime time.Time, currency string) (*db.GetPaymentMetricsRow, error)
	CreateManualPayment(ctx context.Context, params params.CreateManualPaymentParams) (*db.Payment, error)
}

// EmailService handles email sending operations
type EmailService interface {
	SendTransactionalEmail(ctx context.Context, params params.TransactionalEmailParams) error
	SendBatchEmails(ctx context.Context, requests []requests.BatchEmailRequest) ([]responses.BatchEmailResult, error)
	SendDunningEmail(ctx context.Context, template *db.DunningEmailTemplate, data map[string]business.EmailData, toEmail string) error
}

// SubscriptionService handles subscription operations
type SubscriptionService interface {
	GetSubscription(ctx context.Context, workspaceID, subscriptionID uuid.UUID) (*db.Subscription, error)
	ListSubscriptions(ctx context.Context, workspaceID uuid.UUID, limit, offset int32) ([]responses.SubscriptionResponse, int64, error)
	ListSubscriptionsByCustomer(ctx context.Context, workspaceID, customerID uuid.UUID) ([]responses.SubscriptionResponse, error)
	ListSubscriptionsByProduct(ctx context.Context, workspaceID, productID uuid.UUID) ([]db.Subscription, error)
	UpdateSubscription(ctx context.Context, subscriptionID uuid.UUID, req requests.UpdateSubscriptionRequest) (*db.Subscription, error)
	DeleteSubscription(ctx context.Context, workspaceID, subscriptionID uuid.UUID) error

	// New methods for subscription creation and processing
	StoreDelegationData(ctx context.Context, tx pgx.Tx, params params.StoreDelegationDataParams) (*db.DelegationDatum, error)
	CreateSubscriptionWithDelegation(ctx context.Context, tx pgx.Tx, params params.CreateSubscriptionWithDelegationParams) (*params.SubscriptionCreationResult, error)
	ProcessInitialRedemption(ctx context.Context, tx pgx.Tx, params params.InitialRedemptionParams) (*db.Subscription, error)
	ProcessDueSubscriptions(ctx context.Context) (*responses.ProcessDueSubscriptionsResult, error)
	CreateSubscription(ctx context.Context, tx pgx.Tx, params params.CreateSubscriptionParams) (*db.Subscription, error)
}

// InvoiceService handles invoice operations
type InvoiceService interface {
	CreateInvoice(ctx context.Context, params params.InvoiceCreateParams) (*responses.InvoiceResponse, error)
	GetInvoiceWithDetails(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*responses.InvoiceResponse, error)
	FinalizeInvoice(ctx context.Context, workspaceID, invoiceID uuid.UUID) (*db.Invoice, error)
}

// DunningService handles dunning campaigns
type DunningService interface {
	CreateConfiguration(ctx context.Context, params params.DunningConfigParams) (*db.DunningConfiguration, error)
	GetConfiguration(ctx context.Context, id uuid.UUID) (*db.DunningConfiguration, error)
	GetDefaultConfiguration(ctx context.Context, workspaceID uuid.UUID) (*db.DunningConfiguration, error)
	CreateCampaign(ctx context.Context, params params.DunningCampaignParams) (*db.DunningCampaign, error)
	CreateAttempt(ctx context.Context, params params.DunningAttemptParams) (*db.DunningAttempt, error)
	UpdateAttemptStatus(ctx context.Context, attemptID uuid.UUID, status string, errorMsg *string) (*db.DunningAttempt, error)
	RecoverCampaign(ctx context.Context, campaignID uuid.UUID, recoveredAmountCents int64) (*db.DunningCampaign, error)
	FailCampaign(ctx context.Context, campaignID uuid.UUID, finalAction string) (*db.DunningCampaign, error)
	CreateEmailTemplate(ctx context.Context, params params.EmailTemplateParams) (*db.DunningEmailTemplate, error)
	GetCampaignStats(ctx context.Context, workspaceID uuid.UUID, startDate, endDate time.Time) (*db.GetDunningCampaignStatsRow, error)
}

// ProrationCalculator handles proration calculations
type ProrationCalculator interface {
	CalculateUpgradeProration(currentPeriodStart, currentPeriodEnd time.Time, oldAmountCents, newAmountCents int64, changeDate time.Time) *business.ProrationResult
	ScheduleDowngrade(currentPeriodEnd time.Time, changeType string) *business.ScheduleChangeResult
	CalculatePauseCredit(currentPeriodStart, currentPeriodEnd time.Time, amountCents int64, pauseDate time.Time) *business.ProrationResult
	AddBillingPeriod(start time.Time, intervalType string, intervalCount int) time.Time
	FormatProrationExplanation(result *business.ProrationResult) string
}

// GasSponsorshipService handles gas sponsorship operations
type GasSponsorshipService interface {
	ShouldSponsorGas(ctx context.Context, params params.SponsorshipCheckParams) (*business.SponsorshipDecision, error)
	RecordSponsoredTransaction(ctx context.Context, record business.SponsorshipRecord) error
	GetSponsorshipBudgetStatus(ctx context.Context, workspaceID uuid.UUID) (*business.BudgetStatus, error)
	ResetMonthlySponsorshipBudgets(ctx context.Context) error
	CreateDefaultSponsorshipConfig(ctx context.Context, workspaceID uuid.UUID) error
	UpdateSponsorshipConfig(ctx context.Context, workspaceID uuid.UUID, updates business.SponsorshipConfigUpdates) error
	GetSponsorshipAnalytics(ctx context.Context, workspaceID uuid.UUID, days int) (*business.SponsorshipAnalytics, error)
}

// BlockchainService handles blockchain operations
type BlockchainService interface {
	Initialize(ctx context.Context) error
	GetTransactionData(ctx context.Context, txHash string, networkID uuid.UUID) (*business.TransactionData, error)
	GetTransactionDataFromEvent(ctx context.Context, event *db.SubscriptionEvent) (*business.TransactionData, error)
	Close()
}

// ExchangeRateService handles currency exchange rates
type ExchangeRateService interface {
	GetExchangeRate(ctx context.Context, params params.ExchangeRateParams) (*responses.ExchangeRateResult, error)
	ConvertAmount(ctx context.Context, amount float64, fromCurrency, toCurrency string) (float64, *responses.ExchangeRateResult, error)
}

// TaxService handles tax calculations
type TaxService interface {
	CalculateTax(ctx context.Context, params params.TaxCalculationParams) (*responses.TaxCalculationResult, error)
}

// PaymentLinkService handles payment link operations
type PaymentLinkService interface {
	CreatePaymentLink(ctx context.Context, params params.PaymentLinkCreateParams) (*responses.PaymentLinkResponse, error)
	GetPaymentLink(ctx context.Context, workspaceID, linkID uuid.UUID) (*responses.PaymentLinkResponse, error)
	GetPaymentLinkBySlug(ctx context.Context, slug string) (*responses.PaymentLinkResponse, error)
	UpdatePaymentLink(ctx context.Context, workspaceID, linkID uuid.UUID, updates params.PaymentLinkUpdateParams) (*responses.PaymentLinkResponse, error)
	DeactivatePaymentLink(ctx context.Context, workspaceID, linkID uuid.UUID) error
	GetBaseURL() string
	CreatePaymentLinkForInvoice(ctx context.Context, invoice db.Invoice) (*responses.PaymentLinkResponse, error)
}

// DiscountService handles discount operations
type DiscountService interface {
	ApplyDiscount(ctx context.Context, params params.DiscountApplicationParams) (*responses.DiscountApplicationResult, error)
}

// PaymentFailureMonitor monitors payment failures
type PaymentFailureMonitor interface {
	MonitorFailedPayments(ctx context.Context) error
	MonitorFailedSubscriptions(ctx context.Context) error
}

// GasFeeService handles gas fee calculations
type GasFeeService interface {
	EstimateGasFee(ctx context.Context, estimateGasFeeParams params.EstimateGasFeeParams) (*responses.EstimateGasFeeResult, error)
	GetCurrentGasPrice(ctx context.Context, networkID uuid.UUID) (int, error)
}

// DunningRetryEngine handles payment retry logic
type DunningRetryEngine interface {
	ProcessDueCampaigns(ctx context.Context, limit int32) error
}

// SubscriptionManagementService handles subscription changes
type SubscriptionManagementService interface {
	UpgradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newLineItems []requests.LineItemUpdate, reason string) error
	DowngradeSubscription(ctx context.Context, subscriptionID uuid.UUID, newLineItems []requests.LineItemUpdate, reason string) error
	CancelSubscription(ctx context.Context, subscriptionID uuid.UUID, reason string, feedback string) error
	PauseSubscription(ctx context.Context, subscriptionID uuid.UUID, pauseUntil *time.Time, reason string) error
	ResumeSubscription(ctx context.Context, subscriptionID uuid.UUID) error
	ReactivateCancelledSubscription(ctx context.Context, subscriptionID uuid.UUID) error
	PreviewChange(ctx context.Context, subscriptionID uuid.UUID, changeType string, lineItems []requests.LineItemUpdate) (*business.ChangePreview, error)
	GetSubscriptionHistory(ctx context.Context, subscriptionID uuid.UUID, limit int32) ([]db.SubscriptionStateHistory, error)
	ProcessScheduledChanges(ctx context.Context) error
}

// CustomerService handles customer operations
type CustomerService interface {
	CreateCustomer(ctx context.Context, params params.CreateCustomerParams) (*db.Customer, error)
	GetCustomer(ctx context.Context, id uuid.UUID) (*db.Customer, error)
	UpdateCustomer(ctx context.Context, params params.UpdateCustomerParams) (*db.Customer, error)
	DeleteCustomer(ctx context.Context, id uuid.UUID) error
	AddCustomerToWorkspace(ctx context.Context, workspaceID, customerID uuid.UUID) error
	UpdateCustomerOnboardingStatus(ctx context.Context, customerID uuid.UUID, finishedOnboarding bool) (*db.Customer, error)
	GetCustomerByWeb3AuthID(ctx context.Context, web3authID string) (*db.Customer, error)
	CreateCustomerWithWeb3Auth(ctx context.Context, params params.CreateCustomerWithWeb3AuthParams) (*db.Customer, error)
	ListCustomerWallets(ctx context.Context, customerID uuid.UUID) ([]db.CustomerWallet, error)
	CreateCustomerWallet(ctx context.Context, params params.CreateCustomerWalletParams) (*db.CustomerWallet, error)
	ListCustomers(ctx context.Context, params params.ListCustomersParams) (*responses.ListCustomersResult, error)
	ListWorkspaceCustomers(ctx context.Context, params params.ListWorkspaceCustomersParams) (*responses.ListWorkspaceCustomersResult, error)

	// New methods for subscription processing
	ProcessCustomerAndWallet(ctx context.Context, tx pgx.Tx, params params.ProcessCustomerWalletParams) (*db.Customer, *db.CustomerWallet, error)
	CreateCustomerFromWallet(ctx context.Context, tx pgx.Tx, params params.CreateCustomerFromWalletParams) (*db.Customer, *db.CustomerWallet, error)
	FindOrCreateCustomerWallet(ctx context.Context, tx pgx.Tx, params params.FindOrCreateWalletParams) (*db.CustomerWallet, error)
}

// WorkspaceService handles workspace operations
type WorkspaceService interface {
	CreateWorkspace(ctx context.Context, params params.CreateWorkspaceParams) (*db.Workspace, error)
	GetWorkspace(ctx context.Context, id uuid.UUID) (*db.Workspace, error)
	UpdateWorkspace(ctx context.Context, params params.UpdateWorkspaceParams) (*db.Workspace, error)
	DeleteWorkspace(ctx context.Context, id uuid.UUID) error
	GetAccountByWorkspace(ctx context.Context, workspaceID uuid.UUID) (*db.Account, error)
	ListWorkspacesByAccount(ctx context.Context, accountID uuid.UUID) ([]db.Workspace, error)
	ListAllWorkspaces(ctx context.Context) ([]db.Workspace, error)
	GetWorkspaceStats(ctx context.Context, workspaceID uuid.UUID) (*business.WorkspaceStats, error)
}

// ProductService handles product operations
type ProductService interface {
	CreateProduct(ctx context.Context, params params.CreateProductParams) (*db.Product, []db.Price, error)
	GetProduct(ctx context.Context, params params.GetProductParams) (*db.Product, []db.Price, error)
	ListProducts(ctx context.Context, params params.ListProductsParams) (*responses.ListProductsResult, error)
	UpdateProduct(ctx context.Context, params params.UpdateProductParams) (*db.Product, error)
	DeleteProduct(ctx context.Context, productID uuid.UUID, workspaceID uuid.UUID) error
	GetPublicProductByPriceID(ctx context.Context, priceID uuid.UUID) (*responses.PublicProductResponse, error)

	// Subscription validation methods
	ValidateSubscriptionRequest(ctx context.Context, params params.ValidateSubscriptionParams) error
	ValidateProductForSubscription(ctx context.Context, productID uuid.UUID) (*db.Product, error)
	ValidatePriceForSubscription(ctx context.Context, priceID uuid.UUID) (*db.Price, *db.Product, error)
	GetProductTokenWithValidation(ctx context.Context, productTokenID uuid.UUID, productID uuid.UUID) (*db.GetProductTokenRow, error)
}

// WalletService handles wallet operations
type WalletService interface {
	CreateWallet(ctx context.Context, params params.CreateWalletParams) (*db.Wallet, error)
	CreateWalletsForAllNetworks(ctx context.Context, params params.CreateWalletParams) ([]db.Wallet, error)
	GetWallet(ctx context.Context, walletID, workspaceID uuid.UUID) (*db.Wallet, error)
	GetWalletWithCircleData(ctx context.Context, walletID, workspaceID uuid.UUID) (*business.WalletWithCircleData, error)
	ListWalletsByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]db.Wallet, error)
	ListWalletsByType(ctx context.Context, workspaceID uuid.UUID, walletType string) ([]db.Wallet, error)
	ListCircleWallets(ctx context.Context, workspaceID uuid.UUID) ([]db.ListCircleWalletsByWorkspaceIDRow, error)
	ListWalletsWithCircleData(ctx context.Context, workspaceID uuid.UUID) ([]db.ListWalletsWithCircleDataByWorkspaceIDRow, error)
	UpdateWallet(ctx context.Context, workspaceID uuid.UUID, params params.UpdateWalletParams) (*db.Wallet, error)
	DeleteWallet(ctx context.Context, walletID, workspaceID uuid.UUID) error
	GetWalletByAddressAndNetwork(ctx context.Context, workspaceID uuid.UUID, walletAddress, networkType string) (*db.Wallet, error)
	UpdateWalletLastUsed(ctx context.Context, walletID uuid.UUID) error
	ValidateWalletAccess(ctx context.Context, walletID, workspaceID uuid.UUID) error
}

// TokenService handles token operations
type TokenService interface {
	GetToken(ctx context.Context, tokenID uuid.UUID) (*db.Token, error)
	GetTokenByAddress(ctx context.Context, networkID uuid.UUID, contractAddress string) (*db.Token, error)
	ListTokens(ctx context.Context) ([]db.Token, error)
	ListTokensByNetwork(ctx context.Context, networkID uuid.UUID) ([]db.Token, error)
	GetTokenQuote(ctx context.Context, params params.TokenQuoteParams) (*responses.TokenQuoteResult, error)
}

// NetworkService handles network operations
type NetworkService interface {
	GetNetwork(ctx context.Context, networkID uuid.UUID) (*db.Network, error)
	GetNetworkByChainID(ctx context.Context, chainID int32) (*db.Network, error)
	CreateNetwork(ctx context.Context, params params.CreateNetworkParams) (*db.Network, error)
	UpdateNetwork(ctx context.Context, params params.UpdateNetworkParams) (*db.Network, error)
	DeleteNetwork(ctx context.Context, networkID uuid.UUID) error
	ListNetworks(ctx context.Context, params params.ListNetworksParams) ([]db.Network, error)
	ListActiveTokensByNetwork(ctx context.Context, networkID uuid.UUID) ([]db.Token, error)
}

// APIKeyService handles API key operations
type APIKeyService interface {
	CreateAPIKey(ctx context.Context, params params.CreateAPIKeyParams) (db.ApiKey, string, string, error)
	GetAPIKey(ctx context.Context, id, workspaceID uuid.UUID) (db.ApiKey, error)
	GetAllAPIKeys(ctx context.Context) ([]db.ApiKey, error)
	UpdateAPIKey(ctx context.Context, params params.UpdateAPIKeyParams) (db.ApiKey, error)
	DeleteAPIKey(ctx context.Context, id, workspaceID uuid.UUID) error
	ListAPIKeys(ctx context.Context, workspaceID uuid.UUID) ([]db.ApiKey, error)
}

// UserService handles user operations
type UserService interface {
	CreateUser(ctx context.Context, params params.CreateUserParams) (*db.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*db.User, error)
	GetUserByEmail(ctx context.Context, email string) (*db.User, error)
	UpdateUser(ctx context.Context, params params.UpdateUserParams) (*db.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	GetUserWithWorkspaceAccess(ctx context.Context, userID, workspaceID uuid.UUID) (*db.User, error)
	GetUserAccount(ctx context.Context, userID uuid.UUID) (*db.GetUserAccountRow, error)
}

// AccountService handles account operations
type AccountService interface {
	CreateAccount(ctx context.Context, params params.CreateAccountParams) (*db.Account, error)
	GetAccount(ctx context.Context, accountID uuid.UUID) (*db.Account, error)
	UpdateAccount(ctx context.Context, params params.UpdateAccountParams) (*db.Account, error)
	DeleteAccount(ctx context.Context, accountID uuid.UUID) error
	ListAccounts(ctx context.Context) ([]db.Account, error)
	ValidateSignInRequest(metadata map[string]interface{}) (string, string, error)
	SignInOrRegisterAccount(ctx context.Context, createParams params.CreateAccountParams, web3authId, email string) (*business.SignInRegisterData, error)
	OnboardAccount(ctx context.Context, params params.OnboardAccountParams) error
	ValidateAccountAccess(ctx context.Context, accountID, workspaceID uuid.UUID) error
}

// CurrencyService handles currency operations
type CurrencyService interface {
	ListActiveCurrencies(ctx context.Context) ([]responses.CurrencyResponse, error)
	GetCurrency(ctx context.Context, code string) (*responses.CurrencyResponse, error)
	GetWorkspaceCurrencySettings(ctx context.Context, workspaceID uuid.UUID) (*business.WorkspaceCurrencySettings, error)
	UpdateWorkspaceCurrencySettings(ctx context.Context, workspaceID uuid.UUID, req *requests.UpdateWorkspaceCurrencyRequest) (*business.WorkspaceCurrencySettings, error)
	FormatAmount(ctx context.Context, amountCents int64, currencyCode string) (string, error)
	FormatAmountWithCode(ctx context.Context, amountCents int64, currencyCode string) (string, error)
	ListWorkspaceSupportedCurrencies(ctx context.Context, workspaceID uuid.UUID) ([]responses.CurrencyResponse, error)
}

// DashboardMetricsService handles dashboard metrics
type DashboardMetricsService interface {
	GetDailyMetrics(ctx context.Context, workspaceID uuid.UUID, date pgtype.Date) (*db.DashboardMetric, error)
	CreateDashboardMetric(ctx context.Context, params db.CreateDashboardMetricParams) (*db.DashboardMetric, error)
	GetMetricsByDateRange(ctx context.Context, workspaceID uuid.UUID, startDate, endDate pgtype.Date) ([]db.DashboardMetric, error)
}

// ErrorRecoveryService handles error recovery operations
type ErrorRecoveryService interface {
	ReplayWebhookEvent(ctx context.Context, req requests.WebhookReplayRequest) (*responses.WebhookReplayResponse, error)
	RecoverSyncSession(ctx context.Context, req requests.SyncRecoveryRequest) (*responses.SyncRecoveryResponse, error)
	GetDLQStats(ctx context.Context, workspaceID, providerName string, since time.Time) (*responses.DLQProcessingStats, error)
}

// SubscriptionEventService handles subscription events
type SubscriptionEventService interface {
	CreateSubscriptionEvent(ctx context.Context, params params.CreateSubscriptionEventParams) (*db.SubscriptionEvent, error)
	GetSubscriptionEvent(ctx context.Context, eventID, workspaceID uuid.UUID) (*db.SubscriptionEvent, error)
	GetSubscriptionEventByTxHash(ctx context.Context, txHash string) (*db.SubscriptionEvent, error)
	ListSubscriptionEvents(ctx context.Context, params params.ListSubscriptionEventsParams) (*responses.ListSubscriptionEventsResult, error)
	ListSubscriptionEventsBySubscription(ctx context.Context, subscriptionID, workspaceID uuid.UUID) ([]db.SubscriptionEvent, error)
}

// AnalyticsService handles analytics operations
type AnalyticsService interface {
	GetDashboardSummary(ctx context.Context, workspaceID uuid.UUID, currency string) (*business.DashboardSummary, error)
	GetRevenueChart(ctx context.Context, workspaceID uuid.UUID, period string, days int, currency string) (*business.ChartData, error)
	GetCustomerChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, days int, currency string) (*business.ChartData, error)
	GetPaymentMetrics(ctx context.Context, workspaceID uuid.UUID, days int, currency string) (*business.PaymentMetrics, error)
	GetNetworkBreakdown(ctx context.Context, workspaceID uuid.UUID, date time.Time, currency string) (*business.NetworkBreakdown, error)
	GetSubscriptionChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, days int, currency string) (*business.ChartData, error)
	GetMRRChart(ctx context.Context, workspaceID uuid.UUID, metric, period string, months int, currency string) (*business.ChartData, error)
	GetGasFeePieChart(ctx context.Context, workspaceID uuid.UUID, days int, currency string) (*business.PieChartData, error)
	GetHourlyMetrics(ctx context.Context, workspaceID uuid.UUID, currency string) (*business.HourlyMetrics, error)
	TriggerMetricsRefresh(ctx context.Context, workspaceID uuid.UUID, date time.Time) error
}

// BlockchainSyncService handles blockchain synchronization
type BlockchainSyncService interface {
	SyncTransactions(ctx context.Context, workspaceID uuid.UUID) error
	GetSyncStatus(ctx context.Context, workspaceID uuid.UUID) (string, error)
	ResyncFailedTransactions(ctx context.Context, workspaceID uuid.UUID) error
}

// MetricsScheduler handles scheduled metrics updates
type MetricsScheduler interface {
	ScheduleMetricsUpdate(ctx context.Context, workspaceID uuid.UUID, interval time.Duration) error
	RunMetricsUpdate(ctx context.Context, workspaceID uuid.UUID) error
}

// PaymentFailureDetector detects payment failures
type PaymentFailureDetector interface {
	ProcessFailedPaymentWebhook(ctx context.Context, workspaceID uuid.UUID, subscriptionID uuid.UUID, failureData map[string]interface{}) error
	DetectAndCreateCampaigns(ctx context.Context, lookbackMinutes int) (*responses.DetectionResult, error)
}

// CommonServicesInterface defines the interface for CommonServices
// This allows for easier testing and mocking of the CommonServices struct
type CommonServicesInterface interface {
	// Database methods
	GetDB() db.Querier
	GetDBPool() (*pgxpool.Pool, error)
	WithTx(tx pgx.Tx) *db.Queries
	BeginTx(ctx context.Context) (pgx.Tx, *db.Queries, error)
	RunInTransaction(ctx context.Context, fn func(qtx *db.Queries) error) error
	RunInTransactionWithRetry(ctx context.Context, maxRetries int, fn func(qtx *db.Queries) error) error

	// Service getters
	GetLogger() *zap.Logger
	GetAPIKeyService() APIKeyService
	GetTaxService() TaxService
	GetDiscountService() DiscountService
	GetGasSponsorshipService() GasSponsorshipService
	GetCurrencyService() CurrencyService
	GetExchangeRateService() ExchangeRateService

	// Configuration getters
	GetCypheraSmartWalletAddress() string
}
