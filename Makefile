# Variables
BINARY_NAME=pathfinder
TUI_BINARY_NAME=pathfinder-tui
BUILD_DIR=build

# Commands
GO=go
TAGS ?= fts5
BUILD = $(GO) build $(if $(TAGS),-tags=$(TAGS),)
TEST=$(GO) test
CLEAN=$(GO) clean
RUN=./$(BUILD_DIR)/$(BINARY_NAME)
RUN_TUI=./$(BUILD_DIR)/$(TUI_BINARY_NAME)

.PHONY: all build build-tui run run-tui test clean help

all: build

## build: Build the project and put the binaries in the build directory
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-tui:
	@echo "Building $(TUI_BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(BUILD) -o $(BUILD_DIR)/$(TUI_BINARY_NAME) ./cmd/tui
	@echo "Build complete: $(BUILD_DIR)/$(TUI_BINARY_NAME)"

## run: Run the main application
run: build
	@echo "Running main.go..."
	$(BUILD_DIR)/$(BINARY_NAME)


## run-tui: Run the tui applicaiton
run-tui: build-tui
	@echo "Running TUI"
	$(BUILD_DIR)/$(TUI_BINARY_NAME)

## test: Run all tests in the project
test:
	@echo "Running tests..."
	$(TEST) ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(CLEAN)
	rm -rf $(BUILD_DIR)
	@echo "Clean complete."

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = \":.*?## \"}; {printf \"  \\033[36m%-15s\\033[0m %s\\n\", $$1, $$2}'
