# Cyphera Infrastructure & Deployment Overview

This document outlines the infrastructure setup for the Cyphera API and Delegation Server, focusing on the deployment process using Terraform, GitHub Actions, and the Serverless Framework.

**Core Technologies:**

*   **AWS:** Cloud provider for all infrastructure components.
*   **Terraform:** Infrastructure as Code (IaC) tool to define and manage AWS resources.
*   **Serverless Framework:** Tool for deploying the Go API Lambda function and related API Gateway resources.
*   **Docker:** Used to containerize the Node.js Delegation Server.
*   **GitHub Actions:** CI/CD platform for automating builds, tests, and deployments.
*   **AWS SSM Parameter Store:** Used for storing non-sensitive configuration and pointers (ARNs, endpoints) shared between Terraform and applications.
*   **AWS Secrets Manager:** Used for securely storing and managing sensitive secrets (API keys, private keys, DB passwords).

## I. Terraform Infrastructure (`infrastructure/terraform/` directory)

Terraform defines the foundational cloud infrastructure required by the applications.

**Key Components Managed by Terraform:**

*   **Networking:**
    *   **VPC** (`main.tf`, using `terraform-aws-modules/vpc/aws`): Creates a Virtual Private Cloud with public and private subnets across multiple Availability Zones.
    *   **NAT Gateway** (`main.tf`): Provides outbound internet access for resources in private subnets (enabled for `dev` and `prod`, single gateway used for cost optimization).
    *   **Security Groups** (`main.tf`, `rds.tf`, `delegation_server_alb.tf`): Network firewalls controlling traffic between resources (e.g., Lambda SG, RDS SG, ALB SG, ECS Task SG). Rules restrict access based on source security groups (least privilege).
*   **Compute:**
    *   **ECS Cluster** (`main.tf`): A cluster (`cyphera-delegation-cluster-${stage}`) to host the Delegation Server tasks.
    *   **ECS Task Definition** (`delegation_server_ecs.tf`): Defines the Delegation Server container (image, CPU/Memory, ports, environment variables, secrets). CPU/Memory are conditionally sized based on stage (`dev`/`prod`).
        *   **Workaround Note:** Due to a persistent issue in the Terraform AWS provider (tested up to v5.40) where changes to SSM Parameter Store values referenced via `valueFrom` in `container_definitions` incorrectly trigger a resource replacement plan, a `lifecycle { ignore_changes = [container_definitions] }` block is applied to this resource. This prevents Terraform from repeatedly trying to replace the task definition unnecessarily.
        *   **Consequence:** While changes to the *values* stored in the referenced SSM parameters or Secrets Manager secrets will be picked up by new tasks automatically, any *structural* changes made to the `container_definitions` block within `infrastructure/terraform/delegation_server_ecs.tf` (e.g., updating the image tag, adding/removing ports, adding/removing environment variables or secrets) **will NOT be applied by `terraform apply`**. These structural changes must be manually applied by creating a new task definition revision in the AWS ECS console and updating the ECS service to use it.
        *   **Future:** Periodically check Terraform AWS provider release notes for fixes related to planning `container_definitions` with `valueFrom`. If fixed, this `lifecycle` block can potentially be removed.
    *   **ECS Service** (`delegation_server_ecs.tf`): Manages running instances (tasks) of the Delegation Server Task Definition on Fargate, ensuring the desired count (conditional based on stage) is running and registers them with the ALB Target Group.
*   **Data Store:**
    *   **RDS PostgreSQL Instance** (`rds.tf`): Managed relational database. Instance size, storage, Multi-AZ, backups, and performance insights are conditionally configured based on stage for cost optimization and production readiness.
    *   **Secrets Manager Secret Link:** Terraform ensures RDS manages the master password in its own automatically created secret and makes the ARN available via an SSM Parameter (`/cyphera/rds-secret-arn-${stage}`).
*   **Load Balancing:**
    *   **Application Load Balancer (ALB)** (`delegation_server_alb.tf`): Internal ALB distributing gRPC traffic (via HTTPS listener on port 443) to the Delegation Server ECS tasks.
    *   **Target Group** (`delegation_server_alb.tf`): Configured for GRPC protocol, routing traffic to ECS task IPs on port 50051. Includes gRPC-compatible health checks.
    *   **Listener** (`delegation_server_alb.tf`): Listens on port 443 (HTTPS) and uses the imported wildcard ACM certificate for TLS termination before forwarding to the target group.
*   **Container Registry:**
    *   **ECR Repository** (`delegation_server_ecr.tf`): Private Docker image registry (`cyphera-delegation-server-${stage}`) for the Delegation Server images built by CI/CD.
*   **Identity & Access Management (IAM):**
    *   **ECS Task Execution Role** (`delegation_server_ecs.tf`): Grants permissions necessary for ECS agent and Fargate to pull images, send logs, and fetch secrets/parameters for injection into the container.
*   **Configuration & Secrets References:**
    *   **SSM Parameters** (`ssm_parameters.tf`, `outputs.tf`): Creates parameters to store configuration values (CORS, URLs, wallet address) and outputs from other resources (RDS secret ARN, ALB DNS, Lambda networking IDs), making them available for Serverless and applications. Stage-specific naming is used where appropriate. Placeholders are used for values requiring manual updates post-apply, using `lifecycle { ignore_changes = [value] }` to prevent Terraform overwrites.
    *   **Secrets Manager References** (`delegation_server_ecs.tf`): Data sources reference Secrets Manager secrets for injection into ECS tasks.
*   **DNS / Certificates:**
    *   **ACM Certificate Resources** (`acm.tf`): Manages the state of existing ACM certificates (imported manually).
    *   **SSM Parameter for Cert ARN** (`ssm_parameters.tf`): Manages the state of the SSM parameter storing the imported wildcard certificate ARN, used by Serverless.

**Terraform Workflow:**

Terraform uses a state file stored securely in an S3 backend (`s3://cyphera-terraform-state/cyphera-api/terraform.tfstate`) with encryption enabled. State locking is implicitly handled by S3 via DynamoDB (if configured, recommended) or natively by S3 to prevent concurrent modifications.

1.  **Initialization:** Before running any commands in a new checkout or after configuration changes, initialize Terraform. This downloads provider plugins and configures the backend.
    ```bash
    cd infrastructure/terraform
    TF_LOG=debug GODEBUG=asyncpreemptoff=1 terraform init
    # Or, if backend config changed:
    # TF_LOG=debug GODEBUG=asyncpreemptoff=1 terraform init -reconfigure
    ```
    *Note: `TF_LOG=debug GODEBUG=asyncpreemptoff=1` provides verbose logging for troubleshooting.*
2.  **Planning:** Preview the changes Terraform will make to your infrastructure without actually applying them. Always review the plan carefully. Use `-var="stage=dev"` or `-var="stage=prod"` to target the specific environment.
    ```bash
    cd infrastructure/terraform
    TF_LOG=debug GODEBUG=asyncpreemptoff=1 terraform plan -var="stage=[dev|prod]"
    ```
3.  **Applying:** Create or update the infrastructure according to the plan. Requires confirmation unless `-auto-approve` is used (use with caution).
    ```bash
    cd infrastructure/terraform
    TF_LOG=debug GODEBUG=asyncpreemptoff=1 terraform apply -var="stage=[dev|prod]"
    ```
4.  **Importing (Manual Step for Existing Resources):** To bring existing AWS resources under Terraform management:
    *   Define the `resource` block in your `.tf` files.
    *   Run the `import` command, providing the Terraform resource address and the AWS resource ID/ARN.
        ```bash
        cd infrastructure/terraform
        # Example for SSM Parameter:
        TF_LOG=debug GODEBUG=asyncpreemptoff=1 terraform import aws_ssm_parameter.wildcard_cert_arn /cyphera/wildcard-api-cert-arn
        # Example for ACM Certificate:
        TF_LOG=debug GODEBUG=asyncpreemptoff=1 terraform import aws_acm_certificate.wildcard_api arn:aws:acm:us-east-1:[ACCOUNT_ID]:certificate/[CERT_ID]
        ```

## II. Configuration & Secrets Management (AWS SSM & Secrets Manager)

Runtime configuration and secrets are stored centrally in AWS, not in code or CI/CD variables.

*   **Secrets Manager:** Stores high-sensitivity secrets (RDS password, Delegation Server private key, Circle API key, Supabase JWT). Accessed by applications/services with specific IAM permissions. Supports automatic rotation (configured manually for RDS).
*   **SSM Parameter Store:** Stores less sensitive configuration (URLs, CORS settings, wallet address), pointers (ARNs, endpoints), and Terraform outputs needed by Serverless or applications. Parameters storing sensitive info use the `SecureString` type. Accessed with specific IAM permissions.

Terraform creates most parameters (some with placeholders requiring manual updates post-apply), while Serverless and application code read them at runtime.

## III. Deployment Pipelines (GitHub Actions)

Two main workflows automate the deployment process:

**A. Go API (`.github/workflows/cyphera-api.yml`)**

*   **Trigger:** Push/PR to `dev` or `main` branches (excluding delegation-server changes).
*   **Jobs:**
    *   `test`: Runs Go unit and integration tests (requires service dependencies like Postgres).
    *   `lint`: Runs golangci-lint.
    *   `build`: Compiles the Go binary (`bootstrap`) for AWS Lambda (provided.al2 runtime requires linux/amd64). Uploads the binary as an artifact.
    *   `deploy`:
        *   Runs only on `dev` and `main` branches.
        *   Uses the corresponding AWS Environment (`dev`/`prod`) for secrets.
        *   Downloads the build artifact.
        *   Configures AWS credentials using GitHub Secrets (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`).
        *   Installs Node.js and Serverless Framework.
        *   Runs `serverless deploy --stage [dev|prod]`. Serverless reads `serverless.yml`, interprets the `${ssm:...}` and `${secretsmanager:...}` variables to fetch configuration from AWS, packages the code, and deploys/updates the Lambda function, API Gateway, and associated resources (IAM role, custom domain).

**B. Delegation Server (`.github/workflows/delegation-server.yml`)**

*   **Trigger:** Push/PR to `dev` or `main` branches (only on changes within `delegation-server/` or the workflow file itself).
*   **Jobs:**
    *   `lint`: Runs linters (e.g., ESLint via `make`).
    *   `test`: Runs Node.js tests (e.g., Jest via `make`).
    *   `build`: Installs dependencies and builds the Node.js application (`make delegation-server-build`). Archives necessary deployment files (dist, package.json, lockfile) as an artifact (though not currently used by deploy job).
    *   `deploy`:
        *   Runs only on `dev` and `main` branches.
        *   Uses the corresponding AWS Environment (`dev`/`prod`) for secrets.
        *   Configures AWS credentials.
        *   Logs into the ECR repository created by Terraform (`cyphera-delegation-server-${stage}`).
        *   Builds the Docker image using `infrastructure/docker/delegation-server/Dockerfile` (which now uses Node 20).
        *   Tags the image with the Git SHA.
        *   Pushes the image to ECR.
        *   Runs `aws ecs update-service --force-new-deployment` targeting the ECS service created by Terraform (`cyphera-delegation-server-${stage}`). This tells ECS to pull the latest image (identified by the Git SHA tag implicitly) and deploy new tasks.

## IV. Serverless Framework (`serverless.yml`)

Defines the Go API deployment on AWS Lambda.

*   **Provider:** Configures AWS region, runtime (`provided.al2`), stage.
*   **Environment Variables:** Defines environment variables for the Lambda function. **Crucially, it uses `${ssm:...}` and `${secretsmanager:...}` syntax to resolve values from AWS Parameter Store and Secrets Manager at deployment time.** This includes DB connection details, external service URLs/keys, CORS settings, etc.
*   **IAM Role:** Defines specific IAM permissions the Lambda function needs, including `secretsmanager:GetSecretValue` and `ssm:GetParameters` for the specific secrets/parameters it reads, plus VPC and logging permissions.
*   **VPC:** Configures the Lambda to run within the private subnets and security group defined by Terraform (using stage-specific SSM lookups).
*   **Functions:** Defines the Lambda function (`main`), handler (`bootstrap`), and API Gateway HTTP API events (`/{proxy+}`).
*   **Custom Domain:** Configures the `[stage].api.cypherapay.com` custom domain, linking it to the deployed HTTP API and using the wildcard certificate ARN fetched from SSM.

## V. Overall Deployment Flow

1.  **Infrastructure Changes:** Modify Terraform code (`*.tf` files).
2.  **Terraform Plan:** Run `TF_LOG=debug GODEBUG=asyncpreemptoff=1 terraform plan -var="stage=[dev|prod]"` to review changes.
3.  **Terraform Apply:** Run `TF_LOG=debug GODEBUG=asyncpreemptoff=1 terraform apply -var="stage=[dev|prod]"` to provision/update infrastructure.
4.  **(If Necessary) Manual Config Update:** Update placeholder values in SSM Parameter Store via AWS Console (ensure `lifecycle { ignore_changes = [value] }` is set in TF for parameters updated manually).
5.  **Application Code Changes:** Modify Go API or Delegation Server code.
6.  **Push to GitHub:** Push changes to `dev` or `main` branch.
7.  **CI/CD Execution:** GitHub Actions trigger automatically:
    *   Build and test code.
    *   Deploy Go API via Serverless Framework (reading config from AWS).
    *   Build and push Delegation Server Docker image to ECR, then update ECS service.

This structured approach ensures infrastructure and application deployments are automated, repeatable, secure, and manageable across different environments.