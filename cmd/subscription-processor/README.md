# Subscription Processor

The Subscription Processor is a command-line tool for processing recurring subscription payments in the Cyphera API system. It automatically identifies subscriptions that are due for renewal, processes the payments using stored delegation credentials, and updates subscription records.

## Features

- Automatically identify and process subscriptions due for payment
- Schedule regular checks using configurable intervals
- Support both one-time execution and continuous operation
- Automatic marking of completed subscriptions when their term ends
- Proper error handling with detailed logging
- Graceful shutdown on termination signals

## Prerequisites

- Go 1.23 or later
- PostgreSQL database with Cyphera API schema
- Valid environment configuration

## Configuration

### Environment Variables

The following environment variables are required:

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://user:password@localhost:5432/cyphera` |
| `CYPHERA_SMART_WALLET_ADDRESS` | The address of the Cyphera smart wallet contract | `0x1234...` |
| `DELEGATION_GRPC_ADDR` | The address of the delegation gRPC server (optional) | `localhost:50051` |
| `DELEGATION_RPC_TIMEOUT` | Timeout for RPC calls (optional) | `3m` |

### Command-Line Options

| Flag | Description | Default | Valid Values |
|------|-------------|---------|-------------|
| `--interval` | Time between subscription checks | `5m` | Any valid Go duration: `30s`, `5m`, `1h`, `2h30m` |
| `--once` | Run once and exit | `false` | Flag presence enables one-time mode |
| `--help` | Display help information | - | - |

## Usage

### Getting Help

To see all available options:

```bash
go run cmd/subscription-processor/main.go --help
# or with the compiled binary
./subscription-processor --help
```

### Development

```bash
# Run once and exit
go run cmd/subscription-processor/main.go --once

# Run with default 5-minute interval
go run cmd/subscription-processor/main.go

# Run with custom interval (15 minutes)
go run cmd/subscription-processor/main.go --interval=15m

# Run with a complex interval (1 hour and 30 minutes)
go run cmd/subscription-processor/main.go --interval=1h30m
```

### Production

Build the binary:

```bash
go build -o subscription-processor cmd/subscription-processor/main.go
```

Run as a service:

```bash
./subscription-processor --interval=15m
```

## Service Integration

### Running as a systemd service

Create a systemd service file at `/etc/systemd/system/cyphera-subscription.service`:

```ini
[Unit]
Description=Cyphera Subscription Processor
After=network.target

[Service]
Type=simple
User=cyphera
WorkingDirectory=/opt/cyphera
ExecStart=/opt/cyphera/subscription-processor --interval=15m
Restart=on-failure
RestartSec=10
Environment=DATABASE_URL=postgresql://user:password@localhost:5432/cyphera
Environment=CYPHERA_SMART_WALLET_ADDRESS=0x1234567890123456789012345678901234567890
Environment=DELEGATION_GRPC_ADDR=localhost:50051
Environment=DELEGATION_RPC_TIMEOUT=3m
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
```

Enable and start the service:

```bash
sudo systemctl enable cyphera-subscription
sudo systemctl start cyphera-subscription
```

### Docker Deployment

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o subscription-processor cmd/subscription-processor/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/subscription-processor .
CMD ["./subscription-processor", "--interval=15m"]
```

Build and run:

```bash
docker build -t cyphera-subscription .
docker run -d \
  -e DATABASE_URL="postgresql://user:password@db:5432/cyphera" \
  -e CYPHERA_SMART_WALLET_ADDRESS="0x1234567890123456789012345678901234567890" \
  -e DELEGATION_GRPC_ADDR="delegation-service:50051" \
  cyphera-subscription
```

### Kubernetes CronJob (Alternative)

For environments where continuous running is not ideal, a Kubernetes CronJob can be used instead:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cyphera-subscription-processor
spec:
  schedule: "*/15 * * * *"  # Every 15 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: subscription-processor
            image: cyphera-subscription:latest
            args:
            - "--once"
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: cyphera-secrets
                  key: database-url
            - name: CYPHERA_SMART_WALLET_ADDRESS
              valueFrom:
                configMapKeyRef:
                  name: cyphera-config
                  key: smart-wallet-address
            - name: DELEGATION_GRPC_ADDR
              value: "delegation-service:50051"
          restartPolicy: OnFailure
``` 