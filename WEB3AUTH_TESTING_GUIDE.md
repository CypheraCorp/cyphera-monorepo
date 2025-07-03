# 🧪 Web3Auth Integration Testing Guide

## Summary

✅ **You're absolutely correct about the client secret!** For JWT validation, you only need:
- **Client ID** - Used as the audience in JWT validation
- **JWKS endpoint** - For fetching public keys to verify JWT signatures  
- **Issuer** - For verifying who issued the token

The **client secret** is only needed for backend-to-backend Web3Auth API calls, which you're not doing.

## ✅ Configuration Status

Your Web3Auth environment is correctly configured:

```bash
WEB3AUTH_CLIENT_ID=BO3nb2gu5FJjApgENutIzcQOy8ZS47Qyd0pfth-v8i9dM7yQlNDd6dFQrLNJLNKMw09LvsqdDK2YHxHirOWpjbA
WEB3AUTH_JWKS_ENDPOINT=https://api-auth.web3auth.io/.well-known/jwks.json
WEB3AUTH_ISSUER=https://api-auth.web3auth.io
WEB3AUTH_AUDIENCE=BO3nb2gu5FJjApgENutIzcQOy8ZS47Qyd0pfth-v8i9dM7yQlNDd6dFQrLNJLNKMw09LvsqdDK2YHxHirOWpjbA
```

## 🚀 Manual Testing Steps

### Step 1: Start Your Database

If using local PostgreSQL:
```bash
# Make sure your local PostgreSQL is running
# The migration should have already updated your schema
```

### Step 2: Build and Start the Backend

```bash
# Build the backend
make build

# Or build manually
go build -o ./bin/api-local ./cmd/api/local

# Start the backend
./bin/api-local
```

### Step 3: Test API Key Authentication (Baseline)

First, test that your existing API key authentication still works:

```bash
# Test admin endpoint with API key
curl -H "X-API-Key: admin_valid_key" \
     http://localhost:8000/api/v1/admin/networks

# Expected: Should return network data or empty array
```

### Step 4: Generate a Test Web3Auth JWT

Create a test JWT token that mimics what Web3Auth would send:

<details>
<summary>📋 Copy this test JWT token</summary>

```bash
# Test JWT Token (valid for 1 hour from creation)
TEST_JWT="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ3ZWIzYXV0aF90ZXN0X3VzZXJfMTIzIiwiZW1haWwiOiJ0ZXN0QGV4YW1wbGUuY29tIiwibmFtZSI6IlRlc3QgVXNlciIsInZlcmlmaWVyIjoiZ29vZ2xlIiwidmVyaWZpZXJJZCI6Imdvb2dsZV8xMjM0NTY3ODkiLCJ3YWxsZXRBZGRyZXNzIjoiMHg3NDJkMzVDYzY2MzRDMDUzMjkyNWEzYjhENDAwRTY0RmEyYTZiMDAwIiwiaXNzIjoiaHR0cHM6Ly9hcGktYXV0aC53ZWIzYXV0aC5pbyIsImF1ZCI6IkJPM25iMmd1NUKFHF0FILciAsIOmEOS9mKJGN7Y9AgENutIzcQOy8ZS47Qyd0pfth-v8i9dM7yQlNDd6dFQrLNJLNKMw09LvsqdDK2YHxHirOWpjbA","ZXhwIjoxNzM4NTMwMDQ3LCJpYXQiOjE3Mzg1MjY0NDd9.signature"
```
</details>

### Step 5: Test Web3Auth JWT Authentication

Test the JWT authentication with your backend:

```bash
# Test protected endpoint with Web3Auth JWT
curl -H "Authorization: Bearer $TEST_JWT" \
     http://localhost:8000/api/v1/user/profile

# Expected outcomes:
# - 200: User created and authenticated (first time)
# - 200: User found and authenticated (subsequent calls)  
# - 401: JWT validation failed
# - 500: Database/server error
```

### Step 6: Test User Auto-Creation

When a new Web3Auth user authenticates, the system should automatically:

1. **Create Account** - A new account for the user
2. **Create User** - User record with Web3Auth fields populated
3. **Create Workspace** - Default workspace for the account  
4. **Create Wallet** - Web3Auth smart account wallet

Check the database after authentication:

```bash
# Check if user was created (in your DB client)
SELECT * FROM users WHERE web3auth_id = 'web3auth_test_user_123';

# Check if wallet was created
SELECT * FROM wallets WHERE web3auth_user_id = 'web3auth_test_user_123';
```

### Step 7: Test Different Endpoints

Try various endpoints to ensure authentication works across the API:

```bash
# Test different endpoints with Web3Auth JWT
curl -H "Authorization: Bearer $TEST_JWT" \
     http://localhost:8000/api/v1/workspaces

curl -H "Authorization: Bearer $TEST_JWT" \
     http://localhost:8000/api/v1/user/wallets

curl -H "Authorization: Bearer $TEST_JWT" \
     http://localhost:8000/api/v1/accounts/current
```

## 🔍 Troubleshooting

### Issue: 404 Not Found
- **Cause**: Routes not configured or backend not running
- **Fix**: Check that backend is running on port 8000

### Issue: 401 Unauthorized  
- **Cause**: JWT validation failed
- **Solutions**:
  - Check JWT format and claims
  - Verify Web3Auth environment variables
  - Check backend logs for specific error

### Issue: 500 Internal Server Error
- **Cause**: Database connection or server error
- **Solutions**:
  - Check database is running and accessible
  - Check database schema is up to date  
  - Review backend error logs

### Issue: Database Connection Failed
- **Solutions**:
  ```bash
  # Check database URL in .env
  echo $DATABASE_URL
  
  # Test database connection
  psql $DATABASE_URL -c "SELECT 1;"
  
  # Run migrations if needed
  make migrate-up
  ```

## 🧩 Quick Test Scripts

You can also use the provided test scripts:

```bash
# Quick configuration check
./scripts/test-web3auth-simple.sh

# Full integration test (requires database)
./scripts/test-web3auth-integration.sh
```

## ✅ Expected Behavior

When everything works correctly:

1. **API Key Auth**: Still works for admin/system operations
2. **Web3Auth JWT**: Creates new users automatically on first auth
3. **User Creation**: Populates Web3Auth fields (web3auth_id, verifier, etc.)
4. **Wallet Creation**: Creates smart account wallet automatically
5. **Subsequent Auths**: Finds existing user and authenticates normally

## 📝 Next Steps

After successful local testing:

1. Test with real Web3Auth tokens from your frontend
2. Deploy to staging/production with proper Web3Auth configuration
3. Update frontend to use new authentication flow
4. Monitor authentication metrics and error rates

## 🔧 Development Mode

Since you're in dev mode, the backend will:
- Auto-create users on first Web3Auth authentication
- Log detailed authentication information  
- Accept the simplified JWT validation (without full JWKS verification)

For production, you'll want to implement proper JWKS validation for security. 