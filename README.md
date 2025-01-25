# cyphera-api

## Environment Configuration

This project uses environment variables for configuration. To set up:

1. Copy the template file to create your local environment file:
   ```bash
   cp .env.template .env
   ```

2. Edit `.env` and update the values:
   - `ACTALINK_API_KEY`: Your API key for Actalink (required)

## Running the Project

To run install docker first: https://docs.docker.com/get-started/get-docker/

Then, run the following command to start the server and postgres database:

```bash
docker compose up
```

Then, make a request to the API. Message admin for access to the API key:
```bash
curl -X GET 'http://localhost:8000/api/v1/nonce' -H 'x-api-key: {API_KEY}'
```

## API Documentation

The API documentation is available through Swagger UI at `http://localhost:8000/swagger/index.html` when the server is running.

To update the swagger documentation, run the following command:

```bash
make gen
```

Then run the server again.