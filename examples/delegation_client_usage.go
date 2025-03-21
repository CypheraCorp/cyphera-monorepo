package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cyphera-api/internal/client"
)

// ExampleDelegationClient demonstrates how to use the delegation client directly
func ExampleDelegationClient() {
	// Create a new delegation client
	delegationClient, err := client.NewDelegationClient()
	if err != nil {
		log.Printf("Failed to create delegation client: %v", err)
		return
	}
	defer delegationClient.Close()

	// Create a sample delegation (in a real application, this would come from your database)
	delegation := map[string]interface{}{
		"delegate":  "0x1234567890123456789012345678901234567890",
		"delegator": "0x0987654321098765432109876543210987654321",
		"authority": "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		"caveats":   []map[string]interface{}{},
		"salt":      "0x123456789",
		"signature": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}

	// Convert the delegation to JSON
	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		log.Printf("Failed to serialize delegation: %v", err)
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Redeem the delegation
	txHash, err := delegationClient.RedeemDelegation(ctx, delegationJSON)
	if err != nil {
		log.Printf("Failed to redeem delegation: %v", err)
		return
	}

	fmt.Println("Delegation successfully redeemed! Transaction hash:", txHash)
	// Output: Delegation successfully redeemed! Transaction hash: 0x123...
}

// ExampleDelegationBatchProcessing demonstrates how to batch process multiple delegations
func ExampleDelegationBatchProcessing() {
	// Create a new delegation client
	delegationClient, err := client.NewDelegationClient()
	if err != nil {
		log.Printf("Failed to create delegation client: %v", err)
		return
	}
	defer delegationClient.Close()

	// Sample delegations to process (in a real application, these would come from your database)
	delegations := []map[string]interface{}{
		{
			"delegate":  "0x1234567890123456789012345678901234567890",
			"delegator": "0x0987654321098765432109876543210987654321",
			"authority": "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
			"caveats":   []map[string]interface{}{},
			"salt":      "0x123456789",
			"signature": "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
		{
			"delegate":  "0x2345678901234567890123456789012345678901",
			"delegator": "0x1098765432109876543210987654321098765432",
			"authority": "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
			"caveats":   []map[string]interface{}{},
			"salt":      "0x234567890",
			"signature": "0xbcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		},
	}

	// Process each delegation
	for i, delegation := range delegations {
		delegationJSON, err := json.Marshal(delegation)
		if err != nil {
			log.Printf("Failed to serialize delegation %d: %v", i, err)
			continue
		}

		// Create a context with timeout for each request
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Redeem the delegation
		txHash, err := delegationClient.RedeemDelegation(ctx, delegationJSON)
		if err != nil {
			log.Printf("Failed to redeem delegation %d: %v", i, err)
			cancel()
			continue
		}

		fmt.Println("Delegation", i, "successfully redeemed! Transaction hash:", txHash)
		cancel()
	}
	// Output:
	// Delegation 0 successfully redeemed! Transaction hash: 0x123...
	// Delegation 1 successfully redeemed! Transaction hash: 0x456...
}
