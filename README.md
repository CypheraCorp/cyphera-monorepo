# cyphera-api

To run, first download the dependencies (use at least go version go 1.23):

```
go mod download
```

Then run the server:

```
go run cmd/api/main.go
```

Then, make a request to the API. Message admin for access to the API key
```
curl -X GET 'http://localhost:8000/api/v1/nonce` -H 'x-api-key: {API_KEY}' 
```

## API Documentation

The API documentation is available through Swagger UI at `http://localhost:8000/swagger/index.html` when the server is running. You can access it at:

to update the swagger documentation, run the following command:

```
swag init -g cmd/api/main.go
```

then run the server again.