package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ps "github.com/cyphera/cyphera-api/libs/go/client/payment_sync"
	"github.com/cyphera/cyphera-api/libs/go/db"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"
)

// Ensure StripeService implements PaymentSyncService interface
var _ ps.PaymentSyncService = (*StripeService)(nil)

// StripeService implements the PaymentSyncService for Stripe.
// Method implementations for specific resources (Customer, Product, etc.) are in separate files
// within this package (e.g., customer.go, product.go).
type StripeService struct {
	client        *stripe.Client
	webhookSecret string
	logger        *zap.Logger
	db            *db.Queries
}

// NewStripeService creates a new instance of StripeService.
// It does not yet configure the API key, that happens in Configure.
func NewStripeService(logger *zap.Logger, dbQueries *db.Queries) *StripeService {
	return &StripeService{
		logger: logger,
		db:     dbQueries,
	}
}

// GetServiceName returns the name of the service.
func (s *StripeService) GetServiceName() string {
	return "stripe"
}

// Configure initializes the Stripe service with API key and webhook secret.
func (s *StripeService) Configure(ctx context.Context, config map[string]string) error {
	apiKey, ok := config["api_key"]
	if !ok || apiKey == "" {
		return fmt.Errorf("stripe API key not provided in configuration")
	}

	webhookSecret, ok := config["webhook_secret"]
	if !ok || webhookSecret == "" {
		// For some operations, webhook secret might not be strictly necessary for the client part,
		// but it's good to expect it for a full sync service.
		// Depending on strictness, you might return an error or just log a warning.
		// For now, let's make it required for completeness of configuration.
		return fmt.Errorf("stripe webhook secret not provided in configuration")
	}

	s.client = stripe.NewClient(apiKey, nil)
	s.webhookSecret = webhookSecret

	return nil
}

// CheckConnection verifies that the service can connect to Stripe.
// For Stripe, this could involve making a simple, non-mutating API call, like listing account details.
func (s *StripeService) CheckConnection(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("stripe client not configured. Call Configure first")
	}

	_, err := s.client.V1Accounts.Retrieve(ctx, &stripe.AccountRetrieveParams{})
	if err != nil {
		return fmt.Errorf("failed to connect to Stripe: %w", err)
	}
	return nil
}

// --- External Account / Payment Method Management ---
// For this service, with Bridge.xyz handling primary off-ramping, Stripe's role in managing
// external bank accounts for user payouts is diminished.
// These methods might manage Stripe Customer Bank Accounts if Stripe is used for payouts *from Stripe balance*,
// but not for user-provided accounts for Bridge off-ramps.

func (s *StripeService) CreateExternalAccount(ctx context.Context, customerExternalID string, accountData ps.ExternalAccount, setAsDefault bool) (ps.ExternalAccount, error) {
	// accountData.Provider should ideally be checked. If not 'stripe', this method might be incorrect.
	// If Stripe is used, this would map to creating a BankAccount token and attaching to a Customer, or Connect external accounts.
	return ps.ExternalAccount{}, fmt.Errorf("CreateExternalAccount via Stripe not fully implemented for the current off-ramp model; Bridge.xyz is primary")
}

func (s *StripeService) GetExternalAccount(ctx context.Context, customerExternalID string, externalAccountID string) (ps.ExternalAccount, error) {
	// This would involve fetching a BankAccount or Connect external account from Stripe.
	return ps.ExternalAccount{}, fmt.Errorf("GetExternalAccount via Stripe not fully implemented for the current off-ramp model")
}

func (s *StripeService) UpdateExternalAccount(ctx context.Context, customerExternalID string, externalAccountID string, accountData ps.ExternalAccount) (ps.ExternalAccount, error) {
	return ps.ExternalAccount{}, fmt.Errorf("UpdateExternalAccount via Stripe not supported or not fully implemented")
}

func (s *StripeService) DeleteExternalAccount(ctx context.Context, customerExternalID string, externalAccountID string) error {
	// This would involve detaching/deleting a BankAccount or Connect external account.
	return fmt.Errorf("DeleteExternalAccount via Stripe not fully implemented for the current off-ramp model")
}

func (s *StripeService) ListExternalAccounts(ctx context.Context, customerExternalID string, params ps.ListParams) ([]ps.ExternalAccount, string, error) {
	// This would list BankAccounts for a Stripe Customer or Connect external accounts.
	// Note: customerExternalID here is ps.Customer.ExternalID, map to Stripe Customer ID.
	return nil, "", fmt.Errorf("ListExternalAccounts via Stripe not fully implemented for the current off-ramp model")
}

func (s *StripeService) SetDefaultExternalAccount(ctx context.Context, customerExternalID string, externalAccountID string) error {
	// This would set a default BankAccount on a Stripe Customer or Connect external account.
	// Note: customerExternalID here is ps.Customer.ExternalID.
	return fmt.Errorf("SetDefaultExternalAccount via Stripe not fully implemented for the current off-ramp model")
}

// Note: All other interface methods (webhook handling, invoice management, transaction management,
// customer management, product management, price management, subscription management) are implemented
// in their respective specialized files: webhook.go, invoice.go, transaction.go, customer.go, product.go,
// price.go, subscription.go

// StartInitialSync initiates a complete initial sync session
// @godoc StartInitialSync starts an initial data synchronization from Stripe to the database
func (s *StripeService) StartInitialSync(ctx context.Context, workspaceID string, config ps.InitialSyncConfig) (ps.SyncSession, error) {
	s.logger.Info("Starting initial sync",
		zap.String("workspace_id", workspaceID),
		zap.Any("config", config))

	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return ps.SyncSession{}, fmt.Errorf("invalid workspace ID: %w", err)
	}

	// Use defaults if not provided
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if len(config.EntityTypes) == 0 {
		config.EntityTypes = []string{"customers", "products", "prices", "subscriptions"}
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 2 // seconds
	}

	// Marshal config to JSON
	configJSON, err := json.Marshal(map[string]interface{}{
		"batch_size":     config.BatchSize,
		"full_sync":      config.FullSync,
		"starting_after": config.StartingAfter,
		"ending_before":  config.EndingBefore,
		"max_retries":    config.MaxRetries,
		"retry_delay":    config.RetryDelay,
	})
	if err != nil {
		return ps.SyncSession{}, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create sync session
	session, err := s.db.CreateSyncSession(ctx, db.CreateSyncSessionParams{
		WorkspaceID:  wsID,
		ProviderName: s.GetServiceName(),
		SessionType:  "initial_sync",
		Status:       "pending",
		EntityTypes:  config.EntityTypes,
		Config:       configJSON,
	})
	if err != nil {
		return ps.SyncSession{}, fmt.Errorf("failed to create sync session: %w", err)
	}

	// Update status to running
	updatedSession, err := s.db.UpdateSyncSessionStatus(ctx, db.UpdateSyncSessionStatusParams{
		ID:     session.ID,
		Status: "running",
	})
	if err != nil {
		s.logger.Error("Failed to update session status to running", zap.Error(err))
		return s.mapDBSessionToPSSession(session), err
	}

	// Run the sync in the background
	go func() {
		syncCtx := context.Background() // Use background context for async operation
		if err := s.runSyncProcess(syncCtx, &updatedSession, config); err != nil {
			s.logger.Error("Initial sync process failed",
				zap.String("session_id", updatedSession.ID.String()),
				zap.Error(err))

			// Marshal error summary
			errorJSON, _ := json.Marshal(map[string]interface{}{
				"error":     err.Error(),
				"failed_at": time.Now(),
			})

			// Update session with error
			if _, updateErr := s.db.UpdateSyncSessionError(syncCtx, db.UpdateSyncSessionErrorParams{
				ID:           updatedSession.ID,
				ErrorSummary: errorJSON,
			}); updateErr != nil {
				s.logger.Error("Failed to update sync session error", zap.Error(updateErr))
			}
		}
	}()

	return s.mapDBSessionToPSSession(updatedSession), nil
}

// Helper method to convert database session to interface session
func (s *StripeService) mapDBSessionToPSSession(dbSession db.PaymentSyncSession) ps.SyncSession {
	session := ps.SyncSession{
		ID:          dbSession.ID.String(),
		WorkspaceID: dbSession.WorkspaceID.String(),
		Provider:    dbSession.ProviderName,
		SessionType: dbSession.SessionType,
		Status:      dbSession.Status,
		EntityTypes: dbSession.EntityTypes,
		CreatedAt:   dbSession.CreatedAt.Time.Unix(),
		UpdatedAt:   dbSession.UpdatedAt.Time.Unix(),
	}

	// Parse JSON fields
	if len(dbSession.Config) > 0 {
		_ = json.Unmarshal(dbSession.Config, &session.Config)
	}
	if len(dbSession.Progress) > 0 {
		_ = json.Unmarshal(dbSession.Progress, &session.Progress)
	}
	if len(dbSession.ErrorSummary) > 0 {
		_ = json.Unmarshal(dbSession.ErrorSummary, &session.ErrorSummary)
	}

	// Handle nullable timestamps
	if dbSession.StartedAt.Valid {
		startedAt := dbSession.StartedAt.Time.Unix()
		session.StartedAt = &startedAt
	}
	if dbSession.CompletedAt.Valid {
		completedAt := dbSession.CompletedAt.Time.Unix()
		session.CompletedAt = &completedAt
	}

	return session
}
