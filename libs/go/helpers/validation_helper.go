package helpers

import (
	"strings"
)

// IsAddressValid checks if the provided string is a valid Ethereum address
// It verifies:
// 1. The address is exactly 42 characters long (including 0x prefix)
// 2. The address starts with "0x"
// 3. The remaining 40 characters are valid hexadecimal
func IsAddressValid(address string) bool {
	// Check length
	if len(address) != 42 {
		return false
	}

	// Check "0x" prefix
	if !strings.HasPrefix(address, "0x") {
		return false
	}

	// Check if the address contains only hex characters after the 0x prefix
	for _, c := range address[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// IsPrivateKeyValid checks if the provided string is a valid Ethereum private key
// It verifies:
// 1. The key is exactly 66 characters long (including 0x prefix)
// 2. The key starts with "0x"
// 3. The remaining 64 characters are valid hexadecimal
func IsPrivateKeyValid(key string) bool {
	// Check length (32 bytes = 64 hex chars + 2 chars for "0x")
	if len(key) != 66 {
		return false
	}

	// Check "0x" prefix
	if !strings.HasPrefix(key, "0x") {
		return false
	}

	// Check if the key contains only hex characters after the 0x prefix
	for _, c := range key[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}
