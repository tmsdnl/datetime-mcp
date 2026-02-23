# Makefile for datetime-mcp

BINARY    := datetime-mcp
CMD       := ./cmd/datetime-mcp

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE      ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS   := -s -w \
             -X main.version=$(VERSION) \
             -X main.commit=$(COMMIT) \
             -X main.date=$(DATE)

BUILD_FLAGS := CGO_ENABLED=0

DIST := dist

.PHONY: all build clean test

all: \
    $(DIST)/$(BINARY)_darwin_arm64 \
    $(DIST)/$(BINARY)_darwin_amd64 \
    $(DIST)/$(BINARY)_linux_arm64 \
    $(DIST)/$(BINARY)_linux_amd64 \
    $(DIST)/$(BINARY)_windows_arm64.exe \
    $(DIST)/$(BINARY)_windows_amd64.exe

$(DIST)/$(BINARY)_darwin_arm64:
	@mkdir -p $(DIST)
	$(BUILD_FLAGS) GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $@ $(CMD)

$(DIST)/$(BINARY)_darwin_amd64:
	@mkdir -p $(DIST)
	$(BUILD_FLAGS) GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $@ $(CMD)

$(DIST)/$(BINARY)_linux_arm64:
	@mkdir -p $(DIST)
	$(BUILD_FLAGS) GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $@ $(CMD)

$(DIST)/$(BINARY)_linux_amd64:
	@mkdir -p $(DIST)
	$(BUILD_FLAGS) GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $@ $(CMD)

$(DIST)/$(BINARY)_windows_arm64.exe:
	@mkdir -p $(DIST)
	$(BUILD_FLAGS) GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $@ $(CMD)

$(DIST)/$(BINARY)_windows_amd64.exe:
	@mkdir -p $(DIST)
	$(BUILD_FLAGS) GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $@ $(CMD)

build:
	$(BUILD_FLAGS) go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

test:
	go test ./...

clean:
	rm -rf $(DIST) $(BINARY)
