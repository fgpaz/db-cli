.PHONY: build build-all test lint clean

BUILD_DIR := bin
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags "-s -w -X main.version=$(VERSION)"

# Current platform defaults
GOOS   := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
OUT    := $(BUILD_DIR)/db-cli
ifeq ($(GOOS),windows)
  OUT := $(BUILD_DIR)/db-cli.exe
endif

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(OUT) ./cmd/db-cli

build-all:
	@mkdir -p $(BUILD_DIR)/windows-amd64 \
	         $(BUILD_DIR)/windows-arm64 \
	         $(BUILD_DIR)/linux-amd64 \
	         $(BUILD_DIR)/linux-arm64
	GOOS=windows GOARCH=amd64 $(LDFLAGS) -o $(BUILD_DIR)/windows-amd64/db-cli.exe ./cmd/db-cli
	GOOS=windows GOARCH=arm64 $(LDFLAGS) -o $(BUILD_DIR)/windows-arm64/db-cli.exe ./cmd/db-cli
	GOOS=linux   GOARCH=amd64 $(LDFLAGS) -o $(BUILD_DIR)/linux-amd64/db-cli ./cmd/db-cli
	GOOS=linux   GOARCH=arm64 $(LDFLAGS) -o $(BUILD_DIR)/linux-arm64/db-cli ./cmd/db-cli

test:
	go test -v -race -count=1 ./...

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)/*
