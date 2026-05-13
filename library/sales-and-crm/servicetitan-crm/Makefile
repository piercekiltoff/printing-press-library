.PHONY: build test lint install clean

build:
	go build -o bin/servicetitan-crm-pp-cli ./cmd/servicetitan-crm-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/servicetitan-crm-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/servicetitan-crm-pp-mcp ./cmd/servicetitan-crm-pp-mcp

install-mcp:
	go install ./cmd/servicetitan-crm-pp-mcp

build-all: build build-mcp
