.PHONY: build test lint fmt

build:
	go build -o jobwatch .

test:
	go test ./...

lint:
	go vet ./...
	golangci-lint run

fmt:
	gofmt -w .
