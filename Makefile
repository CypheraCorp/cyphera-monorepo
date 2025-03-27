.PHONY: all build test clean lint run swagger deploy test-all test-integration stop-integration-server ensure-executable proto-build-go proto-build-js proto-build-all subscription-processor

# Go parameters
BINARY_NAME=cyphera-api
MAIN_PACKAGE=./cmd/api/main
GO=go

all: lint test build

build:
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

// TODO: no unit tests are running in the delegation-server
test:
	$(GO) test -v ./...

# Run all tests including integration tests
test-all: test test-integration

# Ensure scripts are executable
ensure-executable:
	@chmod +x scripts/integration-test.sh
	@chmod +x scripts/start-dev.sh

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
	rm -f bin/cyphera-api-dev
	rm -rf bin/air*

lint:
	$(GO) vet ./...
	gofmt -l .

run:
	$(GO) run $(MAIN_PACKAGE)

swagger:
	swag init -g cmd/api/main/main.go

deploy:
	# Add deployment steps here

# Run the subscription processor
subscription-processor:
	@echo "Starting subscription processor with 1-minute interval..."
	$(GO) run ./cmd/subscription-processor/main.go --interval=1m

dev: ensure-executable
	@echo "Starting development environment with hot reloading for the API server..."
	./scripts/start-dev.sh

# Build proto definitions for Go
proto-build-go:
	@echo "Generating Go gRPC code from proto definitions..."
	protoc --go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		internal/proto/delegation.proto
	@echo "✅ Go gRPC code successfully generated in internal/proto/"

# Build proto definitions for Node.js delegation server
proto-build-js:
	@echo "Generating Node.js gRPC code from proto definitions..."
	cd delegation-server && npm run proto:build
	@echo "✅ Node.js gRPC code successfully generated in delegation-server/src/proto/"

# Build all proto definitions
proto-build-all: proto-build-go proto-build-js
	@echo "✅ All gRPC code successfully generated"

# Individual server commands (used directly by the start-dev.sh script)
api-server:
	$(GO) run $(MAIN_PACKAGE)

api-server-air:
	air

delegation-server:
	cd delegation-server && npm run dev

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
	@echo "  make api-server     - Run the API server without live reload"
	@echo "  make api-server-air - Run the API server with air for live reload"
	@echo "  make delegation-server - Run the delegation server" 
	@echo "  make subscription-processor - Run the subscription processor"
	@echo "  make gen            - Generate SQLC code"
	@echo "  make proto-build-go - Generate Go gRPC code from proto definitions"
	@echo "  make proto-build-js - Generate Node.js gRPC code from proto definitions"
	@echo "  make proto-build-all - Generate both Go and Node.js gRPC code"
gen:
	sqlc generate