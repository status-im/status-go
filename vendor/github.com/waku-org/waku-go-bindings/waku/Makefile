# Makefile for Waku Go Bindings

# Directories
THIRD_PARTY_DIR := ../third_party
NWAKU_REPO := https://github.com/waku-org/nwaku
NWAKU_DIR := $(THIRD_PARTY_DIR)/nwaku

.PHONY: all clean prepare build-libwaku build

# Default target
all: build

# Prepare third_party directory and clone nwaku
prepare:
	@echo "Creating third_party directory..."
	@mkdir -p $(THIRD_PARTY_DIR)

	@echo "Cloning nwaku repository..."
	@if [ ! -d "$(NWAKU_DIR)" ]; then \
		cd $(THIRD_PARTY_DIR) && \
		git clone $(NWAKU_REPO) && \
		cd $(NWAKU_DIR) && \
		make update; \
	else \
		echo "nwaku repository already exists."; \
	fi

# Build libwaku
build-libwaku: prepare
	@echo "Building libwaku..."
	@cd $(NWAKU_DIR) && make libwaku

# Build Waku Go Bindings
build: build-libwaku
	@echo "Building Waku Go Bindings..."
	go build ./...

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -rf $(THIRD_PARTY_DIR)
	@rm -f waku-go-bindings