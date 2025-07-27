.PHONY: build clean deploy

build:
	GOOS=linux GOARCH=amd64 go build -o bin/bootstrap cmd/main.go

clean:
	rm -rf ./bin

deploy: clean build
	sam deploy --template-file ../../infrastructure/aws-sam/template-dunning.yaml --stack-name dunning-processor --capabilities CAPABILITY_IAM

test:
	go test ./...

run-local:
	go run cmd/main.go