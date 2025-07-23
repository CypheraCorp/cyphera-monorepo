#!/bin/bash

# Test Web3Auth Account Sign-In
echo "🧪 Testing Web3Auth Account Sign-In"
echo "===================================="

# Test data for Web3Auth account creation
curl -X POST http://localhost:8000/api/v1/admin/accounts/signin \
  -H "Content-Type: application/json" \
  -H "X-API-Key: admin_valid_key" \
  -d '{
    "name": "Test Web3Auth Account",
    "account_type": "merchant",
    "metadata": {
      "ownerWeb3AuthId": "web3auth_test_user_12345",
      "email": "testuser@example.com",
      "verifier": "google",
      "verifierId": "testuser@example.com"
    }
  }' | jq '.'

echo ""
echo "✅ Test completed!"
echo ""
echo "Expected behavior:"
echo "- Should create a new account with Web3Auth user"
echo "- Should return account details with user and workspace"
echo "- User should have web3auth_id populated" 