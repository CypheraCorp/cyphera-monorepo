package handlers

import (
	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/client/payment_sync"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/interfaces"
	"github.com/cyphera/cyphera-api/libs/go/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// HandlerFactory creates handlers with proper dependency injection
type HandlerFactory struct {
	// Database
	db db.Querier

	// Common services
	commonServices *CommonServices

	// Services - these should be interfaces in production
	subscriptionManagementService interfaces.SubscriptionManagementService
	dunningService                interfaces.DunningService
	dunningRetryEngine            interfaces.DunningRetryEngine
	paymentLinkService            interfaces.PaymentLinkService
	paymentService                interfaces.PaymentService
	emailService                  interfaces.EmailService
	invoiceService                interfaces.InvoiceService
	gasSponsorshipService         interfaces.GasSponsorshipService
	productService                interfaces.ProductService
	subscriptionService           interfaces.SubscriptionService
	customerService               interfaces.CustomerService
	workspaceService              interfaces.WorkspaceService
	accountService                interfaces.AccountService
	userService                   interfaces.UserService
	walletService                 interfaces.WalletService
	tokenService                  interfaces.TokenService
	networkService                interfaces.NetworkService
	analyticsService              interfaces.AnalyticsService
	blockchainService             interfaces.BlockchainService
	errorRecoveryService          interfaces.ErrorRecoveryService
	subscriptionEventService      interfaces.SubscriptionEventService
	paymentFailureMonitor         interfaces.PaymentFailureMonitor
	paymentFailureDetector        interfaces.PaymentFailureDetector
	APIKeyService                 interfaces.APIKeyService

	// External clients
	cmcClient *coinmarketcap.Client

	// Configuration
	cypheraSmartWalletAddress string
	cmcAPIKey                 string
	paymentLinkBaseURL        string

	// Logger
	logger *zap.Logger
}

// HandlerFactoryConfig contains all configuration for the handler factory
type HandlerFactoryConfig struct {
	// Database
	DB db.Querier

	// Services - pass concrete implementations that satisfy the interfaces
	SubscriptionManagementService interfaces.SubscriptionManagementService
	DunningService                interfaces.DunningService
	DunningRetryEngine            interfaces.DunningRetryEngine
	PaymentLinkService            interfaces.PaymentLinkService
	PaymentService                interfaces.PaymentService
	EmailService                  interfaces.EmailService
	InvoiceService                interfaces.InvoiceService
	GasSponsorshipService         interfaces.GasSponsorshipService
	ProductService                interfaces.ProductService
	SubscriptionService           interfaces.SubscriptionService
	CustomerService               interfaces.CustomerService
	WorkspaceService              interfaces.WorkspaceService
	AccountService                interfaces.AccountService
	UserService                   interfaces.UserService
	WalletService                 interfaces.WalletService
	TokenService                  interfaces.TokenService
	NetworkService                interfaces.NetworkService
	AnalyticsService              interfaces.AnalyticsService
	BlockchainService             interfaces.BlockchainService
	ErrorRecoveryService          interfaces.ErrorRecoveryService
	SubscriptionEventService      interfaces.SubscriptionEventService
	PaymentFailureMonitor         interfaces.PaymentFailureMonitor
	PaymentFailureDetector        interfaces.PaymentFailureDetector
	APIKeyService                 interfaces.APIKeyService
	TaxService                    interfaces.TaxService
	DiscountService               interfaces.DiscountService
	CurrencyService               interfaces.CurrencyService
	ExchangeRateService           interfaces.ExchangeRateService

	// External clients
	CMCClient *coinmarketcap.Client

	// Configuration
	CypheraSmartWalletAddress string
	CMCAPIKey                 string
	PaymentLinkBaseURL        string

	// Logger
	Logger *zap.Logger
}

// NewHandlerFactory creates a new handler factory with all dependencies
func NewHandlerFactory(config HandlerFactoryConfig) *HandlerFactory {
	if config.Logger == nil {
		config.Logger = zap.L()
	}

	// Create common services with interfaces
	commonServices := NewCommonServices(CommonServicesConfig{
		DB:                        config.DB,
		CypheraSmartWalletAddress: config.CypheraSmartWalletAddress,
		CMCClient:                 config.CMCClient,
		CMCAPIKey:                 config.CMCAPIKey,
		APIKeyService:             config.APIKeyService,
		Logger:                    config.Logger,
		TaxService:                config.TaxService,
		DiscountService:           config.DiscountService,
		GasSponsorshipService:     config.GasSponsorshipService,
		CurrencyService:           config.CurrencyService,
		ExchangeRateService:       config.ExchangeRateService,
	})

	return &HandlerFactory{
		db:                            config.DB,
		commonServices:                commonServices,
		subscriptionManagementService: config.SubscriptionManagementService,
		dunningService:                config.DunningService,
		dunningRetryEngine:            config.DunningRetryEngine,
		paymentLinkService:            config.PaymentLinkService,
		paymentService:                config.PaymentService,
		emailService:                  config.EmailService,
		invoiceService:                config.InvoiceService,
		gasSponsorshipService:         config.GasSponsorshipService,
		productService:                config.ProductService,
		subscriptionService:           config.SubscriptionService,
		customerService:               config.CustomerService,
		workspaceService:              config.WorkspaceService,
		accountService:                config.AccountService,
		userService:                   config.UserService,
		walletService:                 config.WalletService,
		tokenService:                  config.TokenService,
		networkService:                config.NetworkService,
		analyticsService:              config.AnalyticsService,
		blockchainService:             config.BlockchainService,
		errorRecoveryService:          config.ErrorRecoveryService,
		subscriptionEventService:      config.SubscriptionEventService,
		paymentFailureMonitor:         config.PaymentFailureMonitor,
		paymentFailureDetector:        config.PaymentFailureDetector,
		APIKeyService:                 config.APIKeyService,
		cmcClient:                     config.CMCClient,
		cypheraSmartWalletAddress:     config.CypheraSmartWalletAddress,
		cmcAPIKey:                     config.CMCAPIKey,
		paymentLinkBaseURL:            config.PaymentLinkBaseURL,
		logger:                        config.Logger,
	}
}

// CreateDefaultFactory creates a factory with concrete implementations
// This is useful for backward compatibility
func CreateDefaultFactory(
	db *db.Queries,
	dbPool *pgxpool.Pool,
	cypheraSmartWalletAddress string,
	cmcClient *coinmarketcap.Client,
	cmcAPIKey string,
	resendAPIKey string,
	fromEmail string,
	fromName string,
	baseURL string,
	rpcAPIKey string,
	delegationClient *dsClient.DelegationClient,
	paymentSyncClient *payment_sync.PaymentSyncClient,
) *HandlerFactory {
	logger := zap.L()

	// Create all concrete services
	emailService := services.NewEmailService(resendAPIKey, fromEmail, fromName, logger)
	paymentService := services.NewPaymentService(db, cmcAPIKey)
	currencyService := services.NewCurrencyService(db)
	exchangeRateService := services.NewExchangeRateService(db, cmcAPIKey)
	taxService := services.NewTaxService(db)
	discountService := services.NewDiscountService(db)
	gasSponsorshipService := services.NewGasSponsorshipService(db)

	// Create services that depend on other services
	subscriptionManagementService := services.NewSubscriptionManagementService(db, paymentService, emailService)
	dunningService := services.NewDunningService(db, logger)
	dunningRetryEngine := services.NewDunningRetryEngine(db, logger, dunningService, emailService, delegationClient)

	paymentLinkService := services.NewPaymentLinkService(db, logger, baseURL)
	invoiceService := services.NewInvoiceService(db, logger, taxService, discountService, gasSponsorshipService, currencyService, exchangeRateService)
	productService := services.NewProductService(db)
	customerService := services.NewCustomerService(db)
	subscriptionService := services.NewSubscriptionService(db, delegationClient, paymentService, customerService)
	workspaceService := services.NewWorkspaceService(db)
	accountService := services.NewAccountService(db)
	userService := services.NewUserService(db)
	walletService := services.NewWalletService(db)
	tokenService := services.NewTokenService(db, cmcClient)
	networkService := services.NewNetworkService(db)
	analyticsService := services.NewAnalyticsService(db, dbPool)
	blockchainService := services.NewBlockchainService(db, rpcAPIKey)
	errorRecoveryService := services.NewErrorRecoveryService(db, logger, paymentSyncClient)
	subscriptionEventService := services.NewSubscriptionEventService(db)
	paymentFailureMonitor := services.NewPaymentFailureMonitor(db, logger, dunningService)
	paymentFailureDetector := services.NewPaymentFailureDetector(db, logger, dunningService)
	apiKeyService := services.NewAPIKeyService(db)

	// Also update the factory to include DBPool in the config for CommonServices
	return &HandlerFactory{
		db: db,
		commonServices: NewCommonServices(CommonServicesConfig{
			DB:                        db,
			DBPool:                    dbPool, // Add the missing dbPool here
			CypheraSmartWalletAddress: cypheraSmartWalletAddress,
			CMCClient:                 cmcClient,
			CMCAPIKey:                 cmcAPIKey,
			APIKeyService:             apiKeyService,
			Logger:                    logger,
			TaxService:                taxService,
			DiscountService:           discountService,
			GasSponsorshipService:     gasSponsorshipService,
			CurrencyService:           currencyService,
			ExchangeRateService:       exchangeRateService,
		}),
		subscriptionManagementService: subscriptionManagementService,
		dunningService:                dunningService,
		dunningRetryEngine:            dunningRetryEngine,
		paymentLinkService:            paymentLinkService,
		paymentService:                paymentService,
		emailService:                  emailService,
		invoiceService:                invoiceService,
		gasSponsorshipService:         gasSponsorshipService,
		productService:                productService,
		subscriptionService:           subscriptionService,
		customerService:               customerService,
		workspaceService:              workspaceService,
		accountService:                accountService,
		userService:                   userService,
		walletService:                 walletService,
		tokenService:                  tokenService,
		networkService:                networkService,
		analyticsService:              analyticsService,
		blockchainService:             blockchainService,
		errorRecoveryService:          errorRecoveryService,
		subscriptionEventService:      subscriptionEventService,
		paymentFailureMonitor:         paymentFailureMonitor,
		paymentFailureDetector:        paymentFailureDetector,
		APIKeyService:                 apiKeyService,
		cmcClient:                     cmcClient,
		cypheraSmartWalletAddress:     cypheraSmartWalletAddress,
		cmcAPIKey:                     cmcAPIKey,
		paymentLinkBaseURL:            "https://pay.cyphera.com",
		logger:                        logger,
	}
}

// Handler creation methods

// NewSubscriptionManagementHandler creates a new subscription management handler
func (f *HandlerFactory) NewSubscriptionManagementHandler() *SubscriptionManagementHandler {
	return NewSubscriptionManagementHandler(
		f.subscriptionManagementService,
		f.logger,
	)
}

// NewDunningHandler creates a new dunning handler
func (f *HandlerFactory) NewDunningHandler() *DunningHandler {
	return NewDunningHandler(
		f.commonServices,
		f.dunningService,
		f.dunningRetryEngine,
	)
}

// NewErrorRecoveryHandler creates a new error recovery handler
func (f *HandlerFactory) NewErrorRecoveryHandler() *ErrorRecoveryHandlers {
	return NewErrorRecoveryHandlers(
		f.errorRecoveryService,
		f.logger,
	)
}

// NewPaymentLinkHandler creates a new payment link handler
func (f *HandlerFactory) NewPaymentLinkHandler() *PaymentLinkHandler {
	return NewPaymentLinkHandler(
		f.commonServices,
		f.paymentLinkService,
	)
}

// NewInvoiceHandler creates a new invoice handler
func (f *HandlerFactory) NewInvoiceHandler() *InvoiceHandler {
	return NewInvoiceHandler(
		f.commonServices,
		f.invoiceService,
		f.paymentLinkService,
		f.logger,
	)
}

// NewGasSponsorshipHandler creates a new gas sponsorship handler
func (f *HandlerFactory) NewGasSponsorshipHandler() *GasSponsorshipHandler {
	return NewGasSponsorshipHandler(
		f.commonServices,
		f.gasSponsorshipService,
		f.logger,
	)
}

// NewProductHandler creates a new product handler
func (f *HandlerFactory) NewProductHandler(delegationClient *dsClient.DelegationClient) *ProductHandler {
	return NewProductHandler(
		f.commonServices,
		delegationClient,
		f.productService,
		f.subscriptionService,
		f.customerService,
		f.logger,
	)
}

// NewSubscriptionHandler creates a new subscription handler
func (f *HandlerFactory) NewSubscriptionHandler(delegationClient *dsClient.DelegationClient) *SubscriptionHandler {
	return NewSubscriptionHandler(
		f.commonServices,
		delegationClient,
		f.subscriptionService,
		f.paymentService,
		f.logger,
	)
}

// NewAPIKeyHandler creates a new API key handler
func (f *HandlerFactory) NewAPIKeyHandler() *APIKeyHandler {
	return NewAPIKeyHandler(
		f.commonServices,
		f.logger,
	)
}

// NewAccountHandler creates a new account handler
func (f *HandlerFactory) NewAccountHandler() *AccountHandler {
	return NewAccountHandler(
		f.commonServices,
		f.accountService,
		f.walletService,
	)
}

// NewAnalyticsHandler creates a new analytics handler
func (f *HandlerFactory) NewAnalyticsHandler() *AnalyticsHandler {
	return NewAnalyticsHandler(
		f.commonServices,
		f.analyticsService,
		f.logger,
	)
}

// NewCurrencyHandler creates a new currency handler
func (f *HandlerFactory) NewCurrencyHandler() *CurrencyHandler {
	return NewCurrencyHandler(
		f.commonServices,
		f.commonServices.CurrencyService,
	)
}

// NewCustomerHandler creates a new customer handler
func (f *HandlerFactory) NewCustomerHandler() *CustomerHandler {
	return NewCustomerHandler(
		f.commonServices,
		f.customerService,
	)
}

// NewNetworkHandler creates a new network handler
func (f *HandlerFactory) NewNetworkHandler() *NetworkHandler {
	return NewNetworkHandler(
		f.commonServices,
		f.networkService,
	)
}

// NewPaymentFailureWebhookHandler creates a new payment failure webhook handler
func (f *HandlerFactory) NewPaymentFailureWebhookHandler() *PaymentFailureWebhookHandler {
	return NewPaymentFailureWebhookHandler(
		f.commonServices,
		f.paymentFailureDetector,
	)
}

// NewPaymentPageHandler creates a new payment page handler
func (f *HandlerFactory) NewPaymentPageHandler() *PaymentPageHandler {
	return NewPaymentPageHandler(
		f.commonServices,
		f.paymentLinkService,
	)
}

// NewSubscriptionEventHandler creates a new subscription event handler
func (f *HandlerFactory) NewSubscriptionEventHandler() *SubscriptionEventHandler {
	return NewSubscriptionEventHandler(
		f.commonServices,
		f.subscriptionEventService,
	)
}

// NewTokenHandler creates a new token handler
func (f *HandlerFactory) NewTokenHandler() *TokenHandler {
	return NewTokenHandler(
		f.commonServices,
		f.tokenService,
	)
}

// NewUserHandler creates a new user handler
func (f *HandlerFactory) NewUserHandler() *UserHandler {
	return NewUserHandler(
		f.commonServices,
		f.userService,
	)
}

// NewWorkspaceHandler creates a new workspace handler
func (f *HandlerFactory) NewWorkspaceHandler() *WorkspaceHandler {
	return NewWorkspaceHandler(
		f.commonServices,
		f.workspaceService,
	)
}

// NewWalletHandler creates a new wallet handler
func (f *HandlerFactory) NewWalletHandler() *WalletHandler {
	return NewWalletHandler(
		f.commonServices,
		f.walletService,
	)
}

// CreateDunningHandler creates a dunning handler
func (f *HandlerFactory) CreateDunningHandler() *DunningHandler {
	return NewDunningHandler(f.commonServices, f.dunningService, f.dunningRetryEngine)
}

// CreateErrorRecoveryHandler creates an error recovery handler
func (f *HandlerFactory) CreateErrorRecoveryHandler() *ErrorRecoveryHandlers {
	return NewErrorRecoveryHandlers(f.errorRecoveryService, f.logger)
}

// CreatePaymentFailureWebhookHandler creates a payment failure webhook handler
func (f *HandlerFactory) CreatePaymentFailureWebhookHandler() *PaymentFailureWebhookHandler {
	return NewPaymentFailureWebhookHandler(f.commonServices, f.paymentFailureDetector)
}

// GetCommonServices returns the common services instance
func (f *HandlerFactory) GetCommonServices() *CommonServices {
	return f.commonServices
}

// GetDB returns the database querier
func (f *HandlerFactory) GetDB() db.Querier {
	return f.db
}

// GetLogger returns the logger
func (f *HandlerFactory) GetLogger() *zap.Logger {
	return f.logger
}

// CreateRedemptionProcessor creates a redemption processor
func (f *HandlerFactory) CreateRedemptionProcessor(delegationClient *dsClient.DelegationClient, workerCount int, bufferSize int) *services.RedemptionProcessor {
	return services.NewRedemptionProcessor(f.db, delegationClient, f.paymentService, workerCount, bufferSize)
}
