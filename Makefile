.PHONY: all build test clean lint run swagger deploy test-all test-integration stop-integration-server ensure-executable

# Go parameters
BINARY_NAME=cyphera-api
MAIN_PACKAGE=./cmd/api/main
GO=go

all: lint test build

build:
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

test:
	$(GO) test -v ./...

# Run all tests including integration tests
test-all: test test-integration

# Ensure scripts are executable
ensure-executable:
	@chmod +x scripts/integration-test.sh

# Run integration tests with the delegation server
test-integration: ensure-executable
	@echo "Running delegation integration tests with mock server..."
	DELEGATION_LOCAL_MODE=true ./scripts/integration-test.sh --cli

# Stop integration server processes if they're still running
stop-integration-server:
	@echo "Stopping integration test server processes..."
	-@pkill -f "npm run start:mock" 2>/dev/null || true
	@echo "Checking for processes using port 50051 (gRPC)..."
	-@lsof -ti:50051 | xargs kill -9 2>/dev/null || true
	@echo "Servers stopped"

clean:
	$(GO) clean
	rm -f bin/$(BINARY_NAME)

lint:
	$(GO) vet ./...
	gofmt -l .

run:
	$(GO) run $(MAIN_PACKAGE)

swagger:
	swag init -g cmd/api/main/main.go

deploy:
	# Add deployment steps here

dev:
	./scripts/start-dev.sh

# Individual server commands (used directly by the start-dev.sh script)
api-server:
	$(GO) run $(MAIN_PACKAGE)

delegation-server:
	cd delegation-server && npm run start:mock

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  make all            - Run default targets: lint, test, and build"
	@echo "  make build          - Build the binary"
	@echo "  make test           - Run all unit tests"
	@echo "  make test-all       - Run all tests, including integration tests"
	@echo "  make test-integration - Run integration tests with mock server"
	@echo "  make stop-integration-server - Stop integration server processes"
	@echo "  make clean          - Clean build files"
	@echo "  make lint           - Run linter"
	@echo "  make run            - Run the application"
	@echo "  make swagger        - Generate Swagger documentation"
	@echo "  make deploy         - Deploy the application"
	@echo "  make dev            - Run the application in development mode (loads .env)"
	@echo "  make delegation-server - Run the delegation server" 
	@echo "  make gen            - Generate SQLC code"
gen:
	sqlc generate