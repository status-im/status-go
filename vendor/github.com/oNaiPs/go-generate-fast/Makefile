# Variables
GO := go
SRC_DIR := src
OUT_DIR := bin
TARGET := $(OUT_DIR)/go-generate-fast

# List of all Go source files
SOURCES := main.go $(shell find $(SRC_DIR) -name "*.go")

BUILD_TAGS :=
TEST_TAGS := test

# By default, commands are not printed, unless you pass V=1
ifndef V
.SILENT:
endif

GO_DEPS=$(shell go list -e -f '{{join .Imports " "}}' tools.go)

install-deps: ## Installs dependencies
	echo "Installing dependencies"
	go install $(GO_DEPS)

build: $(TARGET) ## Build project

$(TARGET): $(SOURCES)
	mkdir -p $(OUT_DIR)
	echo "Building..."
	$(GO) build -o $(TARGET) --tags=$(BUILD_TAGS)
	echo "Build complete. Output: $(TARGET)"

test: ## Run tests
	echo "Running unit tests..."
	gotestsum --raw-command -- $(GO) -C $(SRC_DIR) test --tags=$(TEST_TAGS) -json -cover ./...

lint: ## Lints files
	echo "Linting..."
	golangci-lint run
lint-fix: ## Lints files, and fixes ones that are fixable 
	echo "Linting and fixing..."
	golangci-lint run --fix

e2e: $(if $(CI),,build) ## Runs e2e tests
	echo "Running e2e tests..."
	e2e/run.sh

clean: ## Cleans build files
	rm -rfv $(OUT_DIR)
	echo Done.

.PHONY: build clean test e2e help

.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

