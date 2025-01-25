.PHONY: all build test clean lint run swagger

# Go parameters
BINARY_NAME=cyphera-api
MAIN_PACKAGE=./cmd/api
GO=go

all: lint test build

build:
	$(GO) build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

test:
	$(GO) test -v ./...

clean:
	$(GO) clean
	rm -f bin/$(BINARY_NAME)

lint:
	golangci-lint run

run:
	$(GO) run $(MAIN_PACKAGE)

swagger:
	swag init -g cmd/api/main.go

# Development tasks
dev: swagger build run

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