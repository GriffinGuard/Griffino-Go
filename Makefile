.PHONY: all build vet test fmt tidy

all: fmt vet test

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy
