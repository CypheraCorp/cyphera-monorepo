#!/bin/bash
set -e

echo "🔧 Setting up LocalStack resources..."

# Wait for LocalStack to be ready
echo "⏳ Waiting for LocalStack to start..."
until curl -s http://localhost:4566/_localstack/health | grep -q '"sqs": "available"'; do
  sleep 2
done

# Create SQS queues
echo "📦 Creating SQS queues..."

# Main webhook queue
aws --endpoint-url=http://localhost:4566 sqs create-queue \
  --queue-name webhook-queue \
  --region us-east-1 || echo "Queue webhook-queue may already exist"

# Dead letter queue
aws --endpoint-url=http://localhost:4566 sqs create-queue \
  --queue-name webhook-dlq \
  --region us-east-1 || echo "Queue webhook-dlq may already exist"

# Create secrets for testing
echo "🔐 Creating test secrets..."

aws --endpoint-url=http://localhost:4566 secretsmanager create-secret \
  --name payment-sync-encryption-key \
  --secret-string "local_development_key_32_characters" \
  --region us-east-1 || echo "Secret may already exist"

echo "✅ LocalStack setup complete!"
