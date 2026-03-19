BINARY_NAME  = wolf
BUILD_DIR    = .
GO           = go
GOFLAGS      = -v

# ─── Version info baked into the binary ──────────────────────────────────────
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS      = -ldflags "-X main.Version=$(VERSION)"

.PHONY: build test run fmt clean help \
        build-all \
        build-linux-amd64 build-linux-arm64 \
        build-darwin-amd64 build-darwin-arm64 \
        build-windows-amd64 build-windows-arm64 \
        test-cover coverage-html

# ─── Primary build ────────────────────────────────────────────────────────────

## build: Build the wolf binary for the current platform
build:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wolf

# ─── Cross-platform builds ────────────────────────────────────────────────────
# The wolf CLI binary itself is pure Go (CGO_ENABLED=0).
# wolf_runtime.c is compiled at *runtime* by the installed clang on the user's
# machine, so no C cross-compilation happens here.
#
# Windows note: wolf_runtime.c compilation on Windows requires:
#   - LLVM/clang (https://releases.llvm.org)
#   - libmysqlclient headers via MSYS2: pacman -S mingw-w64-x86_64-libmariadbclient
#   - hiredis headers via MSYS2:        pacman -S mingw-w64-x86_64-hiredis
#   Or enable -DWOLF_REDIS_ENABLED=0 to skip Redis on Windows.

## build-all: Build wolf for all supported platforms
build-all: \
	build-linux-amd64 \
	build-linux-arm64 \
	build-darwin-amd64 \
	build-darwin-arm64 \
	build-windows-amd64 \
	build-windows-arm64

## build-linux-amd64: Linux x86-64 (most servers / CI runners)
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	$(GO) build $(GOFLAGS) $(LDFLAGS) \
	  -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/wolf

## build-linux-arm64: Linux ARM64 (Raspberry Pi 4/5, AWS Graviton, Oracle Free Tier)
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
	$(GO) build $(GOFLAGS) $(LDFLAGS) \
	  -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/wolf

## build-darwin-amd64: macOS Intel
build-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
	$(GO) build $(GOFLAGS) $(LDFLAGS) \
	  -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/wolf

## build-darwin-arm64: macOS Apple Silicon (M1/M2/M3/M4)
build-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
	$(GO) build $(GOFLAGS) $(LDFLAGS) \
	  -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/wolf

## build-windows-amd64: Windows x86-64 (.exe)
build-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
	$(GO) build $(GOFLAGS) $(LDFLAGS) \
	  -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/wolf

## build-windows-arm64: Windows ARM64 — Surface Pro X, Snapdragon laptops (.exe)
build-windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 \
	$(GO) build $(GOFLAGS) $(LDFLAGS) \
	  -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/wolf

# ─── Testing ──────────────────────────────────────────────────────────────────

## test: Run all tests
test:
	$(GO) test ./... -v -timeout 30m

## test-cover: Run all tests with coverage report printed to terminal
test-cover:
	$(GO) test ./... -v -cover -coverprofile=coverage.out -timeout 30m
	$(GO) tool cover -func=coverage.out

## coverage-html: Generate and open an HTML coverage report
coverage-html:
	$(GO) test ./... -cover -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out

# ─── Dev helpers ──────────────────────────────────────────────────────────────

## run: Build and run wolf
run: build
	./$(BINARY_NAME)

## fmt: Format all Go source files  (gofmt -s simplifies; go fmt is a wrapper)
fmt:
	$(GO) fmt ./...
	gofmt -s -w .

## clean: Remove all build artifacts and test output directories
clean:
	@echo "Removing binaries..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux-amd64
	rm -f $(BINARY_NAME)-linux-arm64
	rm -f $(BINARY_NAME)-darwin-amd64
	rm -f $(BINARY_NAME)-darwin-arm64
	rm -f $(BINARY_NAME)-windows-amd64.exe
	rm -f $(BINARY_NAME)-windows-arm64.exe
	@echo "Removing coverage reports..."
	rm -f coverage.out
	@echo "Removing build output directories..."
	rm -rf wolf_out/
	rm -rf e2e/testdata/wolf_out_*/
	@echo "Clean complete."

# ─── Help ─────────────────────────────────────────────────────────────────────

## help: Show this help message
help:
	@echo "Wolf Language Compiler — available targets:"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'