// Package offramp provides implementations for various off-ramp providers.
package bridge

import (
	// Added context for future API calls
	"fmt"

	offramp "cyphera-api/internal/client/offramp"

	httpclient "cyphera-api/internal/client/http" // Assuming this is the correct import path
	// "time" // Import time if needed for actual implementations later

	"context" // Added context for future API calls

	"net/url"

	"strings"

	"github.com/google/uuid" // For generating idempotency keys
)

// BridgeAdapter implements the OffRampProvider interface for Bridge.
type BridgeAdapter struct {
	// apiKey string // No longer storing apiKey directly if only used for client header
	client *httpclient.HTTPClient
}

// --- Minimal Structs for Bridge KYC Links (to fix compilation errors) ---

// NewBridgeAdapter creates a new instance of BridgeAdapter.
// It requires the Bridge API key and the base URL for the Bridge API.
func NewBridgeAdapter(apiKey string, bridgeAPIBaseURL string) (*BridgeAdapter, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Bridge API key is required")
	}
	if bridgeAPIBaseURL == "" {
		return nil, fmt.Errorf("Bridge API base URL is required")
	}

	// Configure the HTTPClient specifically for Bridge.
	// Bridge API requires an 'Api-Key' header for authentication.
	client := httpclient.NewHTTPClient(
		httpclient.WithBaseURL(bridgeAPIBaseURL),
		httpclient.WithDefaultHeader("Api-Key", apiKey),
		// Consider adding other client configurations if needed:
		// httpclient.WithTimeout(15 * time.Second), // Example timeout
		// httpclient.WithRetryConfig(httpclient.DefaultRetryConfig()), // Use default or custom retries
		// httpclient.WithMiddleware(httpclient.LoggingMiddleware()), // Enable if verbose logging is desired
	)

	return &BridgeAdapter{
		client: client,
	}, nil
}

// GetName returns the unique name of the provider.
func (a *BridgeAdapter) GetName() string {
	return "Bridge"
}

// GetCapabilities describes what networks, currencies, and jurisdictions the provider supports.
func (a *BridgeAdapter) GetCapabilities() (offramp.ProviderCapabilities, error) {
	consolidatedChainIDs, consolidatedCryptoCurrencies := getConsolidatedBridgeCapabilities()

	return offramp.ProviderCapabilities{
		SupportedChainIDs:         consolidatedChainIDs,
		SupportedCryptoCurrencies: consolidatedCryptoCurrencies,
		SupportedFiatCurrencies:   supportedBridgeFiatCurrencies,
		SupportedJurisdictions:    supportedBridgeJurisdictions,
		FeeStructureDetails:       bridgeFeeStructureDetails,
	}, nil
}

// InitiateKYC starts the Know Your Customer (KYC) or Know Your Business (KYB) process
// for an end-user with the provider using Bridge's KYC Links API.
func (a *BridgeAdapter) InitiateKYC(request offramp.KYCRequest) (offramp.KYCResponse, error) {
	ctx := context.Background() // Or use a context passed down from the caller

	bridgeReqType := ""
	fullName := ""

	switch request.UserDetails.EntityType {
	case "INDIVIDUAL":
		bridgeReqType = "individual"
		fullName = request.UserDetails.FirstName + " " + request.UserDetails.LastName
	case "BUSINESS":
		bridgeReqType = "business"
		fullName = request.UserDetails.LegalBusinessName
	default:
		return offramp.KYCResponse{}, fmt.Errorf("BridgeAdapter: unsupported entity type for KYC: %s", request.UserDetails.EntityType)
	}

	if fullName == "" {
		return offramp.KYCResponse{}, fmt.Errorf("BridgeAdapter: full name or legal business name is required for KYC")
	}
	if request.UserDetails.Email == "" {
		return offramp.KYCResponse{}, fmt.Errorf("BridgeAdapter: email is required for KYC")
	}

	// Prepare endorsements based on the IsSepa flag
	var endorsements []string
	if request.IsSepa != nil && *request.IsSepa {
		endorsements = append(endorsements, "sepa") // Add "sepa" endorsement if IsSepa is true
	}
	if len(endorsements) == 0 {
		endorsements = nil
	}

	payload := bridgeKYCLinksRequest{
		FullName:     fullName,
		Email:        request.UserDetails.Email,
		Type:         bridgeReqType,
		Endorsements: endorsements,                // Set based on IsSepa or empty if not applicable
		RedirectURI:  bridgeDefaultKYCRedirectURI, // Populate from const.go
	}

	idempotencyKey := uuid.NewString()
	headers := httpclient.WithHeader("Idempotency-Key", idempotencyKey)

	resp, err := a.client.Post(ctx, "/v0/kyc_links", payload, headers)
	if err != nil {
		// The HTTPClient already wraps basic errors, this adds adapter-specific context.
		return offramp.KYCResponse{}, fmt.Errorf("BridgeAdapter: InitiateKYC POST /v0/kyc_links failed: %w", err)
	}

	var bridgeResp bridgeKYCLinksResponse
	if procErr := a.client.ProcessJSONResponse(resp, &bridgeResp); procErr != nil {
		return offramp.KYCResponse{}, fmt.Errorf("BridgeAdapter: failed to process InitiateKYC response from Bridge: %w", procErr)
	}

	// Determine our internal status based on Bridge's KYC and TOS statuses.
	var internalStatus string
	// Example logic: if KYC or TOS is pending/not started, the process isn't complete from our POV.
	// This might need refinement based on how these statuses are actually used.
	if bridgeResp.KYCStatus == "not_started" || bridgeResp.KYCStatus == "pending_submission" || bridgeResp.KYCStatus == "pending_documents" ||
		bridgeResp.TOSStatus == "pending" || bridgeResp.TOSStatus == "not_accepted" {
		internalStatus = "ACTION_REQUIRED" // Indicates user needs to act on the links.
	} else {
		// If Bridge gives a more definitive status like "pending_review" for KYC itself, that could be used.
		// For now, if not clearly actionable by user, consider it "LINKS_GENERATED" or map to a PENDING status.
		internalStatus = "LINKS_GENERATED" // Default if links are generated but no immediate specific user action is clear from these statuses alone.
	}
	// If bridgeResp.CustomerID is available and KYC/TOS are done, it might indicate a more advanced status.
	// This initial mapping is conservative.

	return offramp.KYCResponse{
		KYCStatus:           internalStatus,
		HostedKYCURL:        bridgeResp.KYCLink,
		ProviderReferenceID: bridgeResp.ID, // Use the `id` from the response as the reference.
		NextSteps:           fmt.Sprintf("User action required: Accept ToS at %s and complete KYC at %s. Current KYC status: %s, ToS status: %s.", bridgeResp.TOSLink, bridgeResp.KYCLink, bridgeResp.KYCStatus, bridgeResp.TOSStatus),
		// Store CustomerID if available and useful at this stage.
		// ProviderCustomerID: bridgeResp.CustomerID, // Add to offramp.KYCResponse if needed
	}, nil
}

// GetKYCStatus retrieves the current KYC/KYB status of an end-user from the provider
// using Bridge's KYC Links API.
func (a *BridgeAdapter) GetKYCStatus(request offramp.KYCStatusRequest) (offramp.KYCStatusResponse, error) {
	ctx := context.Background() // Or use a context passed down from the caller

	if request.ProviderReferenceID == "" {
		return offramp.KYCStatusResponse{}, fmt.Errorf("BridgeAdapter: ProviderReferenceID (kycLinkID) is required for GetKYCStatus")
	}

	endpoint := fmt.Sprintf("/v0/kyc_links/%s", request.ProviderReferenceID)

	resp, err := a.client.Get(ctx, endpoint)
	if err != nil {
		return offramp.KYCStatusResponse{}, fmt.Errorf("BridgeAdapter: GetKYCStatus GET %s failed: %w", endpoint, err)
	}

	var bridgeResp bridgeKYCLinkStatusResponse
	if procErr := a.client.ProcessJSONResponse(resp, &bridgeResp); procErr != nil {
		return offramp.KYCStatusResponse{}, fmt.Errorf("BridgeAdapter: failed to process GetKYCStatus response from Bridge: %w", procErr)
	}

	// Map Bridge's status to our generic status.
	// This mapping should align with the states defined in offramp.KYCStatusResponse.
	ourStatus := ""
	reason := ""
	if len(bridgeResp.RejectionReasons) > 0 {
		// Simple join for now. Could be more structured if needed.
		reason = fmt.Sprintf("Rejection reasons: %v", bridgeResp.RejectionReasons)
	}

	switch bridgeResp.KYCStatus {
	case "not_started", "pending_submission", "pending_documents":
		ourStatus = "ACTION_REQUIRED" // User needs to do something
	case "pending_review", "processing":
		ourStatus = "PENDING_REVIEW"
	case "approved", "active": // "active" added as per GetCapabilities common statuses
		ourStatus = "ACTIVE"
	case "denied", "rejected", "failed": // "failed" added as per GetCapabilities
		ourStatus = "REJECTED"
	default:
		ourStatus = "UNKNOWN" // Or pass through bridgeResp.KYCStatus if it's informative and doesn't map cleanly.
		if bridgeResp.KYCStatus != "" {
			reason = fmt.Sprintf("Bridge KYC Status: %s. %s", bridgeResp.KYCStatus, reason)
		}
	}

	// Consider TOSStatus as well if it gates full "ACTIVE" status.
	// For example, if KYCStatus is "approved" but TOSStatus is "pending", maybe it's not fully "ACTIVE" yet.
	// Current logic prioritizes KYCStatus for the main mapping.

	return offramp.KYCStatusResponse{
		KYCStatus: ourStatus,
		Reason:    reason,
		// ProviderCustomerID: bridgeResp.CustomerID, // Add to offramp.KYCStatusResponse if needed
	}, nil
}

// GetAllKYCLinks retrieves a list of KYC links from Bridge, optionally filtered by customer ID or email.
func (a *BridgeAdapter) GetAllKYCLinks(request offramp.GetAllKYCLinksRequest) (offramp.GetAllKYCLinksResponse, error) {
	ctx := context.Background() // Or use a context passed down from the caller

	// Prepare query parameters
	queryParams := bridgeGetAllKYCLinksRequestQueryParams{
		CustomerID: request.CustomerID,
		Email:      request.Email,
	}

	// Construct endpoint with query parameters
	// Note: The httpclient.Get method might need to support passing query parameters directly,
	// or we build the query string manually here.
	// For now, assuming manual construction or a helper if available.
	endpoint := "/v0/kyc_links"
	queryValues := url.Values{}
	if queryParams.CustomerID != "" {
		queryValues.Add("customer_id", queryParams.CustomerID)
	}
	if queryParams.Email != "" {
		queryValues.Add("email", queryParams.Email)
	}
	if len(queryValues) > 0 {
		endpoint += "?" + queryValues.Encode()
	}

	resp, err := a.client.Get(ctx, endpoint)
	if err != nil {
		return offramp.GetAllKYCLinksResponse{}, fmt.Errorf("BridgeAdapter: GetAllKYCLinks GET %s failed: %w", endpoint, err)
	}

	var bridgeAPIResp bridgeGetAllKYCLinksAPIResponse
	if procErr := a.client.ProcessJSONResponse(resp, &bridgeAPIResp); procErr != nil {
		return offramp.GetAllKYCLinksResponse{}, fmt.Errorf("BridgeAdapter: failed to process GetAllKYCLinks response from Bridge: %w", procErr)
	}

	// Map Bridge response to generic offramp response
	genericKYCLinks := make([]offramp.KYCLinkDetail, len(bridgeAPIResp.Data))
	for i, bridgeLink := range bridgeAPIResp.Data {
		genericRejectionReasons := make([]offramp.RejectionReasonDetail, len(bridgeLink.RejectionReasons))
		for j, bridgeReason := range bridgeLink.RejectionReasons {
			genericRejectionReasons[j] = offramp.RejectionReasonDetail{
				DeveloperReason: bridgeReason.DeveloperReason,
				Reason:          bridgeReason.Reason,
				CreatedAt:       bridgeReason.CreatedAt,
			}
		}

		genericKYCLinks[i] = offramp.KYCLinkDetail{
			ID:                 bridgeLink.ID,
			FullName:           bridgeLink.FullName,
			Email:              bridgeLink.Email,
			Type:               bridgeLink.Type,
			KYCLinkURL:         bridgeLink.KYCLink,
			TOSLinkURL:         bridgeLink.TOSLink,
			KYCStatus:          bridgeLink.KYCStatus,
			TOSStatus:          bridgeLink.TOSStatus,
			ProviderCustomerID: bridgeLink.CustomerID,
			RejectionReasons:   genericRejectionReasons,
		}
	}

	return offramp.GetAllKYCLinksResponse{
		Count: bridgeAPIResp.Count,
		Data:  genericKYCLinks,
	}, nil
}

// GetWallet retrieves a specific wallet and its balances for a customer from Bridge.
func (a *BridgeAdapter) GetWallet(request offramp.GetWalletRequest) (offramp.GetWalletResponse, error) {
	ctx := context.Background() // Or use a context passed down from the caller

	if request.ProviderCustomerID == "" {
		return offramp.GetWalletResponse{}, fmt.Errorf("BridgeAdapter: ProviderCustomerID is required for GetWallet")
	}
	if request.ProviderWalletID == "" {
		return offramp.GetWalletResponse{}, fmt.Errorf("BridgeAdapter: ProviderWalletID is required for GetWallet")
	}

	endpoint := fmt.Sprintf("/v0/customers/%s/wallets/%s", request.ProviderCustomerID, request.ProviderWalletID)

	resp, err := a.client.Get(ctx, endpoint)
	if err != nil {
		return offramp.GetWalletResponse{}, fmt.Errorf("BridgeAdapter: GetWallet GET %s failed: %w", endpoint, err)
	}

	var bridgeResp bridgeGetWalletResponse
	if procErr := a.client.ProcessJSONResponse(resp, &bridgeResp); procErr != nil {
		return offramp.GetWalletResponse{}, fmt.Errorf("BridgeAdapter: failed to process GetWallet response from Bridge: %w", procErr)
	}

	// Map Bridge balances to generic balances
	genericBalances := make([]offramp.WalletBalance, len(bridgeResp.Balances))
	for i, bBalance := range bridgeResp.Balances {
		genericBalances[i] = offramp.WalletBalance{
			Amount:          bBalance.Balance,
			Currency:        bBalance.Currency,
			Network:         bBalance.Chain, // Bridge uses 'chain' for balance network
			ContractAddress: bBalance.ContractAddress,
		}
	}

	return offramp.GetWalletResponse{
		ProviderWalletID: bridgeResp.ID,
		Network:          bridgeResp.Chain, // Bridge uses 'chain' for wallet network
		WalletAddress:    bridgeResp.Address,
		CreatedAt:        bridgeResp.CreatedAt,
		UpdatedAt:        bridgeResp.UpdatedAt,
		Balances:         genericBalances,
	}, nil
}

// GetAllWallets retrieves a list of wallets for a customer from Bridge, with support for pagination.
func (a *BridgeAdapter) GetAllWallets(request offramp.GetAllWalletsRequest) (offramp.GetAllWalletsResponse, error) {
	ctx := context.Background() // Or use a context passed down from the caller

	if request.ProviderCustomerID == "" {
		return offramp.GetAllWalletsResponse{}, fmt.Errorf("BridgeAdapter: ProviderCustomerID is required for GetAllWallets")
	}

	// Prepare query parameters
	queryParams := bridgeGetAllWalletsRequestQueryParams{
		Limit:         request.Limit,
		StartingAfter: request.StartingAfter,
		EndingBefore:  request.EndingBefore,
	}

	endpoint := fmt.Sprintf("/v0/customers/%s/wallets", request.ProviderCustomerID)
	queryValues := url.Values{}
	if queryParams.Limit != nil {
		queryValues.Add("limit", fmt.Sprintf("%d", *queryParams.Limit))
	}
	if queryParams.StartingAfter != nil {
		queryValues.Add("starting_after", *queryParams.StartingAfter)
	}
	if queryParams.EndingBefore != nil {
		queryValues.Add("ending_before", *queryParams.EndingBefore)
	}

	if len(queryValues) > 0 {
		endpoint += "?" + queryValues.Encode()
	}

	resp, err := a.client.Get(ctx, endpoint)
	if err != nil {
		return offramp.GetAllWalletsResponse{}, fmt.Errorf("BridgeAdapter: GetAllWallets GET %s failed: %w", endpoint, err)
	}

	var bridgeAPIResp bridgeGetAllWalletsAPIResponse
	if procErr := a.client.ProcessJSONResponse(resp, &bridgeAPIResp); procErr != nil {
		return offramp.GetAllWalletsResponse{}, fmt.Errorf("BridgeAdapter: failed to process GetAllWallets response from Bridge: %w", procErr)
	}

	// Map Bridge response to generic offramp response
	genericWallets := make([]offramp.CreateWalletResponse, len(bridgeAPIResp.Data))
	for i, bridgeWalletItem := range bridgeAPIResp.Data {
		genericWallets[i] = offramp.CreateWalletResponse{
			ProviderWalletID: bridgeWalletItem.ID,
			Network:          bridgeWalletItem.Chain, // Bridge uses 'chain' for wallet network
			WalletAddress:    bridgeWalletItem.Address,
			CreatedAt:        bridgeWalletItem.CreatedAt,
			UpdatedAt:        bridgeWalletItem.UpdatedAt,
		}
	}

	return offramp.GetAllWalletsResponse{
		Count: bridgeAPIResp.Count,
		Data:  genericWallets,
	}, nil
}

// GetDepositAddress provides a crypto deposit address for the end-user.
func (a *BridgeAdapter) GetDepositAddress(request offramp.DepositAddressRequest) (offramp.DepositAddressResponse, error) {
	// TODO: Implement actual logic for Bridge GetDepositAddress.
	// Bridge uses "Liquidation Addresses". This would involve POSTing to /v0/customers/{customer_id}/liquidation_addresses
	// Remember 'Idempotency-Key'.
	return offramp.DepositAddressResponse{}, fmt.Errorf("BridgeAdapter: GetDepositAddress not implemented")
}

// InitiateOffRamp starts the crypto-to-fiat conversion and payout process.
func (a *BridgeAdapter) InitiateOffRamp(request offramp.OffRampTransactionRequest) (offramp.OffRampTransactionResponse, error) {
	// TODO: Implement actual logic for Bridge InitiateOffRamp using their /v0/transfers API.
	// Remember 'Idempotency-Key'.
	return offramp.OffRampTransactionResponse{}, fmt.Errorf("BridgeAdapter: InitiateOffRamp not implemented")
}

// GetOffRampStatus retrieves the current status of an off-ramp transaction.
func (a *BridgeAdapter) GetOffRampStatus(request offramp.OffRampStatusRequest) (offramp.OffRampStatusResponse, error) {
	// TODO: Implement actual logic for Bridge GetOffRampStatus, likely GET to /v0/transfers/{transfer_id}
	return offramp.OffRampStatusResponse{}, fmt.Errorf("BridgeAdapter: GetOffRampStatus not implemented")
}

// CreateWallet creates a new custodial wallet for a customer on a specific chain using Bridge.
// It maps the generic offramp.CreateWalletRequest to Bridge's specific API requirements.
func (a *BridgeAdapter) CreateWallet(request offramp.CreateWalletRequest) (offramp.CreateWalletResponse, error) {
	ctx := context.Background() // Or use a context passed down from the caller

	if request.ProviderCustomerID == "" {
		return offramp.CreateWalletResponse{}, fmt.Errorf("BridgeAdapter: ProviderCustomerID is required for CreateWallet")
	}
	if request.Network == "" {
		return offramp.CreateWalletResponse{}, fmt.Errorf("BridgeAdapter: Network (chain) is required for CreateWallet")
	}

	// For Bridge, the 'chain' parameter is typically lowercase (e.g., "solana", "base").
	// We should ensure our internal 'Network' value is mapped correctly.
	// For now, a simple lowercase conversion. This might need a more robust mapping if names differ significantly.
	bridgeChain := strings.ToLower(request.Network)

	payload := bridgeCreateWalletRequest{
		Chain: bridgeChain,
	}

	endpoint := fmt.Sprintf("/v0/customers/%s/wallets", request.ProviderCustomerID)
	idempotencyKey := uuid.NewString()
	headers := httpclient.WithHeader("Idempotency-Key", idempotencyKey)

	resp, err := a.client.Post(ctx, endpoint, payload, headers)
	if err != nil {
		return offramp.CreateWalletResponse{}, fmt.Errorf("BridgeAdapter: CreateWallet POST %s failed: %w", endpoint, err)
	}

	var bridgeResp bridgeCreateWalletResponse
	if procErr := a.client.ProcessJSONResponse(resp, &bridgeResp); procErr != nil {
		return offramp.CreateWalletResponse{}, fmt.Errorf("BridgeAdapter: failed to process CreateWallet response from Bridge: %w", procErr)
	}

	return offramp.CreateWalletResponse{
		ProviderWalletID: bridgeResp.ID,
		Network:          request.Network, // Return the network name as passed in the request for consistency
		WalletAddress:    bridgeResp.Address,
		CreatedAt:        bridgeResp.CreatedAt,
		UpdatedAt:        bridgeResp.UpdatedAt,
	}, nil
}

// Ensure BridgeAdapter implements OffRampProvider at compile time.
var _ offramp.OffRampProvider = (*BridgeAdapter)(nil)
