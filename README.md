# cyphera-api

## Environment Configuration

This project uses environment variables for configuration. To set up:

1. Copy the template file to create your local environment file:
   ```bash
   cp .env.template .env
   ```

2. Edit `.env` and update the values:
   - `ACTALINK_API_KEY`: Your API key for Actalink (required)
   - `PORT`: Server port (default: 8000)

## Running the Project

To run, first download the dependencies (use at least go version go 1.23):

```bash
go mod download
```

Then run the server:

```bash
go run cmd/api/main.go
```

Then, make a request to the API. Message admin for access to the API key:
```bash
curl -X GET 'http://localhost:8000/api/v1/nonce' -H 'x-api-key: {API_KEY}'
```

## API Documentation

The API documentation is available through Swagger UI at `http://localhost:8000/swagger/index.html` when the server is running.

To update the swagger documentation, run the following command:

```bash
swag init -g cmd/api/main.go
```

Then run the server again.