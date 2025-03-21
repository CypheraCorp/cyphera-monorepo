.PHONY: all build test clean lint run swagger deploy

# Go parameters
BINARY_NAME=cyphera-api
MAIN_PACKAGE=./cmd/api
GO=go

all: lint test build

build:
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)/main

test:
	$(GO) test -v ./...

clean:
	$(GO) clean
	rm -f bin/$(BINARY_NAME)

lint:
	golangci-lint run

run-local:
	./scripts/local.sh

swag:
	swag init -g cmd/api/main/main.go

sqlc:
	sqlc generate

gen: sqlc swag

# Development tasks
.PHONY: dev

air:
	air

env:
	@if [ -f .env.local ]; then \
		export $$(cat .env.local | grep -v '^#' | xargs); \
	else \
		export $$(cat .env | grep -v '^#' | xargs); \
	fi

dev: swagger air

local:
	./scripts/local.sh

# Docker tasks
docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run -p 8000:8000 $(BINARY_NAME)

# Help command
help:
	@echo "Available commands:"
	@echo "  make build         - Build the application"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build files"
	@echo "  make lint         - Run linter"
	@echo "  make run          - Run the application"
	@echo "  make swagger      - Generate swagger documentation"
	@echo "  make dev          - Run development mode"
	@echo "  make docker-build - Build docker image"
	@echo "  make docker-run   - Run docker container"
	@echo "  make deploy       - Deploy the application"

deploy:
	serverless deploy

clean:
	$(GO) clean
	rm -f bin/$(BINARY_NAME)
	rm -f bootstrap function.zip

# Generate gRPC code
gen-grpc:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/proto/delegation.proto

# Install required Go modules for gRPC
install-grpc-deps:
	go get google.golang.org/grpc
	go get google.golang.org/protobuf/cmd/protoc-gen-go
	go get google.golang.org/grpc/cmd/protoc-gen-go-grpc 