package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"go.uber.org/zap"

	"github.com/cyphera/cyphera-api/libs/go/logger" // Assuming logger setup
)

// SecretsManagerClient wraps the AWS Secrets Manager client.
type SecretsManagerClient struct {
	svc *secretsmanager.Client
	cfg aws.Config
}

// NewSecretsManagerClient creates and initializes a new Secrets Manager client.
// It uses the default AWS configuration chain (environment variables, shared config, IAM role).
func NewSecretsManagerClient(ctx context.Context) (*SecretsManagerClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	svc := secretsmanager.NewFromConfig(cfg)

	return &SecretsManagerClient{
		svc: svc,
		cfg: cfg,
	}, nil
}

// GetSecretString fetches a secret string from AWS Secrets Manager using an ARN specified by an environment variable.
// If the ARN environment variable (secretArnEnvVar) is not set or fetching fails,
// it falls back to reading the secret directly from another environment variable (fallbackEnvVar).
// It intelligently handles secrets stored as plain text OR as a JSON object with a single key
// (where the value associated with that key is the desired secret).
// It returns the secret value or an error if both methods fail.
func (c *SecretsManagerClient) GetSecretString(ctx context.Context, secretArnEnvVar string, fallbackEnvVar string) (string, error) {
	secretArn := os.Getenv(secretArnEnvVar)

	// Attempt to fetch from Secrets Manager if ARN is provided
	if secretArn != "" {
		logger.Log.Debug("Attempting to fetch secret from Secrets Manager", zap.String("arnEnvVar", secretArnEnvVar), zap.String("secretArn", secretArn))
		input := &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretArn),
		}

		result, err := c.svc.GetSecretValue(ctx, input)
		// Process if fetch was successful AND secret string is not nil/empty
		if err == nil && result.SecretString != nil && *result.SecretString != "" {
			fetchedSecretString := *result.SecretString

			// --- Try parsing as JSON with a single key ---
			var secretJSON map[string]string
			jsonErr := json.Unmarshal([]byte(fetchedSecretString), &secretJSON)

			// If JSON parsing worked AND there's exactly one key
			if jsonErr == nil && len(secretJSON) == 1 {
				// Extract the value from the single key
				for key, value := range secretJSON {
					logger.Log.Info("Successfully fetched secret from Secrets Manager (extracted from single-key JSON)",
						zap.String("secretArn", secretArn),
						zap.String("jsonKey", key),
					)
					return value, nil // Return the extracted value
				}
			}
			// --- End JSON parsing attempt ---

			// If it wasn't single-key JSON or parsing failed, assume it's plain text
			if jsonErr != nil {
				logger.Log.Info("Successfully fetched secret from Secrets Manager (treated as plain text)", zap.String("secretArn", secretArn))
			} else {
				// JSON parsing succeeded but wasn't single-key format
				logger.Log.Warn("Fetched secret from Secrets Manager was JSON but not single-key format, returning raw JSON string",
					zap.String("secretArn", secretArn),
					zap.Int("keyCount", len(secretJSON)),
				)
			}
			return fetchedSecretString, nil // Return the raw fetched string

		}
		// Log the Secrets Manager fetch error but continue to fallback
		logger.Log.Warn("Failed to retrieve secret from Secrets Manager, falling back to env var",
			zap.String("secretArnEnvVar", secretArnEnvVar),
			zap.String("secretArn", secretArn),
			zap.String("fallbackEnvVar", fallbackEnvVar),
			zap.Error(err), // Include the error from Secrets Manager
		)
	} else {
		logger.Log.Debug("Secret ARN environment variable not set, falling back to direct env var",
			zap.String("arnEnvVar", secretArnEnvVar),
			zap.String("fallbackEnvVar", fallbackEnvVar),
		)
	}

	// Fallback to direct environment variable
	secretValue := os.Getenv(fallbackEnvVar)
	if secretValue != "" {
		logger.Log.Info("Using secret value from direct environment variable", zap.String("envVar", fallbackEnvVar))
		return secretValue, nil
	}

	// If both methods fail
	logger.Log.Error("Failed to retrieve secret from both Secrets Manager and direct environment variable",
		zap.String("arnEnvVar", secretArnEnvVar),
		zap.String("fallbackEnvVar", fallbackEnvVar),
	)
	return "", fmt.Errorf("secret not found using ARN env var '%s' or direct env var '%s'", secretArnEnvVar, fallbackEnvVar)
}

// GetSecretJSON fetches a secret from AWS Secrets Manager and unmarshals it into the provided struct.
// It expects the secret stored in Secrets Manager to be a JSON string.
// Falls back to os.Getenv(fallbackEnvVar) if ARN is not set or fetch fails, but assumes the fallback is NOT JSON.
// This is specifically tailored for the RDS secret format.
func (c *SecretsManagerClient) GetSecretJSON(ctx context.Context, secretArnEnvVar string, fallbackEnvVar string, target interface{}) error {
	secretArn := os.Getenv(secretArnEnvVar)
	if secretArn != "" {
		logger.Log.Debug("Attempting to fetch JSON secret from Secrets Manager", zap.String("arnEnvVar", secretArnEnvVar), zap.String("secretArn", secretArn))
		input := &secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretArn),
		}

		result, err := c.svc.GetSecretValue(ctx, input)
		if err == nil && result.SecretString != nil {
			err = json.Unmarshal([]byte(*result.SecretString), target)
			if err == nil {
				logger.Log.Info("Successfully fetched and parsed JSON secret from Secrets Manager", zap.String("secretArn", secretArn))
				return nil // Success
			}
			logger.Log.Warn("Failed to unmarshal JSON secret from Secrets Manager, falling back",
				zap.String("secretArn", secretArn),
				zap.Error(err),
			)
			// Fall through to fallback if JSON parsing fails
		} else {
			logger.Log.Warn("Failed to retrieve secret from Secrets Manager, falling back",
				zap.String("secretArn", secretArn),
				zap.Error(err), // Include the error from Secrets Manager
			)
			// Fall through to fallback
		}

	} else {
		logger.Log.Debug("Secret ARN environment variable not set, falling back", zap.String("arnEnvVar", secretArnEnvVar))
		// Fall through to fallback
	}

	// Fallback logic (assumes fallbackEnvVar holds a DSN, not JSON)
	// If you intend the fallback to also be JSON, this needs modification.
	fallbackValue := os.Getenv(fallbackEnvVar)
	if fallbackValue != "" {
		// Cannot unmarshal a DSN into the target struct designed for JSON.
		// This indicates a configuration mismatch if fallback is needed for RDS secret.
		logger.Log.Error("Fallback needed for JSON secret, but fallback value is not JSON",
			zap.String("arnEnvVar", secretArnEnvVar),
			zap.String("fallbackEnvVar", fallbackEnvVar),
		)
		return fmt.Errorf("secrets Manager fetch failed for %s, and fallback %s is not JSON parsable", secretArnEnvVar, fallbackEnvVar)
	}

	logger.Log.Error("Failed to retrieve JSON secret from Secrets Manager and no fallback available",
		zap.String("arnEnvVar", secretArnEnvVar),
		zap.String("fallbackEnvVar", fallbackEnvVar),
	)
	return fmt.Errorf("secret not found or parsable using ARN env var '%s' or direct env var '%s'", secretArnEnvVar, fallbackEnvVar)
}
