.PHONY: build test lint install clean

build:
	go build -o bin/redfin-stingray-unofficial-pp-cli ./cmd/redfin-stingray-unofficial-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/redfin-stingray-unofficial-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/redfin-stingray-unofficial-mcp ./cmd/redfin-stingray-unofficial-mcp

install-mcp:
	go install ./cmd/redfin-stingray-unofficial-mcp

build-all: build build-mcp
