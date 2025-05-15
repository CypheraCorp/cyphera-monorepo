package bridge

// Assuming this is the correct import path
// --- Structs for Bridge KYC Links API ---

// bridgeKYCLinksRequest is the request payload for Bridge's POST /v0/kyc_links endpoint.
type bridgeKYCLinksRequest struct {
	FullName     string   `json:"full_name"`
	Email        string   `json:"email"`
	Type         string   `json:"type"` // "individual" or "business"
	Endorsements []string `json:"endorsements,omitempty"`
	RedirectURI  string   `json:"redirect_uri,omitempty"`
}

// bridgeKYCLinksResponse is the response payload from Bridge's POST /v0/kyc_links endpoint (200 OK).
// Based on user provided "200 KYC LINKS GENERATED"
type bridgeKYCLinksResponse struct {
	ID         string `json:"id"`
	FullName   string `json:"full_name"`
	Email      string `json:"email"`
	Type       string `json:"type"`
	KYCLink    string `json:"kyc_link"`
	TOSLink    string `json:"tos_link"`
	KYCStatus  string `json:"kyc_status"` // e.g., "not_started"
	TOSStatus  string `json:"tos_status"` // e.g., "pending"
	CustomerID string `json:"customer_id"`
}

// bridgeKYCLinkStatusResponse is the response from Bridge's GET /v0/kyc_links/{kycLinkID}
// Based on user provided "200 status for approved user" and "200 Status for rejected user"
type bridgeKYCLinkStatusResponse struct {
	ID               string                        `json:"id"`
	FullName         string                        `json:"full_name"`
	Email            string                        `json:"email"`
	Type             string                        `json:"type"`
	KYCLink          string                        `json:"kyc_link"`
	TOSLink          string                        `json:"tos_link"`
	KYCStatus        string                        `json:"kyc_status"` // e.g., "approved", "rejected"
	RejectionReasons []bridgeRejectionReasonDetail `json:"rejection_reasons,omitempty"`
	TOSStatus        string                        `json:"tos_status"`
	CustomerID       string                        `json:"customer_id"`
}

// bridgeGetAllKYCLinksRequestQueryParams holds the optional query parameters for the GET /v0/kyc_links endpoint.
type bridgeGetAllKYCLinksRequestQueryParams struct {
	CustomerID string `url:"customer_id,omitempty"`
	Email      string `url:"email,omitempty"`
}

// bridgeRejectionReasonDetail represents a detailed rejection reason object from Bridge.
type bridgeRejectionReasonDetail struct {
	DeveloperReason string `json:"developer_reason"`
	Reason          string `json:"reason"`
	CreatedAt       string `json:"created_at"` // Consider time.Time if parsing/comparison is needed
}

// bridgeGetAllKYCLinksAPIResponse is the top-level response structure for the GET /v0/kyc_links endpoint.
type bridgeGetAllKYCLinksAPIResponse struct {
	Count int                           `json:"count"`
	Data  []bridgeKYCLinkStatusResponse `json:"data"` // Reuses bridgeKYCLinkStatusResponse which will be updated
}

// --- Structs for Bridge Create Wallet Endpoint (POST /v0/customers/{customerID}/wallets) ---

// bridgeCreateWalletRequest is the request payload for creating a Bridge wallet.
type bridgeCreateWalletRequest struct {
	Chain string `json:"chain"` // e.g., "solana", "ethereum", "base"
}

// bridgeCreateWalletResponse is the successful response payload from creating a Bridge wallet (201 Created).
type bridgeCreateWalletResponse struct {
	ID        string `json:"id"` // Bridge Wallet ID, e.g., "bw_123"
	Chain     string `json:"chain"`
	Address   string `json:"address"`    // The actual wallet address on the chain
	CreatedAt string `json:"created_at"` // Consider time.Time if parsing/comparison is needed
	UpdatedAt string `json:"updated_at"` // Consider time.Time if parsing/comparison is needed
}

// --- Structs for Bridge Get Wallet Endpoint (GET /v0/customers/{customerID}/wallets/{bridgeWalletID}) ---

// bridgeWalletBalance represents a balance object within a Bridge wallet.
type bridgeWalletBalance struct {
	Balance         string `json:"balance"`          // e.g., "100.25"
	Currency        string `json:"currency"`         // e.g., "usdb", "usdc"
	Chain           string `json:"chain"`            // e.g., "solana"
	ContractAddress string `json:"contract_address"` // e.g., "ENL66PGy8d8j5KNqLtCcg4uidDUac5ibt45wbjH9REzB"
}

// bridgeGetWalletResponse is the successful response payload from retrieving a specific Bridge wallet.
// This extends bridgeCreateWalletResponse with balance information.
type bridgeGetWalletResponse struct {
	ID        string                `json:"id"`
	Chain     string                `json:"chain"`
	Address   string                `json:"address"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
	Balances  []bridgeWalletBalance `json:"balances,omitempty"` // Balances might not always be present or could be empty
}

// --- Structs for Bridge Get All Wallets Endpoint (GET /v0/customers/{customerID}/wallets) ---

// bridgeGetAllWalletsRequestQueryParams holds the optional query parameters for listing wallets.
type bridgeGetAllWalletsRequestQueryParams struct {
	Limit         *int    `url:"limit,omitempty"`          // Pointer to allow omitting if not set, default is 10, max 100
	StartingAfter *string `url:"starting_after,omitempty"` // Pointer to allow omitting
	EndingBefore  *string `url:"ending_before,omitempty"`  // Pointer to allow omitting
}

// bridgeWalletListItem represents a single wallet in a list response from Bridge.
// Note: Balances are not included in the list view according to the provided example.
// This structure is identical to bridgeCreateWalletResponse for now.
// If list items ever differ more significantly, this can be adjusted.
type bridgeWalletListItem struct {
	ID        string `json:"id"`
	Chain     string `json:"chain"`
	Address   string `json:"address"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// bridgeGetAllWalletsAPIResponse is the top-level response structure for listing all wallets for a customer.
type bridgeGetAllWalletsAPIResponse struct {
	Count int                    `json:"count"`
	Data  []bridgeWalletListItem `json:"data"`
}

// --- Structs for Bridge Create Customer Request (POST /v0/customers) ---

// bridgeCustomerRequestAddress represents an address object in the create customer request.
type bridgeCustomerRequestAddress struct {
	StreetLine1 string `json:"street_line_1"`
	StreetLine2 string `json:"street_line_2,omitempty"`
	City        string `json:"city"`
	Subdivision string `json:"subdivision,omitempty"` // ISO 3166-2, required for US
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"` // ISO 3166-1 alpha-3
}

// bridgeIdentifyingInformationRequest represents an identifying document object in a request.
type bridgeIdentifyingInformationRequest struct {
	Type           string `json:"type"`
	IssuingCountry string `json:"issuing_country"` // ISO 3166-1 alpha-3
	Number         string `json:"number"`
	Description    string `json:"description,omitempty"`
	Expiration     string `json:"expiration,omitempty"`  // YYYY-MM-DD
	ImageFront     string `json:"image_front,omitempty"` // Base64 encoded
	ImageBack      string `json:"image_back,omitempty"`  // Base64 encoded
}

// bridgeAssociatedPersonRequest represents an associated person object in a request.
type bridgeAssociatedPersonRequest struct {
	FirstName                        string                                `json:"first_name"`
	MiddleName                       string                                `json:"middle_name,omitempty"`
	LastName                         string                                `json:"last_name"`
	TransliteratedFirstName          string                                `json:"transliterated_first_name,omitempty"`
	TransliteratedMiddleName         string                                `json:"transliterated_middle_name,omitempty"`
	TransliteratedLastName           string                                `json:"transliterated_last_name,omitempty"`
	Email                            string                                `json:"email"`
	Phone                            string                                `json:"phone"`
	ResidentialAddress               bridgeCustomerRequestAddress          `json:"residential_address"`
	TransliteratedResidentialAddress *bridgeCustomerRequestAddress         `json:"transliterated_residential_address,omitempty"`
	BirthDate                        string                                `json:"birth_date"` // YYYY-MM-DD
	HasOwnership                     bool                                  `json:"has_ownership"`
	HasControl                       bool                                  `json:"has_control"`
	IsSigner                         bool                                  `json:"is_signer"`
	IsDirector                       bool                                  `json:"is_director"`
	Title                            string                                `json:"title,omitempty"`
	OwnershipPercentage              *int                                  `json:"ownership_percentage,omitempty"`
	RelationshipEstablishedAt        string                                `json:"relationship_established_at,omitempty"` // YYYY-MM-DD
	Nationality                      string                                `json:"nationality"`                           // ISO 3166-1 alpha-3
	VerifiedSelfieAt                 string                                `json:"verified_selfie_at,omitempty"`
	CompletedCustomerSafetyCheckAt   string                                `json:"completed_customer_safety_check_at,omitempty"`
	IdentifyingInformation           []bridgeIdentifyingInformationRequest `json:"identifying_information"`
}

// bridgeDocumentRequest represents a document object for upload in a request.
type bridgeDocumentRequest struct {
	Purposes    []string `json:"purposes"`
	File        string   `json:"file"` // Base64 encoded
	Description string   `json:"description,omitempty"`
}

// bridgeRegulatedActivityRequest represents regulated activity details in a request.
type bridgeRegulatedActivityRequest struct {
	RegulatedActivitiesDescription    string `json:"regulated_activities_description"`
	PrimaryRegulatoryAuthorityCountry string `json:"primary_regulatory_authority_country"` // ISO 3166-1 alpha-3
	PrimaryRegulatoryAuthorityName    string `json:"primary_regulatory_authority_name"`
	LicenseNumber                     string `json:"license_number"`
}

// bridgeCreateBusinessCustomerRequest is the request payload for Bridge's POST /v0/customers for a business.
type bridgeCreateBusinessCustomerRequest struct {
	Type                             string                                `json:"type"` // "business"
	BusinessLegalName                string                                `json:"business_legal_name"`
	TransliteratedBusinessLegalName  string                                `json:"transliterated_business_legal_name,omitempty"`
	BusinessTradeName                string                                `json:"business_trade_name"`
	TransliteratedBusinessTradeName  string                                `json:"transliterated_business_trade_name,omitempty"`
	BusinessDescription              string                                `json:"business_description"`
	Email                            string                                `json:"email"`
	BusinessType                     string                                `json:"business_type"`
	PrimaryWebsite                   string                                `json:"primary_website,omitempty"`
	OtherWebsites                    []string                              `json:"other_websites,omitempty"`
	RegisteredAddress                bridgeCustomerRequestAddress          `json:"registered_address"`
	TransliteratedRegisteredAddress  *bridgeCustomerRequestAddress         `json:"transliterated_registered_address,omitempty"`
	PhysicalAddress                  bridgeCustomerRequestAddress          `json:"physical_address"`
	TransliteratedPhysicalAddress    *bridgeCustomerRequestAddress         `json:"transliterated_physical_address,omitempty"`
	SignedAgreementID                string                                `json:"signed_agreement_id"`
	IsDAO                            bool                                  `json:"is_dao"`
	ComplianceScreeningExplanation   string                                `json:"compliance_screening_explanation,omitempty"`
	AssociatedPersons                []bridgeAssociatedPersonRequest       `json:"associated_persons"`
	Documents                        []bridgeDocumentRequest               `json:"documents,omitempty"`
	Endorsements                     []string                              `json:"endorsements,omitempty"`
	BusinessIndustry                 []string                              `json:"business_industry"`
	OwnershipThreshold               *int                                  `json:"ownership_threshold,omitempty"`
	HasMaterialIntermediaryOwnership *bool                                 `json:"has_material_intermediary_ownership,omitempty"`
	EstimatedAnnualRevenueUSD        string                                `json:"estimated_annual_revenue_usd,omitempty"`
	ExpectedMonthlyPaymentsUSD       *int                                  `json:"expected_monthly_payments_usd,omitempty"`
	OperatesInProhibitedCountries    *bool                                 `json:"operates_in_prohibited_countries,omitempty"`
	AccountPurpose                   string                                `json:"account_purpose"`
	AccountPurposeOther              string                                `json:"account_purpose_other,omitempty"`
	HighRiskActivitiesExplanation    string                                `json:"high_risk_activities_explanation,omitempty"`
	HighRiskActivities               []string                              `json:"high_risk_activities,omitempty"`
	SourceOfFunds                    string                                `json:"source_of_funds"`
	SourceOfFundsDescription         string                                `json:"source_of_funds_description,omitempty"`
	ConductsMoneyServices            bool                                  `json:"conducts_money_services"`
	ConductsMoneyServicesUsingBridge *bool                                 `json:"conducts_money_services_using_bridge,omitempty"`
	ConductsMoneyServicesDescription string                                `json:"conducts_money_services_description,omitempty"`
	IdentifyingInformation           []bridgeIdentifyingInformationRequest `json:"identifying_information,omitempty"`
	RegulatedActivity                *bridgeRegulatedActivityRequest       `json:"regulated_activity,omitempty"`
}

// --- Structs for Bridge Create Customer Response (POST /v0/customers) ---

// bridgeCustomerAddressResponse is the address structure within the Create Customer success response.
// Note: This might be identical to bridgeCustomerRequestAddress. If so, they can be merged.
// For now, keeping it separate in case response structure deviates slightly.
type bridgeCustomerAddressResponse struct {
	StreetLine1 string `json:"street_line_1"`
	City        string `json:"city"`
	State       string `json:"state"` // For US, this is typically the 2-letter code.
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"` // ISO 3166-1 alpha-3
}

// bridgeEndorsementResponse represents an endorsement status within the Create Customer success response.
type bridgeEndorsementResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// bridgeBeneficialOwnerResponse represents a beneficial owner within the Create Customer success response.
type bridgeBeneficialOwnerResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// bridgeCreateCustomerResponse is the successful response payload from Bridge's POST /v0/customers endpoint (201 Created).
type bridgeCreateCustomerResponse struct {
	ID                        string                          `json:"id"`
	FirstName                 string                          `json:"first_name"`
	LastName                  string                          `json:"last_name"`
	Status                    string                          `json:"status"`
	Type                      string                          `json:"type"`
	Email                     string                          `json:"email"`
	HasAcceptedTermsOfService bool                            `json:"has_accepted_terms_of_service"`
	Address                   bridgeCustomerAddressResponse   `json:"address"`
	RejectionReasons          []string                        `json:"rejection_reasons,omitempty"`
	RequirementsDue           []string                        `json:"requirements_due,omitempty"`
	FutureRequirementsDue     []string                        `json:"future_requirements_due,omitempty"`
	Endorsements              []bridgeEndorsementResponse     `json:"endorsements,omitempty"`
	BeneficialOwners          []bridgeBeneficialOwnerResponse `json:"beneficial_owners,omitempty"`
	CreatedAt                 string                          `json:"created_at"`
	UpdatedAt                 string                          `json:"updated_at"`
}
