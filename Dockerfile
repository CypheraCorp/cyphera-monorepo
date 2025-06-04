FROM golang:1.23-alpine

WORKDIR /app

# Install git, build dependencies, and swag
RUN apk add --no-cache git build-base
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Generate swagger documentation
RUN swag init --dir ./internal/handlers --generalInfo ../../cmd/api/main/main.go --output ./docs --tags='!exclude'

# Set environment variables
ENV GIN_MODE=debug

# Build the application
RUN go build -o cyphera-api ./cmd/api/local

# Expose port 8000
EXPOSE 8000

# Run the application
CMD ["./cyphera-api"]