# Makefile for SDS Go Bindings

# The NIM_SDS_REPO_PATH env var should be defined beforehand and it represents the absolute path
# to the cloned nim-sds repository, i.e, ....../third_party/nim-sds
export CGO_CFLAGS="-I${NIM_SDS_REPO_PATH}/library/"
export CGO_LDFLAGS="-L${NIM_SDS_REPO_PATH}/build -lsds -Wl,-rpath,${NIM_SDS_REPO_PATH}/build"

.PHONY: all clean prepare build-libsds build

# Default target
all: build

# Prepare third_party directory and clone nim-sds
prepare:
	@echo "Creating third_party directory..."
	@mkdir -p ../third_party

	@if [ ! -d "../third_party/nim-sds" ]; then \
		echo "Cloning nim-sds repository..." ; \
		cd ../third_party/ && \
		git clone https://github.com/waku-org/nim-sds && \
		cd nim-sds && \
		git checkout master; \
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
