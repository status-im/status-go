# Makefile for Waku Go Bindings

# Directories
THIRD_PARTY_DIR := $(shell pwd)/../third_party
NWAKU_REPO := https://github.com/waku-org/nwaku
NWAKU_DIR := $(THIRD_PARTY_DIR)/nwaku

.PHONY: all clean build-libwaku build

# Default target
all: build

# Build libwaku
build-libwaku:
	@echo "Building libwaku..."
	@cd $(NWAKU_DIR) && make libwaku

# Build Waku Go Bindings

build: export CGO_CFLAGS = "-I${NWAKU_DIR}/library/"
build: export CGO_LDFLAGS = "-L${NWAKU_DIR}/build/ -lwaku -L${NWAKU_DIR} -Wl,-rpath,${NWAKU_DIR}/build/"
build:
	@echo "Building Waku Go Bindings..."
	go build ./...

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -rf $(THIRD_PARTY_DIR)
	@rm -f waku-go-bindings