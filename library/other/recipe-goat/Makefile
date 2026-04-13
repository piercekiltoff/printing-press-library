.PHONY: build test lint install clean

build:
	go build -o bin/recipe-goat-pp-cli ./cmd/recipe-goat-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/recipe-goat-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/recipe-goat-pp-mcp ./cmd/recipe-goat-pp-mcp

install-mcp:
	go install ./cmd/recipe-goat-pp-mcp

build-all: build build-mcp
