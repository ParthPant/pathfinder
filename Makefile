# Variables
BINARY_NAME=pathfinder
BUILD_DIR=build

# Commands
GO=go
BUILD=$(GO) build
TEST=$(GO) test
CLEAN=$(GO) clean
RUN=./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: all build run test clean help

all: build

## build: Build the project and put the binary in the build directory
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## run: Build and run the project
run: build
	@echo "Running $(BINARY_NAME)..."
	$(RUN)

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
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
