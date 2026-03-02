BINARY_NAME=wolf
BUILD_DIR=.
GO=go
GOFLAGS=-v

.PHONY: build test run fmt clean help

## build: Build the wolf binary
build:
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wolf

## build-all: Build the wolf binary for all supported platforms
build-all: build-linux-amd64 build-darwin-amd64 build-darwin-arm64

## build-linux-amd64: Build for Linux amd64
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/wolf

## build-darwin-amd64: Build for macOS (Intel)
build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/wolf

## build-darwin-arm64: Build for macOS (Apple Silicon)
build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/wolf

## test: Run all tests
test:
	$(GO) test ./... -v

## test-cover: Run all tests with coverage
test-cover:
	$(GO) test ./... -v -cover -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

## run: Build and run wolf
run: build
	./$(BINARY_NAME)

## fmt: Format all Go source files
fmt:
	$(GO) fmt ./...
	gofmt -s -w .

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -rf wolf_out/

## help: Show this help
help:
	@echo "Wolf Language Compiler"
	@echo ""
	@echo "Usage:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'
