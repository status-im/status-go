# Makefile for SDS Go Bindings

.PHONY: all clean prepare build-libsds build

# Default target
all: build

# Prepare third_party directory and clone nim-sds
prepare:
	@echo "Creating third_party directory..."
	@mkdir -p ../third_party

	@echo "Cloning nim-sds repository..."
	@if [ ! -d "../third_party/nim-sds" ]; then \
		cd ../third_party/ && \
		git clone https://github.com/waku-org/nim-sds && \
		cd nim-sds && \
		git checkout add-android-support; \
		make update; \
	else \
		echo "nim-sds repository already exists."; \
	fi

# Build SDS Go Bindings
build: prepare
	@echo "Building libsds..."
	@cd ../third_party/nim-sds && make libsds
	@echo "Building SDS Go Bindings..."
	go build ./...

build-android: prepare
	@echo "Building libsds for Android..."
	@cd ../third_party/nim-sds && make libsds-android
	@echo "Building SDS Go Bindings for Android..."
	go build ./...

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -rf ../third_party/
	@rm -f sds-go-bindings
