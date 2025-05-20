# Product Plan: Two-Way Subscription Sync Service

**Overall Goal:** To enable seamless two-way synchronization of subscription-related data (customers, products, subscriptions, invoices, transactions, etc.) between Cyphera and various external subscription management and accounting platforms, starting with Stripe.

**Phase 1: Core Interface & Stripe Integration (MVP)**

1.  **Define Generic Go Interface (`SubscriptionSyncService`):**
    *   Location: `internal/client/payment_sync/interface.go`
    *   Responsibilities: Abstract core CRUD operations (Create, Read, Update, Delete, List) for key subscription entities and webhook handling.
    *   Entities: Customers, Products, Prices, Subscriptions, Invoices.
2.  **Stripe Implementation:**
    *   Create a Stripe-specific implementation of the `SubscriptionSyncService` interface using the `stripe-go` SDK.
    *   Handle authentication and API calls to Stripe.
    *   Develop mapping logic between Cyphera's internal data models and Stripe's data models.
3.  **Webhook Handling (Stripe):**
    *   Implement an HTTP handler in Cyphera to receive webhook events from Stripe.
    *   Use the `HandleWebhook` method of the Stripe service to validate (using `stripe-go`'s webhook utilities) and process these events (e.g., `customer.created`, `invoice.paid`, `subscription.updated`).
4.  **Initial Sync Functionality:**
    *   Develop a mechanism for an initial data pull from Stripe to Cyphera for existing customers, products, and active subscriptions.
5.  **Ongoing Sync Logic (Unidirectional - Stripe to Cyphera via Webhooks):**
    *   Process webhook events to update Cyphera's database in near real-time based on changes in Stripe.
6.  **Basic Two-Way Sync (Cyphera to Stripe for new entities):**
    *   Implement logic so that when a new relevant entity (e.g., a customer or subscription initiated via Cyphera) is created in Cyphera, it can be pushed to Stripe.
7.  **Configuration & Error Logging:**
    *   Securely manage API keys and webhook secrets for Stripe.
    *   Implement basic logging for sync operations and errors.

**Phase 2: Expanding Entities, Robustness & Basic Conflict Handling**

1.  **Expand Interface & Stripe Implementation:**
    *   Add support for more entities: Transactions (PaymentIntents/Charges), External Accounts (PaymentMethods), and potentially a simplified version of Product Features/Entitlements if crucial for your core logic.
2.  **Robust Error Handling & Retries:**
    *   Implement more sophisticated error handling for API calls and webhook processing.
    *   Introduce retry mechanisms for transient failures during sync operations.
3.  **Basic Conflict Resolution Strategy:**
    *   Define and implement a simple conflict resolution strategy (e.g., "last update wins" based on timestamps, or flag for manual review).
4.  **Idempotency:**
    *   Ensure all operations (especially creates and updates triggered by webhooks) are idempotent to prevent duplicate records or erroneous updates if events are received multiple times.
5.  **Monitoring & Health Checks:**
    *   Add an endpoint or mechanism to check the health of the sync service and its connection to Stripe.

**Phase 3: Full Two-Way Sync & Additional Platform (e.g., Accounting Software)**

1.  **Full Two-Way Sync for Stripe:**
    *   Implement logic for updates and deletions originating in Cyphera to be pushed to Stripe.
    *   Refine conflict resolution.
2.  **New Platform Integration (e.g., QuickBooks, NetSuite):**
    *   Implement the `SubscriptionSyncService` for a second platform. This will help validate the genericity of the interface.
    *   Develop mapping and webhook handling specific to the new platform.
3.  **Refine Generic Interface:**
    *   Adjust the `SubscriptionSyncService` interface based on learnings from integrating the second platform, ensuring it remains abstract enough for future integrations.
4.  **Pluggable Architecture:**
    *   Design the system to easily add new platform integrations by implementing the common interface. Consider a factory pattern or dependency injection to manage different service implementations.

**Phase 4: Advanced Features & Optimization**

1.  **Selective Sync:**
    *   Allow administrators to configure which entities or even which fields are synced for each platform.
2.  **Advanced Conflict Resolution:**
    *   Implement more sophisticated conflict resolution strategies, potentially with a UI for manual intervention.
3.  **Data Validation & Transformation:**
    *   Introduce configurable validation and transformation rules per integration to handle data inconsistencies.
4.  **Performance Optimization:**
    *   Implement batch operations for initial syncs and large updates.
    *   Explore delta syncs where possible instead of full data pulls.
5.  **Admin Dashboard:**
    *   Develop a UI for monitoring sync status, viewing logs, managing configurations, and handling conflicts. 

---

## Appendix A: Stripe Webhook Processing with AWS Serverless & Terraform

This section outlines the implementation plan for handling Stripe webhooks using an AWS serverless architecture (API Gateway, Lambda, SQS) orchestrated with Terraform. This corresponds to **Phase 1, Step 3: Webhook Handling (Stripe)** and parts of **Step 5: Ongoing Sync Logic**.

**Overall Architecture:** API Gateway (receives webhook) -> Lambda (validates & queues) -> SQS (buffers events) -> Lambda (processes events & updates DB).

**Phase 1: Core Infrastructure & Lambda Receiver (Terraform & Go)**

*   **Goal:** Set up the basic AWS infrastructure to receive, validate, and queue Stripe webhook events.
*   **Modules/Components:**
    1.  **Terraform: SQS Queue**
        *   Define an SQS Standard Queue (`stripe_webhook_events_queue`).
        *   Configure appropriate visibility timeout, message retention, and a Dead-Letter Queue (DLQ) (`stripe_webhook_events_dlq`) for failed processing attempts.
        *   **Outputs:** SQS Queue URL, SQS Queue ARN, DLQ ARN.
    2.  **Terraform: IAM Roles & Policies**
        *   **`StripeWebhookReceiverLambdaRole`**: 
            *   Permissions to write logs to CloudWatch.
            *   Permissions to send messages to `stripe_webhook_events_queue`.
            *   Permissions to read the Stripe Webhook Secret (e.g., from AWS Secrets Manager or SSM Parameter Store).
        *   **`StripeEventProcessorLambdaRole`**: (Define now, implement Lambda later)
            *   Permissions to write logs to CloudWatch.
            *   Permissions to read messages from `stripe_webhook_events_queue`.
            *   Permissions to delete messages from `stripe_webhook_events_queue`.
            *   Permissions to interact with your database (e.g., RDS, DynamoDB via VPC endpoints if applicable).
            *   Permissions to call other AWS services or your other API Lambdas if needed for processing.
    3.  **Terraform: AWS Secrets Manager (or SSM Parameter Store)**
        *   Store your Stripe Webhook Signing Secret securely.
        *   **Output:** Secret ARN.
    4.  **Go: `StripeWebhookReceiverLambda` Function**
        *   **Location:** New Lambda handler package (e.g., `cmd/stripe-webhook-receiver` or `internal/functions/stripe_webhook_receiver`).
        *   **Dependencies:**
            *   Your `internal/client/payment_sync/stripe` package (for `StripeService` and `webhook.go`).
            *   AWS SDK for Go (v2 recommended) for SQS and Secrets Manager.
            *   `github.com/aws/aws-lambda-go/lambda` and `github.com/aws/aws-lambda-go/events` (for API Gateway event types).
        *   **Logic:**
            *   Initialize `StripeService` in the Lambda's `init()` or globally:
                *   Fetch Stripe API Key and Webhook Secret from environment variables (populated by Terraform from Secrets Manager/SSM).
                *   Configure the `StripeService`.
            *   **Handler Function (`HandleRequest`)**:
                *   Input: `events.APIGatewayProxyRequest`.
                *   Extract request body and `Stripe-Signature` header.
                *   Call `stripeService.HandleWebhook(ctx, []byte(request.Body), signatureHeader)`.
                *   **If `HandleWebhook` returns an error related to signature or bad request:**
                    *   Log the error.
                    *   Return `events.APIGatewayProxyResponse{StatusCode: 400, Body: "Webhook validation failed"}`.
                *   **If `HandleWebhook` returns any other error (unexpected internal server error):**
                    *   Log the error.
                    *   Return `events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal server error during webhook pre-processing"}`.
                *   **If `HandleWebhook` is successful:**
                    *   Marshal the returned `ps.WebhookEvent` to JSON.
                    *   Create an SQS `SendMessageInput` with the JSON payload and the SQS Queue URL (from env var).
                    *   Use the AWS SDK SQS client to send the message to `stripe_webhook_events_queue`.
                    *   If SQS `SendMessage` fails:
                        *   Log the error.
                        *   Return `events.APIGatewayProxyResponse{StatusCode: 500, Body: "Failed to queue webhook event"}` (Stripe will retry).
                    *   Return `events.APIGatewayProxyResponse{StatusCode: 200, Body: "Webhook received"}`.
    5.  **Terraform: `StripeWebhookReceiverLambda` Resource**
        *   Define an `aws_lambda_function` resource.
        *   Source code: Point to the compiled Go binary for the receiver Lambda.
        *   Handler: `main` (or your compiled binary name).
        *   Runtime: `go1.x` or `provided.al2` (for custom Go runtimes).
        *   Role: Attach `StripeWebhookReceiverLambdaRole`.
        *   Environment Variables:
            *   `STRIPE_API_KEY`
            *   `STRIPE_WEBHOOK_SECRET_ARN` (or the secret value itself if not using Secrets Manager lookup in Lambda, though lookup is better)
            *   `SQS_QUEUE_URL` (from SQS output).
            *   `STRIPE_SERVICE_WEBHOOK_SECRET` (populated from the secret manager)
        *   Memory & Timeout: Start with reasonable defaults (e.g., 128MB, 10-15 seconds timeout - should be very quick).
    6.  **Terraform: API Gateway (HTTP API preferred for simplicity and cost)**
        *   Define an `aws_apigatewayv2_api` (HTTP API).
        *   Define an integration (`aws_apigatewayv2_integration`) of type `AWS_PROXY` for the `StripeWebhookReceiverLambda`.
        *   Define a route (`aws_apigatewayv2_route`) for `POST /webhooks/stripe` that targets the Lambda integration.
        *   Define a stage (`aws_apigatewayv2_stage`) with auto-deployment.
        *   **Output:** API Gateway Invoke URL (this is what you'll give to Stripe).

*   **Testing:**
    *   Deploy Phase 1 infrastructure.
    *   Manually send a test webhook from the Stripe Dashboard to your API Gateway endpoint.
    *   Verify:
        *   `StripeWebhookReceiverLambda` logs show event reception and validation.
        *   A message appears in the `stripe_webhook_events_queue`.
        *   Stripe Dashboard shows a `200 OK` for the webhook delivery.

**Phase 2: Lambda Processor & Database Interaction (Terraform & Go)**

*   **Goal:** Process queued webhook events and update your application's database.
*   **Modules/Components:**
    1.  **Go: `StripeEventProcessorLambda` Function**
        *   **Location:** New Lambda handler package (e.g., `cmd/stripe-event-processor` or `internal/functions/stripe_event_processor`).
        *   **Dependencies:**
            *   Your `internal/client/payment_sync` package (for `ps.WebhookEvent` and `ps.*` types).
            *   AWS SDK for Go (v2) for SQS.
            *   Your database interaction library (e.g., `database/sql`, an ORM, or DynamoDB client).
        *   **Logic:**
            *   **Handler Function (`HandleSQSEvent`)**:
                *   Input: `events.SQSEvent`.
                *   Iterate through `event.Records` (SQS messages).
                *   For each SQS message body:
                    *   Unmarshal the JSON payload back into a `ps.WebhookEvent`.
                    *   Log event details (`ProviderEventID`, `EventType`).
                    *   **Implement a `switch psEvent.EventType` statement:**
                        *   For each handled `EventType` (e.g., `customer.created`, `invoice.paid`):
                            *   Type-assert `psEvent.Data` to the expected `ps.*` type (e.g., `psCustomer, ok := psEvent.Data.(ps.Customer)`).
                            *   Perform database operations:
                                *   Insert new records.
                                *   Update existing records based on `ExternalID`.
                                *   Handle `*.deleted` events appropriately (soft delete, hard delete, update status).
                            *   Implement idempotency checks (e.g., check if `ProviderEventID` has already been processed, or use database constraints).
                        *   Log success or failure of database operations for each event.
                    *   **If processing fails for a message and the error is retryable, allow the Lambda to error out.** SQS will handle retries based on the queue's redrive policy. If non-retryable, log and ensure the message is eventually moved to the DLQ.
    2.  **Terraform: `StripeEventProcessorLambda` Resource**
        *   Define an `aws_lambda_function` resource.
        *   Source code: Compiled Go binary for the processor Lambda.
        *   Handler: `main`.
        *   Runtime: `go1.x` or `provided.al2`.
        *   Role: Attach `StripeEventProcessorLambdaRole`.
        *   Environment Variables:
            *   Database connection details (host, user, password via Secrets Manager, dbname).
            *   Any other necessary configuration.
        *   Memory & Timeout: Adjust based on processing needs (e.g., 256MB, 30-60 seconds).
        *   VPC Config: If your database is in a VPC, configure the Lambda to run in the same VPC.
    3.  **Terraform: SQS Event Source Mapping**
        *   Define an `aws_lambda_event_source_mapping` resource.
        *   Event Source ARN: `stripe_webhook_events_queue` ARN.
        *   Function Name: `StripeEventProcessorLambda` ARN.
        *   Batch Size: Start with a small batch size (e.g., 1 or 5) and adjust based on performance.

*   **Testing:**
    *   Deploy Phase 2 infrastructure.
    *   Send test webhooks from Stripe.
    *   Verify:
        *   Messages are consumed from `stripe_webhook_events_queue`.
        *   `StripeEventProcessorLambda` logs show successful processing.
        *   Your database reflects the changes triggered by the webhook events.
        *   Messages that fail processing (after retries) land in the DLQ.

**Phase 3: Monitoring, Refinements, and Error Handling**

*   **Goal:** Ensure robustness, observability, and easy debugging.
*   **Components:**
    1.  **CloudWatch Alarms & Dashboards:**
        *   Alarms for SQS queue depth (ApproximateNumberOfMessagesVisible) on both main queue and DLQ.
        *   Alarms for Lambda errors and throttles for both functions.
        *   CloudWatch Dashboard to visualize key metrics.
    2.  **Structured Logging:**
        *   Ensure all Lambdas use structured logging (like `zap` which you have) with consistent fields (`eventID`, `eventType`, `correlationID`, etc.) for easier querying in CloudWatch Logs Insights.
    3.  **DLQ Management Strategy:**
        *   Define a process for reviewing and re-processing (if applicable) messages from the DLQ.
    4.  **Idempotency Implementation:**
        *   Review and strengthen idempotency in the `StripeEventProcessorLambda`. Consider a separate table to track processed `ProviderEventID`s if database upserts aren't sufficient.
    5.  **Configuration Management:**
        *   Ensure all sensitive configurations (API keys, DB creds, webhook secrets) are managed securely (e.g., via AWS Secrets Manager) and referenced by Terraform.
    6.  **Expand Event Coverage:**
        *   Incrementally add more `case` statements in both Lambdas to handle more Stripe event types as per your business needs (refer to the TODOs in `webhook.go`). 