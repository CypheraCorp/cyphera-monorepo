package services

import (
	"context"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/api/requests"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/cyphera/cyphera-api/libs/go/types/business"
	"github.com/google/uuid"
)

// Local interfaces to avoid circular dependency with interfaces package

// IProrationCalculator handles proration calculations for subscription changes
type IProrationCalculator interface {
	CalculateUpgradeProration(currentPeriodStart, currentPeriodEnd time.Time, oldAmountCents, newAmountCents int64, changeDate time.Time) *business.ProrationResult
	ScheduleDowngrade(currentPeriodEnd time.Time, changeType string) *business.ScheduleChangeResult
	CalculatePauseCredit(currentPeriodStart, currentPeriodEnd time.Time, amountCents int64, pauseDate time.Time) *business.ProrationResult
	FormatProrationExplanation(result *business.ProrationResult) string
	AddBillingPeriod(start time.Time, intervalType string, intervalCount int) time.Time
	DaysBetween(start, end time.Time) int
	CalculateTrialEndDate(start time.Time, trialDays int) time.Time
	IsInTrial(trialEnd *time.Time) bool
	GetDailyRate(amountCents int64, periodStart, periodEnd time.Time) float64
}

// IPaymentService handles payment processing operations
type IPaymentService interface {
	CreatePaymentFromSubscriptionEvent(ctx context.Context, params params.CreatePaymentFromSubscriptionEventParams) (*db.Payment, error)
	CreateComprehensivePayment(ctx context.Context, params params.CreateComprehensivePaymentParams) (*db.Payment, error)
	GetPayment(ctx context.Context, params params.GetPaymentParams) (*db.Payment, error)
	GetPaymentByTransactionHash(ctx context.Context, txHash string) (*db.Payment, error)
	ListPayments(ctx context.Context, params params.ListPaymentsParams) ([]db.Payment, error)
	UpdatePaymentStatus(ctx context.Context, params params.UpdatePaymentStatusParams) (*db.Payment, error)
	GetPaymentMetrics(ctx context.Context, workspaceID uuid.UUID, startTime, endTime time.Time, currency string) (*db.GetPaymentMetricsRow, error)
	CreateManualPayment(ctx context.Context, params params.CreateManualPaymentParams) (*db.Payment, error)
}

// IEmailService handles email sending operations
type IEmailService interface {
	SendTransactionalEmail(ctx context.Context, params params.TransactionalEmailParams) error
	SendBatchEmails(ctx context.Context, requests []requests.BatchEmailRequest) ([]responses.BatchEmailResult, error)
	SendDunningEmail(ctx context.Context, template *db.DunningEmailTemplate, data map[string]business.EmailData, toEmail string) error
}
