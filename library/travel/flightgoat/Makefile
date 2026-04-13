.PHONY: build test lint install clean

build:
	go build -o bin/flightgoat-pp-cli ./cmd/flightgoat-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/flightgoat-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/flightgoat-pp-mcp ./cmd/flightgoat-pp-mcp

install-mcp:
	go install ./cmd/flightgoat-pp-mcp

build-all: build build-mcp
