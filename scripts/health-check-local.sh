#!/bin/bash
set -e

echo "🔍 Checking local webhook system health..."

# Check API
echo -n "API (port 8080): "
if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check webhook receiver
echo -n "Webhook Receiver (port 3001): "
if curl -sf http://localhost:3001/health > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check database
echo -n "Database (port 5432): "
if pg_isready -h localhost -p 5432 -U apiuser > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check LocalStack
echo -n "LocalStack (port 4566): "
if curl -sf http://localhost:4566/_localstack/health > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check SQS queue
echo -n "SQS Queue: "
if AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 sqs get-queue-url --queue-name webhook-queue --region us-east-1 > /dev/null 2>&1; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

# Check webhook processor
echo -n "Webhook Processor: "
if docker-compose -f docker-compose.webhooks.yml ps webhook-processor | grep -q Up; then
  echo "✅ OK"
else
  echo "❌ FAIL"
fi

echo "🏁 Health check complete!"
