#!/bin/bash

# Simple Web3Auth Configuration Test
# This tests the basic setup without requiring a database

echo "🔧 Web3Auth Configuration Test"
echo "==============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Load environment variables
if [ -f .env ]; then
    source .env
    echo -e "${GREEN}✅ .env file loaded${NC}"
else
    echo -e "${RED}❌ .env file not found${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}📋 Environment Variable Check${NC}"
echo "================================"

# Check Web3Auth environment variables
check_env_var() {
    local var_name="$1"
    local var_value="${!var_name}"
    
    if [ -n "$var_value" ]; then
        echo -e "${GREEN}✅ $var_name${NC}: ${var_value:0:20}..."
    else
        echo -e "${RED}❌ $var_name${NC}: NOT SET"
        return 1
    fi
}

MISSING_VARS=0

check_env_var "WEB3AUTH_CLIENT_ID" || ((MISSING_VARS++))
check_env_var "WEB3AUTH_JWKS_ENDPOINT" || ((MISSING_VARS++))
check_env_var "WEB3AUTH_ISSUER" || ((MISSING_VARS++))
check_env_var "WEB3AUTH_AUDIENCE" || ((MISSING_VARS++))

echo ""
echo -e "${BLUE}🔨 Compilation Check${NC}"
echo "===================="

if go build -o ./bin/test-web3auth ./cmd/api/local 2>/dev/null; then
    echo -e "${GREEN}✅ Golang backend compiles successfully${NC}"
    rm -f ./bin/test-web3auth
else
    echo -e "${RED}❌ Golang backend compilation failed${NC}"
    echo "Run 'go build ./cmd/api/local' to see detailed errors"
    exit 1
fi

echo ""
echo -e "${BLUE}🎫 JWT Token Generation Test${NC}"
echo "================================"

# Test JWT generation if Node.js is available
if command -v node >/dev/null 2>&1; then
    cat > /tmp/test_jwt.js << 'EOF'
const crypto = require('crypto');

function createTestJWT() {
    const header = {
        "alg": "HS256",
        "typ": "JWT"
    };
    
    const payload = {
        "sub": "web3auth_test_user_123",
        "email": "test@example.com",
        "name": "Test User",
        "verifier": "google",
        "verifierId": "google_123456789",
        "walletAddress": "0x742d35Cc6634C0532925a3b8D400E64Fa2a6b000",
        "iss": process.env.WEB3AUTH_ISSUER || "https://api-auth.web3auth.io",
        "aud": process.env.WEB3AUTH_CLIENT_ID || "test",
        "exp": Math.floor(Date.now() / 1000) + 3600,
        "iat": Math.floor(Date.now() / 1000)
    };
    
    const base64UrlEncode = (obj) => {
        return Buffer.from(JSON.stringify(obj))
            .toString('base64')
            .replace(/\+/g, '-')
            .replace(/\//g, '_')
            .replace(/=/g, '');
    };
    
    const encodedHeader = base64UrlEncode(header);
    const encodedPayload = base64UrlEncode(payload);
    const secret = process.env.WEB3AUTH_JWKS_ENDPOINT || 'test-secret';
    
    const signature = crypto
        .createHmac('sha256', secret)
        .update(`${encodedHeader}.${encodedPayload}`)
        .digest('base64')
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=/g, '');
    
    return `${encodedHeader}.${encodedPayload}.${signature}`;
}

try {
    const jwt = createTestJWT();
    console.log('SUCCESS');
    console.log('Token:', jwt.substring(0, 50) + '...');
} catch (error) {
    console.log('ERROR');
    console.log('Error:', error.message);
}
EOF

    JWT_RESULT=$(node /tmp/test_jwt.js 2>&1)
    if echo "$JWT_RESULT" | grep -q "SUCCESS"; then
        echo -e "${GREEN}✅ JWT token generation works${NC}"
        echo "Sample token: $(echo "$JWT_RESULT" | grep "Token:" | cut -d' ' -f2-)"
    else
        echo -e "${RED}❌ JWT token generation failed${NC}"
        echo "Error: $JWT_RESULT"
    fi
    
    rm -f /tmp/test_jwt.js
else
    echo -e "${YELLOW}⚠️  Node.js not found - skipping JWT generation test${NC}"
fi

echo ""
echo -e "${BLUE}🌐 Network Connectivity Test${NC}"
echo "==============================="

# Test Web3Auth JWKS endpoint connectivity
if curl -s --max-time 5 "$WEB3AUTH_JWKS_ENDPOINT" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Web3Auth JWKS endpoint is reachable${NC}"
    echo "  URL: $WEB3AUTH_JWKS_ENDPOINT"
else
    echo -e "${RED}❌ Web3Auth JWKS endpoint is NOT reachable${NC}"
    echo "  URL: $WEB3AUTH_JWKS_ENDPOINT"
    echo "  Check your internet connection or the endpoint URL"
fi

echo ""
echo -e "${BLUE}📊 Summary${NC}"
echo "==========="

if [ $MISSING_VARS -eq 0 ]; then
    echo -e "${GREEN}🎉 All environment variables are set correctly!${NC}"
else
    echo -e "${RED}💥 $MISSING_VARS environment variable(s) missing${NC}"
fi

echo ""
echo -e "${YELLOW}📋 Next Steps:${NC}"
if [ $MISSING_VARS -eq 0 ]; then
    echo "1. Start your local database (if using local setup)"
    echo "2. Run: ./bin/api-local"
    echo "3. Test authentication with:"
    echo "   curl -H \"X-API-Key: admin_valid_key\" http://localhost:8000/api/v1/admin/networks"
    echo ""
    echo "4. Or run the full integration test:"
    echo "   ./scripts/test-web3auth-integration.sh"
else
    echo "1. Fix missing environment variables in .env file"
    echo "2. Re-run this test: ./scripts/test-web3auth-simple.sh"
fi

echo ""
echo -e "${BLUE}🔍 Configuration Values (for debugging):${NC}"
echo "Client ID: ${WEB3AUTH_CLIENT_ID:0:20}..."
echo "JWKS URL: $WEB3AUTH_JWKS_ENDPOINT"
echo "Issuer: $WEB3AUTH_ISSUER"
echo "Audience: ${WEB3AUTH_AUDIENCE:0:20}..." 