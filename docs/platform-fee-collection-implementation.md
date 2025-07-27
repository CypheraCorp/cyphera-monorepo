# Platform Fee Collection Implementation Plan

## Overview

This document outlines the implementation of Cyphera's platform fee collection system, which enables the platform to automatically collect transaction fees while processing customer payments to merchants. The system will handle batch transactions, gas fee recovery, and transparent fee splitting.

## Business Model

### Fee Structure
- **Platform Fee**: 1% of transaction amount + $0.25 flat fee
- **Gas Fee Recovery**: Actual gas costs deducted from merchant payment
- **Minimum Transaction**: $1.00 (to ensure fees don't exceed payment)

### Fee Calculation Example
For a $100 transaction:
```
Transaction Amount: $100.00
Platform Percentage (1%): $1.00
Platform Flat Fee: $0.25
Gas Fee Estimate: $0.75 (example)
-----------------------------------
Total Platform Fee: $2.00
Merchant Receives: $98.00
```

## Database Schema Updates

### 1. Fee Tier Profiles Table
```sql
CREATE TABLE fee_tier_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    
    -- Tier level (lower is better for merchant)
    tier_level INTEGER NOT NULL DEFAULT 1, -- 1=Premium, 2=Standard, 3=Basic
    
    -- Fee structure
    percentage_fee DECIMAL(5,4) NOT NULL, -- e.g., 0.0050 = 0.50%
    flat_fee_cents INTEGER NOT NULL, -- e.g., 15 = $0.15
    
    -- Volume discounts
    monthly_volume_threshold_cents BIGINT, -- Apply this tier if volume exceeds
    annual_volume_threshold_cents BIGINT,
    
    -- Gas fee handling
    gas_fee_coverage_percent DECIMAL(5,2) DEFAULT 0, -- 0-100% platform covers
    gas_fee_cap_cents INTEGER, -- Max gas fee charged to merchant
    
    -- Requirements
    minimum_monthly_revenue_cents BIGINT DEFAULT 0,
    minimum_transaction_cents INTEGER DEFAULT 100,
    maximum_transaction_cents BIGINT,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false, -- Default tier for new merchants
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for default tier lookup
CREATE UNIQUE INDEX idx_fee_tier_default ON fee_tier_profiles(is_default) WHERE is_default = true;
CREATE INDEX idx_fee_tier_active ON fee_tier_profiles(is_active, tier_level);

-- Insert tier profiles
INSERT INTO fee_tier_profiles (name, description, tier_level, percentage_fee, flat_fee_cents, gas_fee_coverage_percent, is_default) VALUES
('Basic', 'Standard pricing for new merchants', 3, 0.0100, 25, 0, true), -- 1% + $0.25
('Growth', 'For growing businesses', 2, 0.0075, 20, 25, false), -- 0.75% + $0.20, platform covers 25% gas
('Enterprise', 'High volume merchants', 1, 0.0050, 15, 50, false), -- 0.5% + $0.15, platform covers 50% gas
('Custom', 'Negotiated pricing', 1, 0.0100, 25, 0, false); -- Placeholder for custom deals
```

### 2. Workspace Fee Settings Table
```sql
CREATE TABLE workspace_fee_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL UNIQUE REFERENCES workspaces(id),
    
    -- Assigned tier
    fee_tier_profile_id UUID NOT NULL REFERENCES fee_tier_profiles(id),
    
    -- Custom overrides (NULL means use tier defaults)
    custom_percentage_fee DECIMAL(5,4),
    custom_flat_fee_cents INTEGER,
    custom_gas_coverage_percent DECIMAL(5,2),
    
    -- Volume tracking
    current_month_volume_cents BIGINT DEFAULT 0,
    current_year_volume_cents BIGINT DEFAULT 0,
    last_volume_check_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Tier upgrade eligibility
    eligible_for_tier_id UUID REFERENCES fee_tier_profiles(id),
    tier_evaluation_date TIMESTAMPTZ,
    
    -- Special conditions
    promotional_rate_ends_at TIMESTAMPTZ,
    legacy_rate_protected BOOLEAN DEFAULT false,
    
    -- Platform wallet override (for special routing)
    custom_platform_wallet_address VARCHAR(255),
    
    -- Metadata
    notes TEXT,
    approved_by VARCHAR(255),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_workspace_fee_settings_workspace ON workspace_fee_settings(workspace_id);
CREATE INDEX idx_workspace_fee_volume_check ON workspace_fee_settings(last_volume_check_at);
```

### 3. Fee Calculation Audit Table
```sql
CREATE TABLE fee_calculation_audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    payment_id UUID REFERENCES payments(id),
    
    -- Applied rates
    applied_tier_id UUID REFERENCES fee_tier_profiles(id),
    applied_percentage_fee DECIMAL(5,4) NOT NULL,
    applied_flat_fee_cents INTEGER NOT NULL,
    applied_gas_coverage_percent DECIMAL(5,2) NOT NULL,
    
    -- Calculation details
    transaction_amount_cents BIGINT NOT NULL,
    calculated_percentage_fee_cents BIGINT NOT NULL,
    calculated_flat_fee_cents BIGINT NOT NULL,
    calculated_gas_fee_cents BIGINT NOT NULL,
    gas_fee_covered_by_platform_cents BIGINT NOT NULL,
    
    -- Final amounts
    total_platform_fee_cents BIGINT NOT NULL,
    merchant_charged_fee_cents BIGINT NOT NULL,
    merchant_receives_cents BIGINT NOT NULL,
    
    -- Why this rate was applied
    rate_reason VARCHAR(50) NOT NULL, -- 'tier_default', 'custom_override', 'promotional', 'volume_discount'
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_fee_audits_workspace ON fee_calculation_audits(workspace_id, created_at DESC);
CREATE INDEX idx_fee_audits_payment ON fee_calculation_audits(payment_id);
```

### 4. Platform Configuration Table
```sql
CREATE TABLE platform_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Platform wallets by network
    ethereum_wallet_address VARCHAR(255) NOT NULL,
    polygon_wallet_address VARCHAR(255) NOT NULL,
    base_wallet_address VARCHAR(255) NOT NULL,
    
    -- Global settings
    global_minimum_transaction_cents INTEGER DEFAULT 100,
    global_maximum_transaction_cents BIGINT,
    
    -- Gas settings
    gas_estimation_multiplier DECIMAL(3,2) DEFAULT 1.2, -- 20% buffer
    gas_price_refresh_interval_seconds INTEGER DEFAULT 300, -- 5 minutes
    
    -- Feature flags
    enable_dynamic_tier_upgrades BOOLEAN DEFAULT true,
    enable_volume_discounts BOOLEAN DEFAULT true,
    enable_gas_fee_sharing BOOLEAN DEFAULT true,
    
    -- Emergency overrides
    maintenance_mode BOOLEAN DEFAULT false,
    force_minimum_fee_percent DECIMAL(5,4), -- Emergency minimum if needed
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Only one platform configuration should exist
CREATE UNIQUE INDEX idx_platform_config_single ON platform_configurations ((true));
```

### 2. Transaction Fee Breakdown Table
```sql
CREATE TABLE transaction_fee_breakdowns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    
    -- Original transaction
    total_amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    
    -- Platform fees
    platform_percentage_fee_cents BIGINT NOT NULL,
    platform_flat_fee_cents BIGINT NOT NULL,
    platform_gas_fee_cents BIGINT NOT NULL,
    total_platform_fee_cents BIGINT NOT NULL,
    
    -- Merchant receives
    merchant_amount_cents BIGINT NOT NULL,
    
    -- Gas details
    estimated_gas_units BIGINT NOT NULL,
    estimated_gas_price_gwei TEXT NOT NULL,
    actual_gas_units BIGINT,
    actual_gas_price_gwei TEXT,
    gas_fee_token VARCHAR(10) DEFAULT 'ETH',
    
    -- Transaction hashes
    platform_fee_tx_hash VARCHAR(255),
    merchant_payment_tx_hash VARCHAR(255),
    
    -- Status
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, completed, failed
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_fee_breakdowns_payment ON transaction_fee_breakdowns(payment_id);
CREATE INDEX idx_fee_breakdowns_status ON transaction_fee_breakdowns(status);
```

### 3. Update Payments Table
```sql
ALTER TABLE payments
ADD COLUMN platform_fee_cents BIGINT DEFAULT 0,
ADD COLUMN merchant_received_cents BIGINT,
ADD COLUMN fee_breakdown_id UUID REFERENCES transaction_fee_breakdowns(id);
```

## Delegation Server Updates

### 1. Batch Transaction Support

Update the delegation server to support multiple transactions in a single delegation:

```proto
// proto/delegation.proto
message BatchTransactionRequest {
    repeated Transaction transactions = 1;
    string delegation_id = 2;
    GasEstimate gas_estimate = 3;
}

message Transaction {
    string to_address = 1;
    string amount = 2;
    string token_address = 3;
    string description = 4;
}

message BatchTransactionResponse {
    repeated TransactionResult results = 1;
    string batch_id = 2;
    GasUsage gas_usage = 3;
}

message TransactionResult {
    string transaction_hash = 1;
    bool success = 2;
    string error_message = 3;
    int32 transaction_index = 4;
}

service DelegationService {
    // Existing methods...
    
    // New batch transaction method
    rpc ExecuteBatchTransaction(BatchTransactionRequest) returns (BatchTransactionResponse);
}
```

### 2. TypeScript Implementation

```typescript
// apps/delegation-server/src/services/batch-transaction.service.ts
import { ethers } from 'ethers';
import { DelegationFramework } from '@metamask/delegation-framework';

export class BatchTransactionService {
    constructor(
        private delegationFramework: DelegationFramework,
        private provider: ethers.Provider
    ) {}

    async executeBatchTransaction(request: BatchTransactionRequest): Promise<BatchTransactionResponse> {
        const { transactions, delegationId, gasEstimate } = request;
        
        // Validate delegation
        const delegation = await this.validateDelegation(delegationId);
        
        // Prepare batch calls
        const calls = transactions.map(tx => ({
            to: tx.toAddress,
            value: ethers.parseUnits(tx.amount, 18),
            data: '0x', // Simple transfer
        }));
        
        // Execute batch through delegation framework
        const batchTx = await this.delegationFramework.executeBatch(
            delegation,
            calls,
            {
                gasLimit: gasEstimate.gasUnits,
                maxFeePerGas: ethers.parseUnits(gasEstimate.gasPriceGwei, 'gwei'),
            }
        );
        
        // Wait for confirmation
        const receipt = await batchTx.wait();
        
        // Parse results
        const results = this.parseTransactionResults(receipt, transactions);
        
        return {
            results,
            batchId: receipt.transactionHash,
            gasUsage: {
                unitsUsed: receipt.gasUsed.toString(),
                actualGasPriceGwei: ethers.formatUnits(receipt.effectiveGasPrice, 'gwei'),
            },
        };
    }
}
```

## Payment Processing Updates

### 1. Dynamic Fee Service

```go
// libs/go/services/platform_fee_service.go

type PlatformFeeService struct {
    db                *db.Queries
    delegationClient  delegation.Client
    blockchainHelper  *helpers.BlockchainHelper
    cache            *cache.Cache // Cache tier configs
}

type WorkspaceFeeConfig struct {
    WorkspaceID            uuid.UUID
    TierID                 uuid.UUID
    TierName               string
    PercentageFee          float64
    FlatFeeCents           int64
    GasCoveragePercent     float64
    CustomPercentageFee    *float64
    CustomFlatFeeCents     *int64
    CustomGasCoverage      *float64
    PlatformWalletAddress  string
}

func (pfs *PlatformFeeService) GetWorkspaceFeeConfig(ctx context.Context, workspaceID uuid.UUID) (*WorkspaceFeeConfig, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("fee_config:%s", workspaceID)
    if cached, ok := pfs.cache.Get(cacheKey); ok {
        return cached.(*WorkspaceFeeConfig), nil
    }
    
    // Get workspace fee settings with tier information
    settings, err := pfs.db.GetWorkspaceFeeSettingsWithTier(ctx, workspaceID)
    if err != nil {
        // If no settings exist, create with default tier
        if err == sql.ErrNoRows {
            defaultTier, err := pfs.db.GetDefaultFeeTier(ctx)
            if err != nil {
                return nil, fmt.Errorf("failed to get default tier: %w", err)
            }
            
            settings, err = pfs.db.CreateWorkspaceFeeSettings(ctx, db.CreateWorkspaceFeeSettingsParams{
                WorkspaceID:      workspaceID,
                FeeTierProfileID: defaultTier.ID,
            })
            if err != nil {
                return nil, err
            }
        } else {
            return nil, err
        }
    }
    
    // Check if workspace is eligible for tier upgrade based on volume
    if settings.EnableDynamicTierUpgrades {
        pfs.checkAndUpdateTierEligibility(ctx, workspaceID, settings)
    }
    
    // Build config with custom overrides
    config := &WorkspaceFeeConfig{
        WorkspaceID:           workspaceID,
        TierID:                settings.FeeTierProfileID,
        TierName:              settings.TierName,
        PercentageFee:         settings.TierPercentageFee,
        FlatFeeCents:          settings.TierFlatFeeCents,
        GasCoveragePercent:    settings.TierGasCoveragePercent,
        PlatformWalletAddress: settings.PlatformWalletAddress,
    }
    
    // Apply custom overrides if they exist
    if settings.CustomPercentageFee.Valid {
        config.CustomPercentageFee = &settings.CustomPercentageFee.Float64
        config.PercentageFee = settings.CustomPercentageFee.Float64
    }
    if settings.CustomFlatFeeCents.Valid {
        config.CustomFlatFeeCents = &settings.CustomFlatFeeCents.Int64
        config.FlatFeeCents = settings.CustomFlatFeeCents.Int64
    }
    if settings.CustomGasCoveragePercent.Valid {
        config.CustomGasCoverage = &settings.CustomGasCoveragePercent.Float64
        config.GasCoveragePercent = settings.CustomGasCoveragePercent.Float64
    }
    
    // Cache for 5 minutes
    pfs.cache.Set(cacheKey, config, 5*time.Minute)
    
    return config, nil
}

func (pfs *PlatformFeeService) CalculateFeesForWorkspace(
    ctx context.Context, 
    workspaceID uuid.UUID, 
    amount int64,
    networkID uuid.UUID,
) (*FeeBreakdown, error) {
    // Get workspace-specific fee configuration
    config, err := pfs.GetWorkspaceFeeConfig(ctx, workspaceID)
    if err != nil {
        return nil, err
    }
    
    // Get platform configuration for gas settings
    platformConfig, err := pfs.db.GetPlatformConfiguration(ctx)
    if err != nil {
        return nil, err
    }
    
    // Calculate base fees
    percentageFee := int64(float64(amount) * config.PercentageFee)
    flatFee := config.FlatFeeCents
    
    // Estimate gas fee
    gasEstimate, err := pfs.estimateGasFee(ctx, networkID, platformConfig.GasEstimationMultiplier)
    if err != nil {
        return nil, err
    }
    
    // Apply gas coverage from tier
    platformCoversGas := int64(float64(gasEstimate) * (config.GasCoveragePercent / 100))
    merchantPaysGas := gasEstimate - platformCoversGas
    
    // Apply gas cap if configured
    if config.GasFeeCap > 0 && merchantPaysGas > config.GasFeeCap {
        merchantPaysGas = config.GasFeeCap
    }
    
    // Total fees
    totalPlatformFee := percentageFee + flatFee - platformCoversGas // Platform pays some gas
    totalMerchantFee := percentageFee + flatFee + merchantPaysGas
    merchantReceives := amount - totalMerchantFee
    
    // Validate minimum transaction
    if amount < platformConfig.GlobalMinimumTransactionCents {
        return nil, fmt.Errorf("transaction too small: minimum is %d cents", 
            platformConfig.GlobalMinimumTransactionCents)
    }
    
    // Create audit record
    audit := db.CreateFeeCalculationAuditParams{
        WorkspaceID:                   workspaceID,
        AppliedTierID:                 config.TierID,
        AppliedPercentageFee:          config.PercentageFee,
        AppliedFlatFeeCents:           flatFee,
        AppliedGasCoveragePercent:     config.GasCoveragePercent,
        TransactionAmountCents:        amount,
        CalculatedPercentageFeeCents:  percentageFee,
        CalculatedFlatFeeCents:        flatFee,
        CalculatedGasFeeCents:         gasEstimate,
        GasFeeCoveredByPlatformCents:  platformCoversGas,
        TotalPlatformFeeCents:         totalPlatformFee,
        MerchantChargedFeeCents:       totalMerchantFee,
        MerchantReceivesCents:         merchantReceives,
        RateReason:                    pfs.determineRateReason(config),
    }
    
    auditID, err := pfs.db.CreateFeeCalculationAudit(ctx, audit)
    if err != nil {
        // Log but don't fail the transaction
        log.Printf("Failed to create fee audit: %v", err)
    }
    
    return &FeeBreakdown{
        TotalAmount:           amount,
        PercentageFee:         percentageFee,
        FlatFee:               flatFee,
        EstimatedGas:          gasEstimate,
        PlatformCoversGas:     platformCoversGas,
        MerchantPaysGas:       merchantPaysGas,
        TotalPlatformFee:      totalPlatformFee,
        TotalMerchantFee:      totalMerchantFee,
        MerchantReceives:      merchantReceives,
        TierName:              config.TierName,
        AuditID:               auditID,
        PlatformWalletAddress: config.PlatformWalletAddress,
    }, nil
}

func (pfs *PlatformFeeService) ProcessPaymentWithDynamicFees(
    ctx context.Context, 
    payment *db.Payment,
    delegationID string,
) error {
    // Calculate fees with workspace-specific rates
    breakdown, err := pfs.CalculateFeesForWorkspace(
        ctx, 
        payment.WorkspaceID,
        payment.AmountInCents,
        payment.NetworkID,
    )
    if err != nil {
        return err
    }
    
    // Update volume tracking
    err = pfs.db.UpdateWorkspaceVolume(ctx, db.UpdateWorkspaceVolumeParams{
        WorkspaceID:    payment.WorkspaceID,
        AmountCents:    payment.AmountInCents,
    })
    if err != nil {
        log.Printf("Failed to update volume: %v", err)
    }
    
    // Pass fee configuration to delegation server
    feeConfig := &delegation.FeeConfiguration{
        PlatformWalletAddress: breakdown.PlatformWalletAddress,
        PlatformFeeAmount:    formatTokenAmount(breakdown.TotalPlatformFee),
        MerchantAmount:       formatTokenAmount(breakdown.MerchantReceives),
        TierName:             breakdown.TierName,
        FeeBreakdown: &delegation.FeeBreakdown{
            PercentageFee: formatTokenAmount(breakdown.PercentageFee),
            FlatFee:       formatTokenAmount(breakdown.FlatFee),
            GasFee:        formatTokenAmount(breakdown.MerchantPaysGas),
        },
    }
    
    // Execute batch transaction with fee config
    batchRequest := &delegation.BatchTransactionRequest{
        Transactions: []*delegation.Transaction{
            {
                ToAddress:   breakdown.PlatformWalletAddress,
                Amount:      feeConfig.PlatformFeeAmount,
                Description: fmt.Sprintf("Platform fee (%s tier)", breakdown.TierName),
            },
            {
                ToAddress:   payment.MerchantWalletAddress,
                Amount:      feeConfig.MerchantAmount,
                Description: "Payment to merchant",
            },
        },
        DelegationId:  delegationID,
        FeeConfig:     feeConfig,
    }
    
    // Execute transaction
    batchResponse, err := pfs.delegationClient.ExecuteBatchTransaction(ctx, batchRequest)
    if err != nil {
        return err
    }
    
    // Update records with results
    // ... (rest of implementation)
    
    return nil
}

func (pfs *PlatformFeeService) ProcessPaymentWithFees(ctx context.Context, payment *db.Payment) error {
    // Calculate fees
    breakdown, err := pfs.CalculateFees(ctx, payment.AmountInCents, true)
    if err != nil {
        return err
    }
    
    // Create fee breakdown record
    feeBreakdown, err := pfs.db.CreateTransactionFeeBreakdown(ctx, db.CreateTransactionFeeBreakdownParams{
        PaymentID:                   payment.ID,
        TotalAmountCents:           payment.AmountInCents,
        Currency:                   payment.Currency,
        PlatformPercentageFeeCents: breakdown.PlatformPercentageFee,
        PlatformFlatFeeCents:       breakdown.PlatformFlatFee,
        PlatformGasFeeCents:        breakdown.PlatformGasFee,
        TotalPlatformFeeCents:      breakdown.TotalPlatformFee,
        MerchantAmountCents:        breakdown.MerchantAmount,
        EstimatedGasUnits:          breakdown.EstimatedGasUnits,
        EstimatedGasPriceGwei:      breakdown.EstimatedGasPriceGwei,
    })
    if err != nil {
        return err
    }
    
    // Prepare batch transaction
    batchRequest := &delegation.BatchTransactionRequest{
        Transactions: []*delegation.Transaction{
            {
                ToAddress:   pfs.feeConfig.PlatformWalletAddress,
                Amount:      formatTokenAmount(breakdown.TotalPlatformFee),
                Description: "Platform fee",
            },
            {
                ToAddress:   payment.MerchantWalletAddress,
                Amount:      formatTokenAmount(breakdown.MerchantAmount),
                Description: "Payment to merchant",
            },
        },
        DelegationId: payment.DelegationID,
        GasEstimate: &delegation.GasEstimate{
            GasUnits:      breakdown.EstimatedGasUnits,
            GasPriceGwei:  breakdown.EstimatedGasPriceGwei,
        },
    }
    
    // Execute batch transaction
    batchResponse, err := pfs.delegationClient.ExecuteBatchTransaction(ctx, batchRequest)
    if err != nil {
        return err
    }
    
    // Update fee breakdown with transaction hashes
    err = pfs.db.UpdateFeeBreakdownTransactions(ctx, db.UpdateFeeBreakdownTransactionsParams{
        ID:                      feeBreakdown.ID,
        PlatformFeeTxHash:       batchResponse.Results[0].TransactionHash,
        MerchantPaymentTxHash:   batchResponse.Results[1].TransactionHash,
        ActualGasUnits:          batchResponse.GasUsage.UnitsUsed,
        ActualGasPriceGwei:      batchResponse.GasUsage.ActualGasPriceGwei,
        Status:                  "completed",
    })
    
    // Update payment record
    err = pfs.db.UpdatePaymentFees(ctx, db.UpdatePaymentFeesParams{
        ID:                    payment.ID,
        PlatformFeeCents:      breakdown.TotalPlatformFee,
        MerchantReceivedCents: breakdown.MerchantAmount,
        FeeBreakdownID:        feeBreakdown.ID,
    })
    
    return nil
}
```

### 2. Subscription Processor Updates

```go
// apps/subscription-processor/internal/processor/processor.go

func (p *Processor) processSubscriptionPayment(ctx context.Context, sub *db.Subscription) error {
    // Existing payment processing...
    
    // After creating payment record, process with fees
    err = p.platformFeeService.ProcessPaymentWithFees(ctx, payment)
    if err != nil {
        return fmt.Errorf("failed to process payment with fees: %w", err)
    }
    
    // Continue with subscription event creation...
}
```

## API Updates

### 1. Fee Management Endpoints

```go
// GET /api/v1/admin/fee-tiers - List all fee tiers (admin only)
type FeeTierResponse struct {
    ID                    string  `json:"id"`
    Name                  string  `json:"name"`
    Description           string  `json:"description"`
    TierLevel             int     `json:"tier_level"`
    PercentageFee         float64 `json:"percentage_fee"`
    FlatFeeCents          int64   `json:"flat_fee_cents"`
    GasCoveragePercent    float64 `json:"gas_coverage_percent"`
    MonthlyVolumeThreshold int64  `json:"monthly_volume_threshold_cents"`
    IsActive              bool    `json:"is_active"`
    IsDefault             bool    `json:"is_default"`
}

// POST /api/v1/admin/fee-tiers - Create new fee tier (admin only)
type CreateFeeTierRequest struct {
    Name                  string  `json:"name" binding:"required"`
    Description           string  `json:"description"`
    TierLevel             int     `json:"tier_level" binding:"required,min=1"`
    PercentageFee         float64 `json:"percentage_fee" binding:"required,min=0,max=1"`
    FlatFeeCents          int64   `json:"flat_fee_cents" binding:"required,min=0"`
    GasCoveragePercent    float64 `json:"gas_coverage_percent" binding:"min=0,max=100"`
    MonthlyVolumeThreshold int64  `json:"monthly_volume_threshold_cents"`
}

// PUT /api/v1/admin/workspaces/:id/fee-tier - Assign tier to workspace (admin only)
type AssignFeeTierRequest struct {
    FeeTierProfileID      string   `json:"fee_tier_profile_id" binding:"required"`
    CustomPercentageFee   *float64 `json:"custom_percentage_fee,omitempty"`
    CustomFlatFeeCents    *int64   `json:"custom_flat_fee_cents,omitempty"`
    CustomGasCoverage     *float64 `json:"custom_gas_coverage_percent,omitempty"`
    PromotionalRateEndsAt *string  `json:"promotional_rate_ends_at,omitempty"`
    Notes                 string   `json:"notes"`
}

// GET /api/v1/workspaces/current/fee-config - Get current workspace's fee configuration
type WorkspaceFeeConfigResponse struct {
    TierName              string   `json:"tier_name"`
    PercentageFee         float64  `json:"percentage_fee"`
    FlatFeeCents          int64    `json:"flat_fee_cents"`
    GasCoveragePercent    float64  `json:"gas_coverage_percent"`
    CurrentMonthVolume    int64    `json:"current_month_volume_cents"`
    CurrentYearVolume     int64    `json:"current_year_volume_cents"`
    NextTierEligibility   *NextTier `json:"next_tier_eligibility,omitempty"`
    PromotionalRateEndsAt *string   `json:"promotional_rate_ends_at,omitempty"`
}

type NextTier struct {
    TierName             string  `json:"tier_name"`
    RequiredMonthlyVolume int64  `json:"required_monthly_volume_cents"`
    CurrentProgress      float64 `json:"current_progress_percent"`
    EstimatedSavings     int64   `json:"estimated_monthly_savings_cents"`
}

// POST /api/v1/fees/preview - Preview fees for a transaction
type FeePreviewRequest struct {
    Amount       int64  `json:"amount" binding:"required,min=100"`
    Currency     string `json:"currency" binding:"required"`
    NetworkID    string `json:"network_id" binding:"required"`
    WorkspaceID  string `json:"workspace_id,omitempty"` // Optional, uses current if not provided
}

type FeePreviewResponse struct {
    TotalAmount      MoneyAmount `json:"total_amount"`
    MerchantReceives MoneyAmount `json:"merchant_receives"`
    FeeBreakdown     FeeDetails  `json:"fee_breakdown"`
    AppliedTier      string      `json:"applied_tier"`
}

type FeeDetails struct {
    PercentageFee     MoneyAmount `json:"percentage_fee"`
    FlatFee           MoneyAmount `json:"flat_fee"`
    EstimatedGasFee   MoneyAmount `json:"estimated_gas_fee"`
    GasCoveredByUs    MoneyAmount `json:"gas_covered_by_platform"`
    MerchantPaysGas   MoneyAmount `json:"merchant_pays_gas"`
    TotalPlatformFee  MoneyAmount `json:"total_platform_fee"`
    TotalMerchantFee  MoneyAmount `json:"total_merchant_fee"`
}

// GET /api/v1/admin/fee-analytics - Platform fee analytics (admin only)
type FeeAnalyticsResponse struct {
    TotalRevenue         MoneyAmount            `json:"total_revenue"`
    RevenueByTier        []TierRevenue          `json:"revenue_by_tier"`
    AverageTransactionFee MoneyAmount           `json:"average_transaction_fee"`
    GasFeeCoverage       MoneyAmount            `json:"total_gas_coverage"`
    TopMerchants         []MerchantFeeAnalytics `json:"top_merchants"`
}
```

### 2. Transaction History Enhancement

```go
// GET /api/v1/transactions/:id
type TransactionDetailsResponse struct {
    Transaction   Transaction   `json:"transaction"`
    FeeBreakdown  FeeBreakdown  `json:"fee_breakdown"`
    BlockchainTxs []BlockchainTx `json:"blockchain_transactions"`
}

type BlockchainTx struct {
    Hash        string    `json:"hash"`
    Type        string    `json:"type"` // platform_fee, merchant_payment
    Amount      string    `json:"amount"`
    Status      string    `json:"status"`
    BlockNumber int64     `json:"block_number"`
    Timestamp   time.Time `json:"timestamp"`
}
```

## Frontend Updates

### 1. Transaction Fee Display

```typescript
// components/TransactionFeeBreakdown.tsx
interface TransactionFeeBreakdownProps {
    amount: number;
    currency: string;
}

export function TransactionFeeBreakdown({ amount, currency }: TransactionFeeBreakdownProps) {
    const { data: feePreview } = useQuery({
        queryKey: ['fee-preview', amount, currency],
        queryFn: () => api.fees.preview({ amount, currency }),
    });
    
    return (
        <div className="border rounded-lg p-4 space-y-2">
            <div className="flex justify-between">
                <span>Transaction Amount:</span>
                <span>{formatMoney(amount, currency)}</span>
            </div>
            
            <div className="border-t pt-2 space-y-1 text-sm text-gray-600">
                <div className="flex justify-between">
                    <span>Platform Fee (1%):</span>
                    <span>{formatMoney(feePreview?.feeBreakdown.percentageFee)}</span>
                </div>
                <div className="flex justify-between">
                    <span>Transaction Fee:</span>
                    <span>{formatMoney(feePreview?.feeBreakdown.flatFee)}</span>
                </div>
                <div className="flex justify-between">
                    <span>Estimated Gas:</span>
                    <span>{formatMoney(feePreview?.feeBreakdown.estimatedGas)}</span>
                </div>
            </div>
            
            <div className="border-t pt-2 font-semibold">
                <div className="flex justify-between">
                    <span>You Receive:</span>
                    <span className="text-green-600">
                        {formatMoney(feePreview?.merchantReceives)}
                    </span>
                </div>
            </div>
        </div>
    );
}
```

### 2. Settings Page for Fee Configuration

```typescript
// pages/settings/platform-fees.tsx
export function PlatformFeesSettings() {
    const { data: config } = useQuery({
        queryKey: ['platform-fee-config'],
        queryFn: api.settings.getPlatformFeeConfig,
    });
    
    return (
        <SettingsLayout>
            <div className="space-y-6">
                <h2 className="text-xl font-semibold">Platform Fee Configuration</h2>
                
                <Alert>
                    <InfoIcon className="h-4 w-4" />
                    <AlertDescription>
                        Platform fees help cover blockchain transaction costs and platform operations.
                        Customers pay these fees on top of your prices.
                    </AlertDescription>
                </Alert>
                
                <div className="grid gap-4">
                    <div className="border rounded-lg p-4">
                        <h3 className="font-medium mb-2">Current Fee Structure</h3>
                        <div className="space-y-2 text-sm">
                            <div>Percentage Fee: {config?.percentageFee * 100}%</div>
                            <div>Transaction Fee: ${config?.flatFeeCents / 100}</div>
                            <div>Gas fees are included in platform fee</div>
                        </div>
                    </div>
                    
                    <div className="border rounded-lg p-4">
                        <h3 className="font-medium mb-2">Platform Wallet</h3>
                        <code className="text-xs bg-gray-100 p-2 rounded block break-all">
                            {config?.platformWalletAddress}
                        </code>
                    </div>
                </div>
            </div>
        </SettingsLayout>
    );
}
```

## Migration and Deployment

### 1. Database Migration
```sql
-- migrations/add_platform_fees.sql
BEGIN;

-- Create tables
CREATE TABLE platform_fee_configurations ...;
CREATE TABLE transaction_fee_breakdowns ...;

-- Update existing tables
ALTER TABLE payments ADD COLUMN platform_fee_cents ...;

-- Insert default configuration
INSERT INTO platform_fee_configurations ...;

COMMIT;
```

### 2. Feature Flags
```go
// Enable platform fees gradually
type FeatureFlags struct {
    EnablePlatformFees           bool
    EnableBatchTransactions      bool
    ShowFeeBreakdownInUI         bool
    RequireMinimumTransaction    bool
}
```

### 3. Rollout Strategy
1. **Phase 1**: Deploy infrastructure without activating fees
2. **Phase 2**: Enable fee calculation preview (read-only)
3. **Phase 3**: Enable for new transactions only
4. **Phase 4**: Full rollout with batch transactions

## Monitoring and Analytics

### 1. Key Metrics
- Total platform fees collected
- Average fee per transaction
- Gas fee accuracy (estimated vs actual)
- Transaction success rate
- Merchant satisfaction scores

### 2. Dashboards
```sql
-- Platform revenue dashboard query
SELECT 
    DATE_TRUNC('day', created_at) as date,
    COUNT(*) as transaction_count,
    SUM(platform_fee_cents) / 100.0 as platform_revenue,
    SUM(merchant_received_cents) / 100.0 as merchant_payouts,
    AVG(platform_fee_cents) / 100.0 as avg_platform_fee,
    SUM(platform_gas_fee_cents) / 100.0 as total_gas_fees
FROM transaction_fee_breakdowns
WHERE status = 'completed'
    AND created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY date
ORDER BY date DESC;
```

## Security Considerations

1. **Wallet Security**: Platform wallet private keys in AWS Secrets Manager
2. **Fee Validation**: Server-side validation of all fee calculations
3. **Audit Trail**: Complete log of all fee transactions
4. **Rate Limiting**: Prevent fee calculation abuse
5. **Monitoring**: Alerts for unusual fee patterns

## Testing Strategy

### 1. Unit Tests
```go
func TestPlatformFeeCalculation(t *testing.T) {
    tests := []struct {
        name           string
        amount         int64
        expectedFee    int64
        expectedMerchant int64
    }{
        {
            name:           "Standard transaction",
            amount:         10000, // $100
            expectedFee:    200,   // $1 + $0.25 + $0.75 gas
            expectedMerchant: 9800,
        },
        // More test cases...
    }
}
```

### 2. Integration Tests
- Test batch transaction execution
- Verify fee splitting accuracy
- Test edge cases (minimum amounts, failed transactions)

### 3. End-to-End Tests
- Complete payment flow with fee collection
- UI displays correct fee breakdowns
- Merchant receives correct amount

## Documentation

### 1. Merchant Documentation
- How platform fees work
- Fee structure explanation
- FAQ about gas fee handling
- Transaction history guide

### 2. API Documentation
- Fee preview endpoint
- Transaction details with fees
- Webhook updates with fee information

### 3. Customer Documentation
- Understanding transaction fees
- Why fees are charged
- How to see fee breakdowns

## Success Metrics

1. **Revenue Target**: Generate sustainable platform revenue
2. **Transparency**: 100% of fees clearly displayed
3. **Efficiency**: <5% variance in gas estimation
4. **Adoption**: >95% successful fee collections
5. **Satisfaction**: Maintain merchant NPS >50

## Example Tier Configurations

### Standard Tier Structure

```sql
-- Basic Tier (Default for new merchants)
-- 1.0% + $0.25, no gas coverage
INSERT INTO fee_tier_profiles (name, tier_level, percentage_fee, flat_fee_cents, gas_fee_coverage_percent) 
VALUES ('Basic', 3, 0.0100, 25, 0);

-- Growth Tier ($10K+ monthly volume)
-- 0.75% + $0.20, 25% gas coverage
INSERT INTO fee_tier_profiles (name, tier_level, percentage_fee, flat_fee_cents, gas_fee_coverage_percent, monthly_volume_threshold_cents) 
VALUES ('Growth', 2, 0.0075, 20, 25, 1000000);

-- Scale Tier ($50K+ monthly volume)
-- 0.6% + $0.15, 40% gas coverage
INSERT INTO fee_tier_profiles (name, tier_level, percentage_fee, flat_fee_cents, gas_fee_coverage_percent, monthly_volume_threshold_cents) 
VALUES ('Scale', 2, 0.0060, 15, 40, 5000000);

-- Enterprise Tier ($250K+ monthly volume)
-- 0.5% + $0.10, 50% gas coverage, gas cap at $2
INSERT INTO fee_tier_profiles (name, tier_level, percentage_fee, flat_fee_cents, gas_fee_coverage_percent, gas_fee_cap_cents, monthly_volume_threshold_cents) 
VALUES ('Enterprise', 1, 0.0050, 10, 50, 200, 25000000);
```

### Special Tier Examples

```sql
-- Launch Partner Tier (6-month promotional)
-- 0.25% + $0.05, 100% gas coverage
INSERT INTO fee_tier_profiles (name, tier_level, percentage_fee, flat_fee_cents, gas_fee_coverage_percent) 
VALUES ('Launch Partner', 1, 0.0025, 5, 100);

-- Non-Profit Tier
-- 0.5% + $0.10, 75% gas coverage
INSERT INTO fee_tier_profiles (name, tier_level, percentage_fee, flat_fee_cents, gas_fee_coverage_percent) 
VALUES ('Non-Profit', 1, 0.0050, 10, 75);

-- High-Risk Tier (for certain industries)
-- 2.0% + $0.50, no gas coverage
INSERT INTO fee_tier_profiles (name, tier_level, percentage_fee, flat_fee_cents, gas_fee_coverage_percent) 
VALUES ('High-Risk', 4, 0.0200, 50, 0);
```

## Fee Calculation Examples

### Example 1: Basic Tier ($100 transaction)
```
Transaction: $100.00
Percentage (1%): $1.00
Flat fee: $0.25
Gas estimate: $0.75
Gas covered by platform: $0.00 (0%)
-----------------------------------
Platform collects: $2.00 ($1.00 + $0.25 + $0.75)
Merchant receives: $98.00
```

### Example 2: Enterprise Tier ($1,000 transaction)
```
Transaction: $1,000.00
Percentage (0.5%): $5.00
Flat fee: $0.10
Gas estimate: $0.75
Gas covered by platform: $0.38 (50%)
Merchant pays gas: $0.37
-----------------------------------
Platform collects: $4.72 ($5.00 + $0.10 - $0.38)
Merchant receives: $994.63 ($1,000 - $5.00 - $0.10 - $0.37)
```

### Example 3: Launch Partner Tier ($50 transaction)
```
Transaction: $50.00
Percentage (0.25%): $0.13
Flat fee: $0.05
Gas estimate: $0.75
Gas covered by platform: $0.75 (100%)
-----------------------------------
Platform collects: -$0.57 (platform loses money but gains adoption)
Merchant receives: $49.82
```

## Migration and Tier Management

### Automatic Tier Upgrades

```go
// Scheduled job to evaluate tier eligibility
func (s *TierService) EvaluateTierUpgrades(ctx context.Context) error {
    // Get all workspaces eligible for evaluation
    workspaces, err := s.db.GetWorkspacesForTierEvaluation(ctx)
    if err != nil {
        return err
    }
    
    for _, ws := range workspaces {
        currentVolume := ws.CurrentMonthVolumeCents
        currentTier := ws.FeeTierProfileID
        
        // Find best eligible tier based on volume
        eligibleTier, err := s.db.GetBestEligibleTier(ctx, currentVolume)
        if err != nil {
            continue
        }
        
        // If better tier available, mark as eligible
        if eligibleTier.TierLevel < ws.CurrentTierLevel {
            s.db.UpdateWorkspaceTierEligibility(ctx, db.UpdateWorkspaceTierEligibilityParams{
                WorkspaceID:        ws.ID,
                EligibleForTierID:  eligibleTier.ID,
                TierEvaluationDate: time.Now(),
            })
            
            // Send notification
            s.notifyTierUpgradeAvailable(ws, eligibleTier)
        }
    }
    
    return nil
}
```

### Manual Tier Assignment (Admin)

```go
func (h *AdminHandler) AssignCustomTier(c *gin.Context) {
    workspaceID := c.Param("id")
    var req AssignFeeTierRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Verify tier exists
    tier, err := h.queries.GetFeeTierProfile(c, req.FeeTierProfileID)
    if err != nil {
        c.JSON(404, gin.H{"error": "Tier not found"})
        return
    }
    
    // Update workspace fee settings
    params := db.UpdateWorkspaceFeeTierParams{
        WorkspaceID:             workspaceID,
        FeeTierProfileID:        tier.ID,
        CustomPercentageFee:     req.CustomPercentageFee,
        CustomFlatFeeCents:      req.CustomFlatFeeCents,
        CustomGasCoveragePercent: req.CustomGasCoverage,
        Notes:                   req.Notes,
        ApprovedBy:              c.GetString("admin_id"),
    }
    
    if req.PromotionalRateEndsAt != nil {
        endTime, _ := time.Parse(time.RFC3339, *req.PromotionalRateEndsAt)
        params.PromotionalRateEndsAt = &endTime
    }
    
    err = h.queries.UpdateWorkspaceFeeTier(c, params)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to update tier"})
        return
    }
    
    // Clear cache
    h.cache.Delete(fmt.Sprintf("fee_config:%s", workspaceID))
    
    c.JSON(200, gin.H{"message": "Tier updated successfully"})
}
```

## Dashboard for Fee Management

### Admin Dashboard Queries

```sql
-- Platform revenue by tier
SELECT 
    ft.name as tier_name,
    COUNT(DISTINCT wfs.workspace_id) as merchant_count,
    SUM(fca.total_platform_fee_cents) / 100.0 as total_revenue,
    AVG(fca.total_platform_fee_cents) / 100.0 as avg_fee_per_transaction,
    SUM(fca.gas_fee_covered_by_platform_cents) / 100.0 as total_gas_covered
FROM fee_calculation_audits fca
JOIN workspace_fee_settings wfs ON fca.workspace_id = wfs.workspace_id
JOIN fee_tier_profiles ft ON wfs.fee_tier_profile_id = ft.id
WHERE fca.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY ft.name
ORDER BY total_revenue DESC;

-- Merchants approaching tier upgrade
SELECT 
    w.name as workspace_name,
    wfs.current_month_volume_cents / 100.0 as current_volume,
    ft_current.name as current_tier,
    ft_next.name as next_tier,
    ft_next.monthly_volume_threshold_cents / 100.0 as threshold,
    (wfs.current_month_volume_cents::float / ft_next.monthly_volume_threshold_cents * 100) as progress_percent
FROM workspace_fee_settings wfs
JOIN workspaces w ON wfs.workspace_id = w.id
JOIN fee_tier_profiles ft_current ON wfs.fee_tier_profile_id = ft_current.id
LEFT JOIN fee_tier_profiles ft_next ON ft_next.tier_level = ft_current.tier_level - 1
WHERE ft_next.id IS NOT NULL
    AND wfs.current_month_volume_cents > ft_next.monthly_volume_threshold_cents * 0.8
ORDER BY progress_percent DESC;
```

### Merchant Dashboard Component

```typescript
export function FeeConfiguration() {
    const { data: config } = useQuery({
        queryKey: ['workspace-fee-config'],
        queryFn: api.workspaces.getCurrentFeeConfig,
    });
    
    return (
        <Card>
            <CardHeader>
                <CardTitle>Your Fee Configuration</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="space-y-4">
                    <div className="flex justify-between">
                        <span>Current Tier:</span>
                        <Badge>{config?.tierName}</Badge>
                    </div>
                    
                    <div className="flex justify-between">
                        <span>Transaction Fees:</span>
                        <span>{config?.percentageFee * 100}% + ${config?.flatFeeCents / 100}</span>
                    </div>
                    
                    <div className="flex justify-between">
                        <span>Gas Coverage:</span>
                        <span>{config?.gasCoveragePercent}% covered by Cyphera</span>
                    </div>
                    
                    {config?.nextTierEligibility && (
                        <Alert>
                            <AlertTitle>Tier Upgrade Available!</AlertTitle>
                            <AlertDescription>
                                You're {config.nextTierEligibility.currentProgressPercent.toFixed(0)}% 
                                of the way to {config.nextTierEligibility.tierName} tier, 
                                which could save you ${config.nextTierEligibility.estimatedMonthlySavingsCents / 100} per month.
                            </AlertDescription>
                        </Alert>
                    )}
                    
                    <div className="pt-4">
                        <div className="text-sm text-gray-600 mb-2">Monthly Volume Progress</div>
                        <Progress value={config?.nextTierEligibility?.currentProgressPercent || 0} />
                        <div className="text-xs text-gray-500 mt-1">
                            ${config?.currentMonthVolume / 100} / ${config?.nextTierEligibility?.requiredMonthlyVolume / 100}
                        </div>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
```

This dynamic platform fee collection system ensures Cyphera can:
1. Generate sustainable revenue with flexible pricing
2. Reward high-volume merchants with better rates
3. Offer promotional pricing for strategic partnerships
4. Maintain complete audit trails for all fee calculations
5. Provide transparency to merchants about their fee structure
6. Automatically optimize merchant costs as they grow