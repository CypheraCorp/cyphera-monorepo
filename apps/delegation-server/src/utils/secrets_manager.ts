import { GetSecretValueCommand, SecretsManagerClient } from "@aws-sdk/client-secrets-manager";
import { logger } from "./utils"; // Assuming logger is in utils

const secretsClient = new SecretsManagerClient({}); // Uses default credential chain (environment, shared config, IAM role)

/**
 * Fetches a secret string from AWS Secrets Manager using an ARN specified by an environment variable.
 * If the ARN environment variable (secretArnEnvVar) is not set or fetching fails,
 * it falls back to reading the secret directly from another environment variable (fallbackEnvVar).
 * It intelligently handles secrets stored as plain text OR as a JSON object with a single key
 * (where the value associated with that key is the desired secret).
 *
 * @param secretArnEnvVar The name of the environment variable holding the AWS Secrets Manager ARN.
 * @param fallbackEnvVar The name of the environment variable holding the direct secret value as a fallback.
 * @returns Promise<string> The secret value.
 * @throws Error if the secret cannot be retrieved from either source.
 */
export async function getSecretValue(secretArnEnvVar: string, fallbackEnvVar: string): Promise<string> {
  logger.info("secretArnEnvVar: ", secretArnEnvVar)
  const secretArn = process.env[secretArnEnvVar];
  logger.info("secretArn: ", secretArn)
  // Attempt to fetch from Secrets Manager if ARN is provided
  if (secretArn) {
    logger.info("secretArn is not empty")
    logger.debug(`Attempting to fetch secret from AWS Secrets Manager for ${secretArnEnvVar}`, { secretArn });
    try {
      logger.info("Attempting to fetch secret from AWS Secrets Manager")
      const command = new GetSecretValueCommand({ SecretId: secretArn });
      logger.info("command: ", command)
      const result = await secretsClient.send(command);
      logger.info("result: ", result)

      if (result.SecretString) {
        logger.info("result.SecretString is not empty")
        const fetchedSecretString = result.SecretString;

        // Try parsing as JSON with a single key
        try {
          logger.info("Attempting to parse as JSON with a single key")
          const secretJSON = JSON.parse(fetchedSecretString);
          if (typeof secretJSON === 'object' && secretJSON !== null) {
            const keys = Object.keys(secretJSON);
            if (keys.length === 1 && typeof secretJSON[keys[0]] === 'string') {
              logger.info(`Successfully fetched and extracted secret from Secrets Manager (single-key JSON) for ${secretArnEnvVar}`, { jsonKey: keys[0] });
              return secretJSON[keys[0]];
            }
          }
          logger.info("Not a JSON or not the expected single-key JSON format, treating as plain text")
        } catch (jsonError) {
          logger.info("Error parsing JSON: ", jsonError)
          // Not a JSON or not the expected single-key JSON format, treat as plain text
          logger.debug(`Secret for ${secretArnEnvVar} is not single-key JSON, treating as plain text.`);
        }
        
        // If it wasn't single-key JSON or parsing failed, assume it's plain text
        logger.info(`Successfully fetched secret from Secrets Manager (plain text) for ${secretArnEnvVar}`);
        return fetchedSecretString;
      }
      logger.info("result.SecretString is empty")
      logger.warn(`SecretString is empty from Secrets Manager for ${secretArnEnvVar}, falling back.`);
    } catch (error) {
      logger.info("Error fetching secret from Secrets Manager: ", error)
      logger.warn(`Failed to retrieve secret from Secrets Manager for ${secretArnEnvVar}, falling back to env var ${fallbackEnvVar}.`, { error, secretArn });
      // Fall through to fallback
    }
  } else {
    logger.info("secretArn is empty")
    logger.debug(`Secret ARN environment variable '${secretArnEnvVar}' not set, falling back to direct env var '${fallbackEnvVar}'.`);
  }

  // Fallback to direct environment variable
  const fallbackSecretValue = process.env[fallbackEnvVar];
  if (fallbackSecretValue) {
    logger.info(`Using secret value from direct environment variable '${fallbackEnvVar}'.`);
    return fallbackSecretValue;
  }

  const errorMessage = `Failed to retrieve secret: '${secretArnEnvVar}' (Secrets Manager) and '${fallbackEnvVar}' (direct env var) are both undefined or failed.`;
  logger.error(errorMessage);
  throw new Error(errorMessage);
} 