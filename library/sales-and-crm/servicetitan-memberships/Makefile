.PHONY: build test lint install clean

build:
	go build -o bin/servicetitan-memberships-pp-cli ./cmd/servicetitan-memberships-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/servicetitan-memberships-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/servicetitan-memberships-pp-mcp ./cmd/servicetitan-memberships-pp-mcp

install-mcp:
	go install ./cmd/servicetitan-memberships-pp-mcp

build-all: build build-mcp
