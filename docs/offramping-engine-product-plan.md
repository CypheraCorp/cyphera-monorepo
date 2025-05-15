# Product Plan: Unified Off-Ramping Engine

**Overall Goal:** To create a robust and flexible off-ramping engine that allows merchants to seamlessly utilize different off-ramp providers (Fern, Bridge, HIFI) based on their specific needs (e.g., supported networks, currencies, jurisdictions, fees). The engine will abstract the complexities of individual provider APIs behind a common interface.

---

## Phase 1: Core Abstraction & MVP Integration

*   **Objective:** Establish the foundational abstraction layer and integrate the three providers for core off-ramp functionality. Provide merchants with a way to use a pre-configured provider.
*   **Key Deliverables:**
    1.  **Define Core Go Interfaces & Structs:**
        *   `OffRampProvider` Interface:
            ```go
            // OffRampProvider defines the common interface for all off-ramp providers.
            type OffRampProvider interface {
                // GetName returns the unique name of the provider.
                GetName() string

                // GetCapabilities describes what networks, currencies, and jurisdictions the provider supports.
                GetCapabilities() (ProviderCapabilities, error)

                // InitiateKYC starts the KYC/KYB process for an end-user.
                // It should return a URL for a hosted KYC flow or necessary details to guide the user.
                // Note: Providers perform their own KYB. This method initiates that process, 
                // which might involve redirecting to a provider's hosted form or submitting 
                // collected data via API for provider verification.
                InitiateKYC(request KYCRequest) (KYCResponse, error)

                // GetKYCStatus retrieves the current KYC/KYB status of an end-user.
                GetKYCStatus(request KYCStatusRequest) (KYCStatusResponse, error)

                // GetDepositAddress provides a crypto deposit address (custodial wallet) for the end-user.
                // This is where the end-user will send crypto to be off-ramped.
                GetDepositAddress(request DepositAddressRequest) (DepositAddressResponse, error)

                // InitiateOffRamp starts the crypto-to-fiat conversion and payout.
                InitiateOffRamp(request OffRampTransactionRequest) (OffRampTransactionResponse, error)

                // GetOffRampStatus retrieves the status of an off-ramp transaction.
                GetOffRampStatus(request OffRampStatusRequest) (OffRampStatusResponse, error)
            }
            ```
        *   Supporting Structs:
            ```go
            // ProviderCapabilities describes the capabilities of an off-ramp provider.
            type ProviderCapabilities struct {
                SupportedNetworks         []string
                SupportedCryptoCurrencies []string
                SupportedFiatCurrencies   []string
                SupportedJurisdictions    []string
                FeeStructureDetails       string // Could be a more complex type or a link to documentation
            }

            // UserInfo contains basic information about an end-user.
            // This should be expanded based on the common minimum requirements for KYC/KYB.
            type UserInfo struct {
                // For Individuals
                FirstName   string
                LastName    string
                Email       string
                DateOfBirth string // e.g., "YYYY-MM-DD"
                // Address fields (Street, City, PostalCode, Country, etc.)
                // ... add more individual fields as needed

                // For Businesses
                LegalBusinessName string
                BusinessType      string // e.g., "LLC", "Corporation"
                RegistrationCountry string
                // ... add more business fields as needed
            }

            // KYCRequest is used to initiate the KYC/KYB process.
            type KYCRequest struct {
                UserID      string // Your internal ID for the end-user
                UserDetails UserInfo
                MerchantID  string // Your internal ID for the merchant initiating this
            }

            // KYCResponse contains the result of initiating KYC/KYB.
            type KYCResponse struct {
                KYCStatus           string // e.g., "PENDING_SUBMISSION", "INITIATED"
                HostedKYCURL        string // URL for the user to complete KYC/KYB, if applicable
                ProviderReferenceID string // ID from the provider for this KYC/KYB attempt
                NextSteps           string // Human-readable next steps, if any
            }

            // KYCStatusRequest is used to request the status of a KYC/KYB process.
            type KYCStatusRequest struct {
                UserID              string // Your internal ID for the end-user
                ProviderReferenceID string // ID from the provider for this KYC/KYB attempt
            }

            // KYCStatusResponse contains the current status of a KYC/KYB process.
            type KYCStatusResponse struct {
                KYCStatus string // e.g., "PENDING_REVIEW", "ACTIVE", "REJECTED", "ACTION_REQUIRED"
                Reason    string // Reason for rejection or action required
            }

            // DepositAddressRequest is used to request a deposit address.
            type DepositAddressRequest struct {
                UserID         string // Your internal ID for the end-user
                Network        string // e.g., "Ethereum", "Polygon", "Solana"
                CryptoCurrency string // e.g., "USDC", "ETH"
            }

            // DepositAddressResponse contains the deposit address details.
            type DepositAddressResponse struct {
                Address        string
                Network        string
                Memo           string // If required (e.g., for certain networks/exchanges)
                QRCodeURI      string // Optional: URI for a QR code representation of the address
            }

            // OffRampTransactionRequest is used to initiate an off-ramp transaction.
            type OffRampTransactionRequest struct {
                UserID                   string          // Your internal ID for the end-user
                SourceCryptoAmount       string          // Using string for decimal precision, e.g., "100.50"
                SourceCryptoCurrency     string          // e.g., "USDC"
                SourceNetwork            string          // e.g., "Ethereum"
                DestinationFiatCurrency  string          // e.g., "USD", "EUR"
                DestinationBankAccountID string          // Your internal ID for the pre-registered bank account with the provider
                IdempotencyKey           string          // To prevent duplicate transactions
            }

            // OffRampTransactionResponse contains the result of initiating an off-ramp.
            type OffRampTransactionResponse struct {
                TransactionID       string // Provider's transaction ID
                Status              string // e.g., "PENDING_FUNDS", "PROCESSING"
                EstimatedFiatAmount string // Using string for decimal precision
                DepositInstructions string // Any instructions for the user to send crypto
            }

            // OffRampStatusRequest is used to request the status of an off-ramp transaction.
            type OffRampStatusRequest struct {
                TransactionID string // Provider's transaction ID
            }

            // OffRampStatusResponse contains the current status of an off-ramp transaction.
            type OffRampStatusResponse struct {
                Status              string    // e.g., "PROCESSING", "COMPLETED", "FAILED", "RETURNED"
                ActualFiatAmount    string    // Using string for decimal precision
                PayoutDate          string    // e.g., "YYYY-MM-DDTHH:MM:SSZ" or just "YYYY-MM-DD"
                ProviderFeeAmount   string    // Fee charged by the provider
                ReasonForFailure    string    // If status is FAILED or RETURNED
            }
            ```
    2.  **Provider Adapters Implementation:**
        *   Create `fern_adapter.go`, `bridge_adapter.go`, and `hifi_adapter.go`.
        *   Each adapter will implement the `OffRampProvider` interface.
        *   Map provider-specific API calls, authentication (API keys), and data models to the common interface methods and structs.
            *   Each provider's adapter must handle the specific mechanism for initiating KYB (e.g., retrieving a hosted URL, submitting KYB data directly if the provider API supports it).
            *   **Fern:** Utilize `Customers API` for KYC/KYB (likely involves their hosted, brandable forms), `Payment accounts API` for wallets (EVM, Solana etc.), `Quotes` & `Transactions API` for off-ramps.
            *   **Bridge:** Utilize `Customers API` or `KYC Links API` for individuals and businesses. The adapter will need to handle data submission or link generation as appropriate for KYB. Use `Liquidation Address API` or `Transfers API` (stablecoin to fiat).
            *   **HIFI:** Utilize `User creation` & `KYC endpoints` for individuals and businesses, potentially submitting collected data for KYB. Use `Liquidation Address` creation, and `Offramp` API.
    3.  **Configuration Management:**
        *   Securely store and manage API keys and other configurations (e.g., webhook secrets) per merchant per provider.
    4.  **Basic Off-Ramp Service:**
        *   An internal service that allows a merchant to specify which provider they want to use for a particular end-user or transaction.
        *   The service will route requests to the correct adapter.
    5.  **Webhook Ingestion (Initial):**
        *   Develop endpoints to receive webhook notifications from Fern, Bridge, and HIFI regarding KYC status changes and transaction updates.
        *   Initial implementation will focus on logging these events and providing a way to correlate them to internal user/transaction records.
        *   Implement signature verification for webhooks as specified by each provider.
            *   Fern: `x-api-signature`, `x-api-timestamp`.
            *   Bridge: Mentions webhook event signature verification.
            *   HIFI: (Assumed, as it's standard practice – docs should be re-checked for specifics if not readily found).

---

## Phase 2: Dynamic Routing & Enhanced Merchant Experience

*   **Objective:** Introduce intelligence for provider selection and provide merchants with better visibility and control.
*   **Key Deliverables:**
    1.  **Dynamic Provider Selection Engine:**
        *   Develop logic to automatically suggest or select the optimal provider based on:
            *   End-user's jurisdiction.
            *   Desired off-ramp currency.
            *   Crypto asset and network being sent.
            *   Reported provider fees and FX rates (requires a system to fetch/estimate these).
            *   Potentially, provider operational status (if available via APIs).
        *   This relies on accurately populating and maintaining the `ProviderCapabilities` for each adapter.
    2.  **Normalized Fee & FX Rate Abstraction:**
        *   Develop a system to query or estimate fees and FX rates across providers for a given transaction scenario.
        *   Present these in a comparable format.
    3.  **Consolidated Transaction & KYC Management:**
        *   APIs for merchants to view a unified list of their end-users' KYC statuses and transaction histories, regardless of the underlying provider used.
    4.  **Advanced Webhook Processing:**
        *   Normalize incoming webhook data into a common internal event format.
        *   Implement robust retry and error handling for webhook processing.
        *   Update internal transaction/KYC statuses based on these normalized events.
    5.  **Merchant Configuration API:**
        *   APIs allowing merchants to configure their provider preferences, default settings, and potentially rules for dynamic routing.

    6.  **Enhanced Bank Account Onboarding & Management (with Plaid Integration):**
        *   **Objective:** Streamline the process for end-users (or merchants on their behalf) to add and manage bank accounts for off-ramping, reducing friction and data entry errors.
        *   **Key Features:**
            *   Integrate Plaid Link into your platform as the primary method for collecting end-user bank account information.
            *   Securely retrieve necessary bank account details (account number, routing number, owner name, type) from Plaid.
            *   Develop logic within each provider adapter (`fern_adapter.go`, `bridge_adapter.go`, `hifi_adapter.go`) to:
                *   Utilize direct Plaid integration (e.g., `processor_token` hand-off, or provider-specific Plaid flows like Bridge's) with the provider if supported for creating/linking off-ramp bank accounts.
                *   If direct Plaid integration is not supported by a provider for this purpose, use the Plaid-sourced bank details to programmatically create the bank account record via the provider's standard API.
            *   Store a reference to the Plaid-linked item and map it to the corresponding bank account IDs created within each off-ramp provider's system.
            *   The `DestinationBankAccountID` in `OffRampTransactionRequest` would then refer to this registered account at the specific provider.
        *   **Benefits:** Improved user experience, reduced data entry errors, standardized bank account collection for your platform, and potentially faster onboarding with providers that have deep Plaid integrations.

---

## Phase 3: Advanced Features, Scalability & Ecosystem

*   **Objective:** Enhance robustness, add value-added services, and prepare for future growth.
*   **Key Deliverables:**
    1.  **On-Ramp Functionality (Optional Extension):**
        *   If business needs dictate, extend the interfaces and adapters to support fiat-to-crypto on-ramping.
        *   Leverage features like Fern's "First-party Onramps," Bridge's "Virtual Accounts," and HIFI's "Virtual Accounts."
    2.  **Automated Failover & Retry Logic:**
        *   Implement strategies to retry a failed transaction with an alternative compatible provider, if configured by the merchant.
    3.  **Developer/Merchant Dashboard (API-driven):**
        *   Expose APIs that could power a simple dashboard for merchants to:
            *   Manage their API keys for the engine.
            *   View analytics (transaction volume, success rates per provider).
            *   Configure routing rules.
            *   Manage webhook endpoints for notifications *from* our engine.
    4.  **Enhanced Monitoring & Alerting:**
        *   Comprehensive internal monitoring of the engine's health, provider API latencies, error rates, and transaction throughput.
        *   Alerting for critical issues.
    5.  **Extensibility for New Providers:**
        *   Refine the adapter pattern and configuration system to simplify the integration of new off-ramp providers in the future.

---

## Key Considerations Throughout Development:

This section outlines crucial aspects that require continuous attention throughout all phases of the project to ensure a robust, secure, and compliant off-ramping engine.

*   **Security:** Prioritize secure storage of API keys, sensitive data, and robust authentication/authorization for your engine's APIs.
*   **Error Handling:** Implement comprehensive error handling and clear logging at each layer of the abstraction. Return meaningful error messages.
*   **Idempotency:** Ensure operations, especially transaction initiation, are idempotent where supported by the underlying providers (all three mention Idempotency-Keys for POST requests). Your engine should also manage idempotency for its own operations.
*   **KYB Process:** Recognize that all off-ramp providers will perform their own KYB/KYC checks on merchants and end-users. The engine's role is to facilitate this by initiating the provider's specific onboarding flow (e.g., via API data submission or by directing to a hosted page) and tracking status, not to bypass or replace the provider's own due diligence.
*   **Compliance & Data Privacy:** While providers handle primary KYC/KYB, your engine will process and store sensitive user data. Ensure compliance with relevant data privacy regulations.

### Data Management & Sensitivity

*   **KYB Data Storage Strategy:**
    *   **Objective:** Minimize storage of sensitive KYB data to reduce compliance and security risks, while retaining enough information for operational efficiency.
    *   **Data to Store (Your Database):**
        *   Internal Merchant ID (your unique identifier for the merchant).
        *   Provider-Specific Merchant IDs (assigned by Fern, Bridge, HIFI, etc.).
        *   KYB Status per provider (e.g., `PENDING`, `ACTIVE`, `REJECTED`) and the timestamp of the last status update.
        *   Basic, non-sensitive business profile data for reference and display (e.g., Legal Business Name, Country of Registration).
        *   Dates of KYB application/initiation per provider.
    *   **Data to AVOID Storing (Rely on Provider as Source of Truth for Sensitive Details):**
        *   Sensitive business financial details submitted for KYB (e.g., bank account numbers, detailed revenue figures).
        *   Tax Identification Numbers (e.g., EINs).
        *   Detailed Beneficial Ownership information (names, addresses, DOBs, government ID numbers of owners/directors).
        *   Copies of business documents submitted for KYB (e.g., articles of incorporation, business licenses).
        *   Detailed source of funds or source of wealth information.
    *   **Rationale:** This approach leverages the off-ramp providers as the primary custodians of sensitive, verified KYB data. It aligns with data minimization principles, reduces your platform's direct compliance overhead for that data, and lessens the security risk associated with storing such information.

*   **Testing:** Thorough unit, integration, and end-to-end testing will be crucial, especially given the interaction with multiple external systems. Mocking provider APIs will be essential for testing.
*   **Documentation:**
    *   Internal: Clear `godoc` for all Go code.
    *   External: API documentation for merchants using your engine.
*   **Rate Limits:** Be mindful of and handle rate limits imposed by each provider's API.
*   **Settlement Times & Processes:** Understand and communicate the different settlement times and processes of each provider, as this impacts the end-user experience.
*   **Provider API Versioning:** Providers may update their APIs. Plan for how to manage and adapt to these changes. 