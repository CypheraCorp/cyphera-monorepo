# cyphera-api

To run, first download the dependencies (use at least go version go 1.18):

```
go mod download
```

Then run the server:

```
go run cmd/api/main.go
```

then make a request to the API.

```
// message admin for access to the API key
curl -X GET 'http://localhost:8000/api/v1/nonce` -H 'x-api-key: {API_KEY}' 
```
