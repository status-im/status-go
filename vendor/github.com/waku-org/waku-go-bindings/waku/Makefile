# Makefile for Waku Go Bindings

# Determine OS and set library extension
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	EXT := so
else ifeq ($(UNAME_S),Darwin)
	EXT := so
else ifeq ($(OS),Windows_NT)
	EXT := dll
else
	$(error Unsupported OS: $(UNAME_S))
endif

THIRD_PARTY_DIR := $(shell pwd)/../third_party
NWAKU_REPO := https://github.com/waku-org/nwaku
NWAKU_DIR := $(THIRD_PARTY_DIR)/nwaku
LIBWAKU_HEADER := $(NWAKU_DIR)/libwaku.h
LIBWAKU := $(NWAKU_DIR)/libwaku.${EXT}

.PHONY: all clean build libwaku

# Default target
all: build

# Prepare third_party directory and clone nwaku
# NOTE: Currently we download a library release.
# 		In the future we should get nwaku from Nimble package manager.
$(LIBWAKU) $(LIBWAKU_HEADER):
	@chmod +x ../scripts/download_nwaku.sh
	@../scripts/download_nwaku.sh

libwaku: $(LIBWAKU) $(LIBWAKU_HEADER)

# Build libwaku
build-libwaku:
	@echo "Building libwaku..."
	@cd $(NWAKU_DIR) && make libwaku

# Build Waku Go Bindings
build: libwaku
	@echo "Building Waku Go Bindings..."
	go build ./...

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -f $(LIBWAKU_HEADER) $(LIBWAKU)
	@rm -f waku-go-bindings