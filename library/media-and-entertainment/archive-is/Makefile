.PHONY: build test lint install clean

build:
	go build -o bin/archive-is-pp-cli ./cmd/archive-is-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/archive-is-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/archive-is-pp-mcp ./cmd/archive-is-pp-mcp

install-mcp:
	go install ./cmd/archive-is-pp-mcp

build-all: build build-mcp
