# Makefile for SDS Go Bindings

# The NIM_SDS_LIB_PATH and NIM_SDS_HEADER_PATH env vars should be defined beforehand
export CGO_CFLAGS="-I${NIM_SDS_HEADER_PATH}/"
export CGO_LDFLAGS="-L${NIM_SDS_LIB_PATH}/ -lsds -Wl,-rpath,${NIM_SDS_LIB_PATH}/"

.PHONY: all clean prepare build-libsds build

# Default target
all: build

# Prepare third_party directory and clone nim-sds
prepare:
	@echo "Preparing third_party directory..."
	@mkdir -p ../third_party

	@echo "Setting up nim-sds..."
	@if [ ! -d "../third_party/nim-sds/.git" ]; then \
		echo "Cloning nim-sds repository..."; \
		git clone https://github.com/waku-org/nim-sds ../third_party/nim-sds; \
	else \
		echo "nim-sds repository already exists."; \
	fi

	@echo "Updating nim-sds to get vendors and compile nim in nimbus-build-system..."
	@cd ../third_party/nim-sds && \
		git fetch origin && \
		git checkout master && \
		echo "Running 'make update'..." && \
		make update

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
