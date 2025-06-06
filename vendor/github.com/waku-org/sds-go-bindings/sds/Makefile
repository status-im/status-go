# Makefile for SDS Go Bindings

# Directories
THIRD_PARTY_DIR := ../third_party
NIM_SDS_REPO := https://github.com/waku-org/nim-sds
NIM_SDS_DIR := $(THIRD_PARTY_DIR)/nim-sds

.PHONY: all clean prepare build-libsds build

# Default target
all: build

# Prepare third_party directory and clone nim-sds
prepare:
	@echo "Creating third_party directory..."
	@mkdir -p $(THIRD_PARTY_DIR)

	@echo "Cloning nim-sds repository..."
	@if [ ! -d "$(NIM_SDS_DIR)" ]; then \
		cd $(THIRD_PARTY_DIR) && \
		git clone $(NIM_SDS_REPO) && \
		cd $(NIM_SDS_DIR) && \
		git checkout fix-statusgo-build; \
		make update; \
	else \
		echo "nim-sds repository already exists."; \
	fi

# Build libsds
build-libsds: prepare
	@echo "Building libsds..."
	@cd $(NIM_SDS_DIR) && make libsds

# Build SDS Go Bindings
build: build-libsds
	@echo "Building SDS Go Bindings..."
	go build ./...

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -rf $(THIRD_PARTY_DIR)
	@rm -f sds-go-bindings