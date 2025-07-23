module github.com/cyphera/cyphera-api/tools/delegation-integration-test

go 1.23.0

toolchain go1.24.2

require github.com/cyphera/cyphera-api/libs/go v0.0.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250528174236-200df99c418a // indirect
	google.golang.org/grpc v1.74.2 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/cyphera/cyphera-api/libs/go => ../../libs/go
