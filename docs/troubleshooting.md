# Troubleshooting Guide

> **Navigation:** [← Architecture](architecture.md) | [↑ README](../README.md) | [Quick Start →](quick-start.md)

Common issues and solutions for the Cyphera platform.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Development Server Problems](#development-server-problems)
- [Database Issues](#database-issues)
- [Authentication Problems](#authentication-problems)
- [Web3 & Blockchain Issues](#web3--blockchain-issues)
- [API & gRPC Issues](#api--grpc-issues)
- [Build & Deployment Issues](#build--deployment-issues)
- [Performance Issues](#performance-issues)

## Installation Issues

### npm install fails with peer dependency errors

**Problem:** Getting ERESOLVE errors during `npm install`

**Solution:**
```bash
# Use legacy peer deps flag
npm install --legacy-peer-deps

# Or from project root
npm run install:ts
```

**Root Cause:** Web3 packages have conflicting peer dependencies, especially `ox` package versions.

### Go module download failures

**Problem:** `go mod download` fails with network errors

**Solution:**
```bash
# Set Go proxy
export GOPROXY=https://proxy.golang.org,direct

# Clear module cache
go clean -modcache

# Retry download
go mod download
```

### Docker permission errors

**Problem:** Cannot start PostgreSQL container

**Solution:**
```bash
# Fix Docker permissions (macOS/Linux)
sudo chown -R $(whoami) /var/lib/docker

# Or use Docker Desktop settings to fix permissions
```

## Development Server Problems

### Port already in use errors

**Problem:** `Error: listen EADDRINUSE: address already in use :::3000`

**Solution:**
```bash
# Find process using port
lsof -ti:3000

# Kill the process
kill -9 <PID>

# Or use different port
PORT=3001 npm run dev
```

### Air hot reload not working

**Problem:** Go API server not reloading on file changes

**Solution:**
```bash
# Check .air.toml configuration
cat apps/api/.air.toml

# Ensure file watching is enabled
# Restart with verbose logging
air -c apps/api/.air.toml -d
```

### Next.js build fails with memory errors

**Problem:** JavaScript heap out of memory during build

**Solution:**
```bash
# Increase Node.js memory limit
export NODE_OPTIONS="--max-old-space-size=4096"
npm run build

# Or use production build settings
NODE_ENV=production npm run build
```

## Database Issues

### PostgreSQL connection refused

**Problem:** `connection refused` errors when connecting to database

**Solution:**
```bash
# Check if PostgreSQL is running
docker compose ps postgres

# Start PostgreSQL
docker compose up postgres -d

# Check connection
psql $DATABASE_URL -c "SELECT 1"
```

### Database migrations fail

**Problem:** Schema initialization errors

**Solution:**
```bash
# Reset database
docker compose down postgres
docker volume rm cyphera-api_postgres_data
docker compose up postgres -d

# Wait for startup then apply schema
sleep 10
make db-migrate
```

### SQLC generation errors

**Problem:** `make gen` fails with SQL parsing errors

**Solution:**
```bash
# Check SQL syntax in query files
ls libs/go/db/queries/*.sql

# Validate specific query
sqlc verify

# Check schema compatibility
cat libs/go/db/schema.sql
```

## Authentication Problems

### Web3Auth login fails

**Problem:** Authentication redirects fail or hang

**Solution:**
```bash
# Check environment variables
echo $NEXT_PUBLIC_WEB3AUTH_CLIENT_ID

# Verify redirect URLs in Web3Auth dashboard
# Ensure CORS settings allow localhost

# Check browser console for errors
# Clear browser cache and cookies
```

### JWT token validation errors

**Problem:** API returns 401 Unauthorized with valid token

**Solution:**
```bash
# Check token expiration
# Verify Web3Auth JWKS endpoint is accessible
curl https://api.web3auth.io/jwks

# Check server logs for validation errors
docker compose logs api

# Verify workspace ID header
# X-Workspace-ID: <workspace_uuid>
```

### API key authentication fails

**Problem:** Service-to-service calls fail with API key

**Solution:**
```bash
# Verify API key format and permissions
# Check API key is not expired
# Ensure correct Authorization header format

# Test API key manually
curl -H "Authorization: sk_test_..." \
     -H "X-Workspace-ID: workspace_id" \
     http://localhost:8080/health
```

## Web3 & Blockchain Issues

### MetaMask connection problems

**Problem:** Wallet connection fails or hangs

**Solution:**
```bash
# Check browser MetaMask extension is enabled
# Verify network configuration matches app settings
# Clear MetaMask cache and reconnect

# Check Web3Auth configuration
# Ensure correct chain IDs are set
```

### Circle API integration errors

**Problem:** Circle wallet operations fail

**Solution:**
```bash
# Verify Circle API key
echo $CIRCLE_API_KEY

# Check Circle API status
curl -H "Authorization: Bearer $CIRCLE_API_KEY" \
     https://api.circle.com/v1/ping

# Review Circle API response errors in logs
```

### Delegation signature failures

**Problem:** Delegation redemption fails with signature errors

**Solution:**
```bash
# Check delegation server logs
docker compose logs delegation-server

# Verify private key configuration
# Ensure delegation hasn't expired
# Check network ID matches blockchain network

# Test delegation server directly
grpcurl -plaintext localhost:50051 list
```

### RPC endpoint issues

**Problem:** Blockchain RPC calls timeout or fail

**Solution:**
```bash
# Test RPC endpoints directly
curl -X POST -H "Content-Type: application/json" \
     --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
     $ETHEREUM_RPC_URL

# Check rate limits on RPC provider
# Switch to backup RPC endpoint
# Verify network connectivity
```

## API & gRPC Issues

### gRPC connection failures

**Problem:** Main API cannot connect to delegation server

**Solution:**
```bash
# Check delegation server is running
docker compose ps delegation-server

# Test gRPC connectivity
grpcurl -plaintext localhost:50051 list

# Check firewall/network settings
# Verify DELEGATION_GRPC_ADDR environment variable

# Check server logs for connection errors
docker compose logs api
```

### API request timeouts

**Problem:** HTTP requests timeout or return 504 errors

**Solution:**
```bash
# Check server resource usage
docker stats

# Increase timeout settings in nginx/load balancer
# Check database connection pool settings
# Monitor slow query logs

# Scale API server instances if needed
```

### CORS errors in browser

**Problem:** Cross-origin request blocked

**Solution:**
```bash
# Check CORS configuration in server.go
# Verify CORS_ALLOWED_ORIGINS environment variable
# Ensure preflight requests are handled

# For development, allow all origins temporarily
export CORS_ALLOWED_ORIGINS="*"
```

## Build & Deployment Issues

### Docker build failures

**Problem:** Docker image build fails with dependency errors

**Solution:**
```bash
# Clear Docker build cache
docker system prune -a

# Build with no cache
docker build --no-cache -t cyphera-api .

# Check Dockerfile for syntax errors
# Verify all files are in build context
```

### Lambda deployment errors

**Problem:** AWS Lambda deployment fails

**Solution:**
```bash
# Check Lambda package size
ls -la lambda-deployment.zip

# Verify binary is built for Linux
GOOS=linux GOARCH=amd64 go build

# Check Lambda configuration
# Memory, timeout, environment variables
# Review CloudWatch logs for runtime errors
```

### Environment variable issues

**Problem:** Configuration not loading in production

**Solution:**
```bash
# Verify environment variables are set
env | grep CYPHERA

# Check AWS Secrets Manager permissions
# Verify parameter names match code expectations
# Check for typos in environment variable names

# Test configuration loading
LOG_LEVEL=debug ./cyphera-api
```

## Performance Issues

### Slow API responses

**Problem:** API endpoints taking too long to respond

**Solution:**
```bash
# Check database query performance
# Add indexes for frequently queried columns
# Review slow query logs

# Monitor database connection pool
# Consider query optimization
# Add response caching where appropriate
```

### High memory usage

**Problem:** Services consuming excessive memory

**Solution:**
```bash
# Monitor memory usage
docker stats

# Check for memory leaks in Go code
go tool pprof http://localhost:6060/debug/pprof/heap

# Optimize database queries
# Adjust garbage collection settings
# Scale horizontally if needed
```

### Database connection pool exhaustion

**Problem:** "too many connections" errors

**Solution:**
```bash
# Check current connections
SELECT count(*) FROM pg_stat_activity;

# Adjust connection pool settings
# Implement connection pooling
# Close unused connections properly

# Monitor connection patterns
# Consider read replicas for read queries
```

## Getting Help

### Enabling Debug Logging

```bash
# Enable debug logging for all services
export LOG_LEVEL=debug

# Start services with verbose output
npm run dev:all

# Check specific service logs
docker compose logs -f api
docker compose logs -f delegation-server
```

### Collecting Diagnostic Information

```bash
# System information
uname -a
node --version
go version
docker --version

# Service status
docker compose ps
curl http://localhost:8080/health
curl http://localhost:3000/api/health

# Network connectivity
ping google.com
nslookup api.circle.com
```

### Common Log Locations

- **API Server:** `docker compose logs api`
- **Web App:** `apps/web-app/logs/`
- **Delegation Server:** `docker compose logs delegation-server`
- **PostgreSQL:** `docker compose logs postgres`
- **Browser Console:** F12 → Console tab

### Creating Bug Reports

When creating bug reports, include:

1. **Environment Details:**
   - Operating system and version
   - Node.js and Go versions
   - Docker version

2. **Steps to Reproduce:**
   - Exact commands run
   - Configuration used
   - Expected vs actual behavior

3. **Log Output:**
   - Error messages
   - Stack traces
   - Relevant log entries

4. **Environment Variables:**
   - Sanitized configuration (no secrets)
   - Network settings
   - Service versions

---

## Related Documentation

- **[Quick Start Guide](quick-start.md)** - Initial setup
- **[Architecture Guide](architecture.md)** - System overview
- **[API Reference](api-reference.md)** - API documentation
- **[Contributing Guide](contributing.md)** - Development workflow

## Support Channels

- **[GitHub Issues](https://github.com/your-org/cyphera-api/issues)** - Bug reports
- **[GitHub Discussions](https://github.com/your-org/cyphera-api/discussions)** - Questions
- **Documentation Updates** - Submit PRs for documentation improvements

---

*Last updated: $(date '+%Y-%m-%d')*
*Need to add a new troubleshooting section? [Submit a PR](../CONTRIBUTING.md)*