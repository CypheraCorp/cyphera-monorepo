package payment_sync

import (
	"crypto/rand"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookSecretEncryption(t *testing.T) {
	// Create a test encryption key
	encryptionKey := make([]byte, 32)
	_, err := rand.Read(encryptionKey)
	require.NoError(t, err)

	// Create payment sync client with test key
	logger := zap.NewNop()
	client := NewPaymentSyncClient(nil, logger, string(encryptionKey))

	testCases := []struct {
		name          string
		webhookSecret string
		expectEmpty   bool
	}{
		{
			name:          "Valid webhook secret",
			webhookSecret: "whsec_test_stripe_webhook_secret_12345",
			expectEmpty:   false,
		},
		{
			name:          "Empty webhook secret",
			webhookSecret: "",
			expectEmpty:   true,
		},
		{
			name:          "Complex webhook secret",
			webhookSecret: "whsec_1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$%^&*()",
			expectEmpty:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test encryption
			encrypted := client.encryptWebhookSecret(tc.webhookSecret)

			if tc.expectEmpty {
				assert.Empty(t, encrypted, "Empty webhook secret should result in empty encrypted string")
				return
			}

			assert.NotEmpty(t, encrypted, "Webhook secret should be encrypted")
			assert.NotEqual(t, tc.webhookSecret, encrypted, "Encrypted secret should not equal original")

			// Test decryption
			decrypted := client.decryptWebhookSecret(encrypted)
			assert.Equal(t, tc.webhookSecret, decrypted, "Decrypted secret should match original")
		})
	}
}

func TestWebhookSecretEncryptionConsistency(t *testing.T) {
	// Create a test encryption key
	encryptionKey := make([]byte, 32)
	_, err := rand.Read(encryptionKey)
	require.NoError(t, err)

	// Create payment sync client with test key
	logger := zap.NewNop()
	client := NewPaymentSyncClient(nil, logger, string(encryptionKey))

	testSecret := "whsec_test_consistency_check"

	// Encrypt the same secret multiple times
	encrypted1 := client.encryptWebhookSecret(testSecret)
	encrypted2 := client.encryptWebhookSecret(testSecret)

	// Encrypted values should be different (due to random nonce)
	assert.NotEqual(t, encrypted1, encrypted2, "Multiple encryptions should produce different ciphertext")

	// But both should decrypt to the same original value
	decrypted1 := client.decryptWebhookSecret(encrypted1)
	decrypted2 := client.decryptWebhookSecret(encrypted2)

	assert.Equal(t, testSecret, decrypted1, "First decryption should match original")
	assert.Equal(t, testSecret, decrypted2, "Second decryption should match original")
}

func TestWebhookSecretDecryptionFailure(t *testing.T) {
	// Create a test encryption key
	encryptionKey := make([]byte, 32)
	_, err := rand.Read(encryptionKey)
	require.NoError(t, err)

	// Create payment sync client with test key
	logger := zap.NewNop()
	client := NewPaymentSyncClient(nil, logger, string(encryptionKey))

	testCases := []struct {
		name      string
		encrypted string
	}{
		{
			name:      "Invalid base64",
			encrypted: "invalid_base64_@#$%",
		},
		{
			name:      "Empty string",
			encrypted: "",
		},
		{
			name:      "Too short ciphertext",
			encrypted: "YWJj", // base64 for "abc" - too short
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decrypted := client.decryptWebhookSecret(tc.encrypted)
			assert.Empty(t, decrypted, "Invalid encrypted data should return empty string")
		})
	}
}
