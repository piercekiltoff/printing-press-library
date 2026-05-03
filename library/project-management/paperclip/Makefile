.PHONY: build test lint install clean

build:
	go build -o bin/paperclip-pp-cli ./cmd/paperclip-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/paperclip-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/paperclip-pp-mcp ./cmd/paperclip-pp-mcp

install-mcp:
	go install ./cmd/paperclip-pp-mcp

build-all: build build-mcp
