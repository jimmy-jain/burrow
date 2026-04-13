# Makefile for Burrow

.PHONY: all build clean release

# Output directory
BIN_DIR := bin

# Go toolchain
GO ?= go
GO_DOWNLOAD_RETRIES ?= 3

# Binaries
ANALYZE := analyze
STATUS := status
WATCH := watch
DUPES := dupes
MCP := mcp

# Source directories
ANALYZE_SRC := ./cmd/analyze
STATUS_SRC := ./cmd/status
WATCH_SRC := ./cmd/watch
DUPES_SRC := ./cmd/dupes
MCP_SRC := ./cmd/mcp

# Build flags
LDFLAGS := -s -w

all: build

# Download modules with retries to mitigate transient proxy/network EOF errors.
mod-download:
	@attempt=1; \
	while [ $$attempt -le $(GO_DOWNLOAD_RETRIES) ]; do \
		echo "Downloading Go modules ($$attempt/$(GO_DOWNLOAD_RETRIES))..."; \
		if $(GO) mod download; then \
			exit 0; \
		fi; \
		sleep $$((attempt * 2)); \
		attempt=$$((attempt + 1)); \
	done; \
	echo "Go module download failed after $(GO_DOWNLOAD_RETRIES) attempts"; \
	exit 1

# Local build (current architecture)
build: mod-download
	@echo "Building for local architecture..."
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(ANALYZE)-go $(ANALYZE_SRC)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(STATUS)-go $(STATUS_SRC)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(WATCH)-go $(WATCH_SRC)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(DUPES)-go $(DUPES_SRC)
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/burrow-mcp $(MCP_SRC)

# Release build targets (run on native architectures for CGO support)
release-amd64: mod-download
	@echo "Building release binaries (amd64)..."
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(ANALYZE)-darwin-amd64 $(ANALYZE_SRC)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(STATUS)-darwin-amd64 $(STATUS_SRC)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(WATCH)-darwin-amd64 $(WATCH_SRC)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(DUPES)-darwin-amd64 $(DUPES_SRC)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/burrow-mcp-darwin-amd64 $(MCP_SRC)

release-arm64: mod-download
	@echo "Building release binaries (arm64)..."
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(ANALYZE)-darwin-arm64 $(ANALYZE_SRC)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(STATUS)-darwin-arm64 $(STATUS_SRC)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(WATCH)-darwin-arm64 $(WATCH_SRC)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(DUPES)-darwin-arm64 $(DUPES_SRC)
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/burrow-mcp-darwin-arm64 $(MCP_SRC)

clean:
	@echo "Cleaning binaries..."
	rm -f $(BIN_DIR)/$(ANALYZE)-* $(BIN_DIR)/$(STATUS)-* $(BIN_DIR)/$(WATCH)-* $(BIN_DIR)/$(DUPES)-* $(BIN_DIR)/$(ANALYZE)-go $(BIN_DIR)/$(STATUS)-go $(BIN_DIR)/$(WATCH)-go $(BIN_DIR)/$(DUPES)-go $(BIN_DIR)/burrow-mcp
