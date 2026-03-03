.PHONY: statusgo all test clean help
.PHONY: statusgo-ios-library statusgo-android-library
.PHONY: build-libwaku test-libwaku clean-libwaku rebuild-libwaku
.PHONY: build-libsds clean-libsds rebuild-libsds
.PHONY: build-storage fetch-storage clean-storage test-storage test-torrent
.PHONY: storage-help

# Clear any GOROOT set outside of the Nix shell
export GOROOT=

# This is a code for automatic help generator.
# It supports ANSI colors and categories.
# To add new item into help output, simply add comments
# starting with '##'. To add category, use @category.
GREEN  := $(shell printf "\e[32m")
WHITE  := $(shell printf "\e[37m")
YELLOW := $(shell printf "\e[33m")
RESET  := $(shell printf "\e[0m")
HELP_FUN = \
		   %help; \
		   while(<>) { push @{$$help{$$2 // 'options'}}, [$$1, $$3] if /^([a-zA-Z0-9\-]+)\s*:.*\#\#(?:@([a-zA-Z\-]+))?\s(.*)$$/ }; \
		   print "Usage: make [target]\n\n"; \
		   for (sort keys %help) { \
			   print "${WHITE}$$_:${RESET}\n"; \
			   for (@{$$help{$$_}}) { \
				   $$sep = " " x (32 - length $$_->[0]); \
				   print "  ${YELLOW}$$_->[0]${RESET}$$sep${GREEN}$$_->[1]${RESET}\n"; \
			   }; \
			   print "\n"; \
		   }

help: SHELL := /bin/sh
help: ##@other Show this help
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

RELEASE_TAG ?= $(shell ./scripts/version.sh)
RELEASE_DIR ?= /tmp/release-$(RELEASE_TAG)
GOLANGCI_BINARY = golangci-lint
LIBS_DIR := $(abspath ./libs)

ifeq ($(OS),Windows_NT)     # is Windows_NT on XP, 2000, 7, Vista, 10...
 detected_OS := Windows
else ifdef OS
 detected_OS := $(OS)
else
 detected_OS := $(strip $(shell uname))
endif

ifeq ($(detected_OS),Windows)
    MAKE_SHELL := bash
else
    MAKE_SHELL := /bin/bash
endif

ifeq ($(MAKECMDGOALS),statusgo-android-library)
    ARCH ?= arm64
    ANDROID_NDK_ROOT ?= $(shell find /nix/store -path "*android-sdk-ndk-27.2.12479018/libexec/android-sdk/ndk/27.2.12479018" -type d 2>/dev/null | head -1)
    ANDROID_API ?= 28
    HOST_OS ?= linux
    ifeq ($(ARCH),x86_64)
        MOBILE_GOARCH := amd64
        ANDROID_CLANG_TARGET := x86_64-linux-android$(ANDROID_API)
    else
        MOBILE_GOARCH := $(ARCH)
        ANDROID_CLANG_TARGET := aarch64-linux-android$(ANDROID_API)
    endif
    ANDROID_BUILD_FLAGS := CC="$(ANDROID_NDK_ROOT)/toolchains/llvm/prebuilt/$(HOST_OS)-x86_64/bin/clang --target=$(ANDROID_CLANG_TARGET) --sysroot=$(ANDROID_NDK_ROOT)/toolchains/llvm/prebuilt/$(HOST_OS)-x86_64/sysroot" CGO_ENABLED=1 GOOS=android GOARCH=$(MOBILE_GOARCH)
    CGO_CFLAGS+=-Os -flto -fembed-bitcode
    CGO_LDFLAGS+=-Os -flto
endif

ifeq ($(MAKECMDGOALS),statusgo-ios-library)
    ARCH ?= arm64
    IPHONE_SDK ?= iphoneos
    IOS_TARGET ?= 13.0
    ifeq ($(ARCH),x86_64)
        MOBILE_GOARCH := amd64
    else
        MOBILE_GOARCH := $(ARCH)
    endif
    IOS_BUILD_FLAGS := CGO_ENABLED=1 GOOS=ios GOARCH=$(MOBILE_GOARCH)
    CGO_CFLAGS+=-Os -flto -arch $(ARCH) -isysroot $$(xcrun --sdk $(IPHONE_SDK) --show-sdk-path) -miphoneos-version-min=$(IOS_TARGET) -fembed-bitcode
    CGO_LDFLAGS+=-Os -flto
endif

ifeq ($(detected_OS),Darwin)
 GOBIN_SHARED_LIB_EXT := dylib
 LIB_EXT := dylib
 GOBIN_SHARED_LIB_CFLAGS := CGO_ENABLED=1 GOOS=darwin
 CGO_CFLAGS+=-I/$(JAVA_HOME)/include -I/$(JAVA_HOME)/include/darwin
else ifeq ($(detected_OS),Windows)
 GOBIN_SHARED_LIB_EXT := dll
 LIB_EXT := dll
 GOBIN_SHARED_LIB_CGO_LDFLAGS := CGO_ENABLED=1 CGO_CFLAGS=-I$(LIBS_DIR) CGO_LDFLAGS="-L$(LIBS_DIR) -lstorage -Wl,-rpath,$(LIBS_DIR)"
else ifeq ($(detected_OS),Linux)
 GOBIN_SHARED_LIB_EXT := so
 LIB_EXT := so
 GOBIN_SHARED_LIB_CGO_LDFLAGS :=  CGO_ENABLED=1 CGO_CFLAGS=-I$(LIBS_DIR) CGO_LDFLAGS="-Wl,-soname,libstatus.so.0 -L$(LIBS_DIR) -lstorage -Wl,-rpath,$(LIBS_DIR)"
endif
export GOPATH ?= $(HOME)/go

GIT_ROOT ?= $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_AUTHOR ?= $(shell git config user.email || echo $$USER)

BUILD_TAGS ?= gowaku_no_rln

# `nwaku` variables

ifeq ($(USE_NWAKU), true)
    BUILD_TAGS += use_nwaku
    NWAKU_VERSION ?= v0.37.0-rc.3
    NWAKU_SOURCE_DIR ?= $(GIT_ROOT)/../nwaku
    LIBWAKU := $(NWAKU_SOURCE_DIR)/build/libwaku.$(LIB_EXT)
    CGO_CFLAGS+=-I$(NWAKU_SOURCE_DIR)/library
	CGO_LDFLAGS+=-L$(NWAKU_SOURCE_DIR)/build -lwaku -Wl,-rpath,$(NWAKU_SOURCE_DIR)/build
endif

# `nim-sds` variables

# Pin nim-sds revision here. Can be a tag (default) or commit hash.
NIM_SDS_VERSION ?= v0.2.4

# Option 1: Provide NIM_SDS_SOURCE_DIR. Make clones it if missing.
NIM_SDS_SOURCE_DIR ?= $(GIT_ROOT)/../nim-sds
# Normalize path separators for Windows (backslashes cause issues when passed through shells)
ifeq ($(mkspecs),win32)
	NIM_SDS_SOURCE_DIR := $(subst \,/,$(NIM_SDS_SOURCE_DIR))
endif

# Option 2: Provide NIM_SDS_LIB_DIR and NIM_SDS_INC_DIR

# Determine which approach to use
ifdef NIM_SDS_LIB_DIR
ifdef NIM_SDS_INC_DIR
    # External lib/include approach (e.g. used in Nix)
    NIM_SDS_BUILD_FROM_SOURCE := false
else
    $(error NIM_SDS_INC_DIR must be provided when NIM_SDS_LIB_DIR is set)
endif
else
    # Source directory approach
    NIM_SDS_LIB_DIR := $(NIM_SDS_SOURCE_DIR)/build
    NIM_SDS_INC_DIR := $(NIM_SDS_SOURCE_DIR)/library
    NIM_SDS_BUILD_FROM_SOURCE := true
endif

LIBSDS := $(NIM_SDS_LIB_DIR)/libsds.$(LIB_EXT)
CGO_CFLAGS+=-I$(NIM_SDS_INC_DIR)
CGO_LDFLAGS+=-L$(NIM_SDS_LIB_DIR) -lsds

# `logos-storage` variables (opt-in)
USE_LOGOS_STORAGE ?= false
USE_TORRENT ?= false
LIBS_DIR ?= $(GIT_ROOT)/libs

# If running in nix shell, these may already be exported by shellHook.
LOGOS_STORAGE_LIB_DIR ?= $(if $(LIBSTORAGE_PATH),$(LIBSTORAGE_PATH)/lib,$(LIBS_DIR))
LOGOS_STORAGE_INC_DIR ?= $(if $(LIBSTORAGE_PATH),$(LIBSTORAGE_PATH)/include,$(LOGOS_STORAGE_LIB_DIR))
LOGOS_STORAGE_VERSION ?= $(shell go list -m -f '{{.Version}}' github.com/logos-storage/logos-storage-go-bindings 2>/dev/null)
LOGOS_STORAGE_ARCH := $(shell uname -m)
ifeq ($(LOGOS_STORAGE_ARCH),x86_64)
  LOGOS_STORAGE_ARCH := amd64
else ifeq ($(LOGOS_STORAGE_ARCH),aarch64)
  LOGOS_STORAGE_ARCH := arm64
endif
ifeq ($(detected_OS),Darwin)
  LOGOS_STORAGE_OS := macos
else ifeq ($(detected_OS),Linux)
  LOGOS_STORAGE_OS := linux
else
  LOGOS_STORAGE_OS := $(shell echo $(detected_OS) | tr A-Z a-z)
endif
LOGOS_STORAGE_DOWNLOAD_URL := https://github.com/logos-storage/logos-storage-go-bindings/releases/download/$(LOGOS_STORAGE_VERSION)/storage-$(LOGOS_STORAGE_OS)-$(LOGOS_STORAGE_ARCH).zip

ifeq ($(USE_LOGOS_STORAGE),true)
	BUILD_TAGS += use_logos_storage
	CGO_CFLAGS += -I$(LOGOS_STORAGE_INC_DIR)
	CGO_LDFLAGS += -L$(LOGOS_STORAGE_LIB_DIR) -lstorage -Wl,-rpath,$(LOGOS_STORAGE_LIB_DIR)
endif

ifeq ($(USE_TORRENT),true)
	BUILD_TAGS += use_torrent
endif

RUNTIME_LIB_DIRS := $(NIM_SDS_LIB_DIR)
ifeq ($(USE_LOGOS_STORAGE),true)
	RUNTIME_LIB_DIRS := $(LOGOS_STORAGE_LIB_DIR):$(RUNTIME_LIB_DIRS)
endif

build-storage: fetch-storage

fetch-storage: ##@build Fetch libstorage for native non-Nix workflows
ifdef LIBSTORAGE_PATH
	@echo "Using libstorage from Nix shell: $(LIBSTORAGE_PATH)"
else
	@if ([ -f "$(LIBS_DIR)/libstorage.so" ] || [ -f "$(LIBS_DIR)/libstorage.dylib" ] || [ -f "$(LIBS_DIR)/libstorage.dll" ]) && [ -f "$(LIBS_DIR)/libstorage.h" ]; then \
		echo "libstorage and libstorage.h already present in $(LIBS_DIR); skipping download"; \
	else \
		echo "Fetching libstorage artifacts from: $(LOGOS_STORAGE_DOWNLOAD_URL)"; \
		mkdir -p "$(LIBS_DIR)"; \
		curl -fSL --create-dirs -o "$(LIBS_DIR)/logos-storage-$(LOGOS_STORAGE_OS)-$(LOGOS_STORAGE_ARCH).zip" "$(LOGOS_STORAGE_DOWNLOAD_URL)"; \
		unzip -o -qq "$(LIBS_DIR)/logos-storage-$(LOGOS_STORAGE_OS)-$(LOGOS_STORAGE_ARCH).zip" -d "$(LIBS_DIR)"; \
		rm -f "$(LIBS_DIR)"/logos-storage-*.zip; \
	fi
endif

clean-storage: ##@other Remove downloaded native libstorage artifacts
	@echo "Removing local libstorage artifacts from $(LIBS_DIR)"
	@rm -f "$(LIBS_DIR)"/libstorage.so "$(LIBS_DIR)"/libstorage.dylib "$(LIBS_DIR)"/libstorage.dll "$(LIBS_DIR)"/libstorage.h

test-storage: fetch-storage $(LIBSDS) ##@tests Run logosstorage-related package tests via gotestsum
	go generate -tags "$(BUILD_TAGS) use_logos_storage" ./services/logosstorage
	LD_LIBRARY_PATH="$(LOGOS_STORAGE_LIB_DIR):$(RUNTIME_LIB_DIRS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS) -L$(LOGOS_STORAGE_LIB_DIR) -lstorage -Wl,-rpath,$(LOGOS_STORAGE_LIB_DIR)" \
	CGO_CFLAGS="$(CGO_CFLAGS) -I$(LOGOS_STORAGE_INC_DIR)" \
	gotestsum --packages="./services/logosstorage" -f testname -- -count 1 -tags "$(BUILD_TAGS) use_logos_storage gowaku_skip_migrations"
	LD_LIBRARY_PATH="$(LOGOS_STORAGE_LIB_DIR):$(RUNTIME_LIB_DIRS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS) -L$(LOGOS_STORAGE_LIB_DIR) -lstorage -Wl,-rpath,$(LOGOS_STORAGE_LIB_DIR)" \
	CGO_CFLAGS="$(CGO_CFLAGS) -I$(LOGOS_STORAGE_INC_DIR)" \
	gotestsum --packages="./protocol/communities/archive/logosstorage" -f testname -- -count 1 -tags "$(BUILD_TAGS) use_logos_storage gowaku_skip_migrations"
	LD_LIBRARY_PATH="$(LOGOS_STORAGE_LIB_DIR):$(RUNTIME_LIB_DIRS)" \
	CGO_LDFLAGS="$(CGO_LDFLAGS) -L$(LOGOS_STORAGE_LIB_DIR) -lstorage -Wl,-rpath,$(LOGOS_STORAGE_LIB_DIR)" \
	CGO_CFLAGS="$(CGO_CFLAGS) -I$(LOGOS_STORAGE_INC_DIR)" \
	gotestsum --packages="./protocol" -f testname -- -count 1 -tags "$(BUILD_TAGS) use_logos_storage gowaku_skip_migrations" \
	-run TestMessengerCommunitiesTokenPermissionsSuite/TestUploadDownloadLogosStorageHistoryArchives

test-torrent: $(LIBSDS) ##@tests Run torrent archive package tests via gotestsum
	LD_LIBRARY_PATH="$(RUNTIME_LIB_DIRS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" \
	gotestsum --packages="./protocol/communities/archive/torrent" -f testname -- -count 1 -tags "$(BUILD_TAGS) use_torrent gowaku_skip_migrations"
	LD_LIBRARY_PATH="$(RUNTIME_LIB_DIRS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" \
	gotestsum --packages="./protocol" -f testname -- -count 1 -tags "$(BUILD_TAGS) use_torrent gowaku_skip_migrations" \
	-run TestMessengerCommunitiesTokenPermissionsSuite/TestImportDecryptedArchiveMessages

storage-help: ##@build Show history archive build/test toggles and env vars
	@echo "History archive build toggles:"
	@echo "  USE_TORRENT=true enables build tag: use_torrent"
	@echo "  USE_LOGOS_STORAGE=true enables for builds: use_logos_storage"
	@echo "  USE_LOGOS_STORAGE=true also wires libstorage include/lib/runtime paths"
	@echo ""
	@echo "test-storage always runs with logos-storage support enabled."
	@echo ""
	@echo "Variables:"
	@echo "  USE_LOGOS_STORAGE          (default: false)"
	@echo "  USE_TORRENT                (default: false)"
	@echo "  LOGOS_STORAGE_LIB_DIR      (default: \$$LIBSTORAGE_PATH/lib or ./libs)"
	@echo "  LOGOS_STORAGE_INC_DIR      (default: \$$LIBSTORAGE_PATH/include or LOGOS_STORAGE_LIB_DIR)"
	@echo "  FUNCTIONAL_TESTS_USE_LOGOS_STORAGE (default: false)"
	@echo "  FUNCTIONAL_TESTS_BUILD_TAGS        (default: gowaku_no_rln)"
	@echo ""
	@echo "Examples:"
	@echo "  make test-unit USE_LOGOS_STORAGE=true"
	@echo "  make test-unit USE_TORRENT=true"
	@echo "  make test-unit USE_LOGOS_STORAGE=true USE_TORRENT=true"
	@echo "  make fetch-storage"
	@echo "  make build-storage"
	@echo "  make test-torrent"
	@echo "  make test-storage"
	@echo "  FUNCTIONAL_TESTS_USE_LOGOS_STORAGE=true ./scripts/run_functional_tests.sh"

# mbedtls configuration for go-sqlcipher
ifeq ($(detected_OS),Windows)
 # On Windows, use portable C implementations and add -Werror=implicit-function-declaration workaround
 CGO_CFLAGS+=-Wno-implicit-function-declaration
endif

# Common flags

BUILD_FLAGS ?= -ldflags=""
BUILD_FLAGS_MOBILE ?=

networkid ?= StatusChain

DOCKER_IMAGE_NAME ?= statusteam/status-go
DOCKER_IMAGE_CUSTOM_TAG ?= $(RELEASE_TAG)
DOCKER_TEST_WORKDIR = /go/src/github.com/status-im/status-go/
DOCKER_TEST_IMAGE = golang:1.13

GO_CMD_PATHS := $(filter-out library, $(wildcard cmd/*))
GO_CMD_NAMES := $(notdir $(GO_CMD_PATHS))
GO_CMD_BUILDS := $(addprefix build/bin/, $(GO_CMD_NAMES))

# Location of symlinks to derivations that should not be garbage collected
export _NIX_GCROOTS = ./.nix-gcroots

#----------------
# Nix targets
#----------------

# Use $(call sh, <COMMAND>) instead of $(shell <COMMAND>) to avoid
# invoking a Nix shell when normal shell will suffice, it's faster.
# This works because it's defined before we set SHELL to Nix one.
define sh
$(shell $(1))
endef

# TODO: Define more specific shells.
ifneq ($(detected_OS),Windows)
# No need for shell.sh script anymore, we use nix develop directly
endif

shell: ##@prepare Enter into a pre-configured shell
ifndef IN_NIX_SHELL
	@echo "Entering nix development environment..."
	@nix --extra-experimental-features 'nix-command flakes' develop
else
	@echo -e "$(YELLOW)Nix shell is already active$(RESET)"
endif

nix-repl: SHELL := /bin/sh
nix-repl: ##@nix Start an interactive Nix REPL
	nix repl --file flake.nix

nix-gc-protected: SHELL := /bin/sh
nix-gc-protected:
	@echo -e "$(YELLOW)The following paths are protected:$(RESET)" && \
	ls -1 $(_NIX_GCROOTS) | sed 's/^/ - /'

nix-upgrade: SHELL := /bin/sh
nix-upgrade: ##@nix Upgrade Nix interpreter to current version.
	nix/scripts/upgrade.sh

nix-gc: nix-gc-protected ##@nix Garbage collect all packages older than 20 days from /nix/store
	nix-store --gc

nix-clean: ##@nix Remove all status-mobile build artifacts from /nix/store
	nix/scripts/clean.sh

nix-purge: SHELL := /bin/sh
nix-purge: ##@nix Completely remove Nix setup, including /nix directory
	nix/scripts/purge.sh

#----------------
# General targets
#----------------
all: $(GO_CMD_NAMES)

CGO_ENV := CGO_ENABLED=1 CGO_CFLAGS="$(CGO_CFLAGS)" CGO_LDFLAGS="-Wl,-rpath,$(LIBS_DIR) $(CGO_LDFLAGS)"
ifeq ($(detected_OS),Darwin)
CGO_ENV += DYLD_LIBRARY_PATH=$(LIBS_DIR):$(NIM_SDS_LIB_DIR):$$DYLD_LIBRARY_PATH
else
CGO_ENV += LD_LIBRARY_PATH=$(LIBS_DIR):$(NIM_SDS_LIB_DIR):$$LD_LIBRARY_PATH
endif

.PHONY: $(GO_CMD_NAMES) $(GO_CMD_PATHS) $(GO_CMD_BUILDS)
$(GO_CMD_BUILDS): generate fetch-storage $(LIBWAKU) $(LIBSDS)
$(GO_CMD_BUILDS): ##@build Build any Go project from cmd folder
	$(CGO_ENV) go build -v \
		-tags '$(BUILD_TAGS)' $(BUILD_FLAGS) \
		-o ./$@ ./cmd/$(notdir $@)
	@echo "Compilation done."
	@echo "Run \"build/bin/$(notdir $@) -h\" to view available commands."

# Flag needed by nim-based dependencies (e.g., nwaku and nim-sds) that also use nimbus-build-system.
# When USE_SYSTEM_NIM=1 skips compiling Nim compiler locally and instead,
# enforces to use system-installed Nim.
USE_SYSTEM_NIM ?= 1

# Libwaku targets

$(NWAKU_SOURCE_DIR): ##@build Clone nwaku
ifeq ($(USE_NWAKU),true)
	@echo "Cloning nwaku $(NWAKU_VERSION)..."
	git clone --branch $(NWAKU_VERSION) https://github.com/waku-org/nwaku.git $(NWAKU_SOURCE_DIR)
endif

clone-nwaku: $(NWAKU_SOURCE_DIR)

$(LIBWAKU): clone-nwaku
ifeq ($(USE_NWAKU),true)
	@echo "Building libwaku" $(LIBWAKU)
	$(MAKE) -C $(NWAKU_SOURCE_DIR) libwaku USE_SYSTEM_NIM=$(USE_SYSTEM_NIM) NIMFLAGS=-d:noSignalHandler SHELL=$(MAKE_SHELL)
endif

build-libwaku: $(LIBWAKU)

test-libwaku: | $(LIBWAKU)
	go test -tags '$(BUILD_TAGS) use_nwaku' -run TestDial ./wakuv2/... -count 1 -v -json | jq -r '.Output'

clean-libwaku:
	@echo "Removing libwaku"
	rm $(LIBWAKU)

rebuild-libwaku: | clean-libwaku $(LIBWAKU)

# libsds targets

.PHONY: clone-nim-sds
clone-nim-sds: ##@build Clone or update nim-sds
ifeq ($(NIM_SDS_BUILD_FROM_SOURCE),true)
	@echo "Cloning or updating nim-sds ..."
	if [ ! -d "$(NIM_SDS_SOURCE_DIR)" ]; then \
		git clone https://github.com/waku-org/nim-sds.git $(NIM_SDS_SOURCE_DIR); \
	else \
		cd $(NIM_SDS_SOURCE_DIR) && git fetch --tags; \
	fi
	cd $(NIM_SDS_SOURCE_DIR) && git checkout $(NIM_SDS_VERSION)
endif

$(LIBSDS): clone-nim-sds
ifeq ($(NIM_SDS_BUILD_FROM_SOURCE),true)
	@echo "Building nim-sds: $(LIBSDS)"
	$(MAKE) -C $(NIM_SDS_SOURCE_DIR) update USE_SYSTEM_NIM=$(USE_SYSTEM_NIM)
	$(MAKE) -C $(NIM_SDS_SOURCE_DIR) libsds USE_SYSTEM_NIM=$(USE_SYSTEM_NIM) NIMFLAGS=-d:noSignalHandler SHELL=$(MAKE_SHELL)
else
	@test -f $(LIBSDS) || (echo "Error: libsds not found at $(LIBSDS)" && exit 1)
endif

build-libsds: $(LIBSDS)


## Target-specific architecture mapping for libsds Android build
# Note: nim-sds uses 'amd64' for both x86 and x86_64
build-libsds-android: SDSARCH = $(strip $(if $(filter arm64,$(ARCH)),arm64,\
	$(if $(filter arm,$(ARCH)),arm,\
	$(if $(filter amd64,$(ARCH)),amd64,\
	$(if $(filter x86 x86_64,$(ARCH)),amd64,\
	$(error Unsupported ARCH '$(ARCH)'. Please set ARCH to one of: arm64, arm, amd64, x86, x86_64))))))
build-libsds-android: clone-nim-sds
	@echo "Building nim-sds for Android" $(LIBSDS)
	$(MAKE) -C $(NIM_SDS_SOURCE_DIR) libsds-android ARCH=$(SDSARCH) ANDROID_NDK_ROOT=$(ANDROID_NDK_ROOT) USE_SYSTEM_NIM=1 SHELL=$(MAKE_SHELL)

build-libsds-ios: clone-nim-sds
	@echo "Building nim-sds for iOS" $(LIBSDS)
	$(MAKE) -C $(NIM_SDS_SOURCE_DIR) libsds-ios USE_SYSTEM_NIM=$(USE_SYSTEM_NIM) SHELL=$(MAKE_SHELL)

clean-libsds:
	@echo "Removing libsds"
	rm $(LIBSDS)

rebuild-libsds: | clean-libsds $(LIBSDS)

# Status-go targets

# Add rpath to libstdc++ to avoid runtime errors with NIX
ifeq ($(detected_OS),Darwin)
	CXXLIB      := $(shell $(CXX) -print-file-name=libc++.1.dylib)
	CXXLIB_DIR  := $(dir $(CXXLIB))
	LD_PATH := $(LIBS_DIR):$(CXXLIB_DIR):$(DYLD_LIBRARY_PATH)
    LD_NAME := DYLD_LIBRARY_PATH
else ifeq ($(detected_OS),Windows)
	CXXLIB      := $(shell $(CXX) -print-file-name=libstdc++-6.dll)
	CXXLIB_DIR  := $(dir $(CXXLIB))
	LD_PATH := $(LIBS_DIR);$(CXXLIB_DIR);$(PATH)
    LD_NAME := PATH
else
	CXXLIB      := $(shell $(CXX) -print-file-name=libstdc++.so.6)
	CXXLIB_DIR  := $(dir $(CXXLIB))
	LD_PATH := $(LIBS_DIR):$(CXXLIB_DIR):$(LD_LIBRARY_PATH)
    LD_NAME := LD_LIBRARY_PATH
endif

#----------------
# LogosStorage
# Status-go targets

statusgo: ##@build Build status-go as status-backend server
statusgo: build/bin/status-backend

status-backend: ##@build Build status-backend to run status-go as HTTP server
status-backend: build/bin/status-backend

run-status-backend: PORT ?= 0
run-status-backend: $(LIBSDS)
run-status-backend: generate
run-status-backend: ##@run Start status-backend server listening to localhost:PORT
	LD_LIBRARY_PATH="$(NIM_SDS_LIB_DIR)" CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" \
	go run -mod=mod ./cmd/status-backend --address localhost:${PORT}

push-notification-server: ##@build Build push-notification-server
push-notification-server: build/bin/push-notification-server

cmd: ##@build Build all public apps in ./cmd
cmd: status-backend push-notification-server

status-go-deps:
	go clean -cache || true
	go clean -modcache || true
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1

statusgo-c-bindings: STATUS_GO_BINDINGS_PATH ?= build/bin/statusgo-lib
statusgo-c-bindings:
	@## tools/generate-cbindings/README.md explains the magic incantation behind this
	mkdir -p $(STATUS_GO_BINDINGS_PATH)
	go run ./tools/generate-cbindings > $(STATUS_GO_BINDINGS_PATH)/main.go

statusgo-library: STATUS_GO_BINDINGS_PATH ?= build/bin/statusgo-lib
statusgo-library: STATUS_GO_LIBRARY_OUT ?= build/bin
statusgo-library: generate
statusgo-library: statusgo-c-bindings $(LIBWAKU) $(LIBSDS)  ##@cross-compile Build status-go as static library for current platform
	@echo "Building static library..."
	$(CGO_ENV) go build \
		-tags '$(BUILD_TAGS)' \
		$(BUILD_FLAGS) \
		-buildmode=c-archive \
		-o $(STATUS_GO_LIBRARY_OUT)/libstatus.a \
		"$(STATUS_GO_BINDINGS_PATH)/main.go"
	@echo "Static library built: $(STATUS_GO_LIBRARY_OUT)/libstatus.a"

statusgo-shared-library: generate
statusgo-shared-library: statusgo-c-bindings $(LIBWAKU) $(LIBSDS) ##@cross-compile Build status-go as shared library for current platform
	@echo "Building shared library..."
	@echo "Tags: $(BUILD_TAGS)"
	CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" \
 	go build \
		-tags '$(BUILD_TAGS)' \
		$(BUILD_FLAGS) \
		-buildmode=c-shared \
		-o build/bin/libstatus.$(GOBIN_SHARED_LIB_EXT) \
		./build/bin/statusgo-lib
ifeq ($(detected_OS),Linux)
	cd build/bin && \
	ls -lah . && \
	mv ./libstatus.$(GOBIN_SHARED_LIB_EXT) ./libstatus.$(GOBIN_SHARED_LIB_EXT).0 && \
	ln -s ./libstatus.$(GOBIN_SHARED_LIB_EXT).0 ./libstatus.$(GOBIN_SHARED_LIB_EXT)
endif
	@echo "Shared library built:"
	@ls -la build/bin/libstatus.*

statusgo-android-library: generate statusgo-c-bindings $(LIBWAKU) build-libsds-android ##@cross-compile Build status-go as Android mobile library
	@echo "Building Android mobile library..."
	$(ANDROID_BUILD_FLAGS) $(CGO_ENV) go build -buildmode=c-shared -tags 'gowaku_no_rln nowatchdog disable_torrent' \
		-ldflags="-checklinkname=0 -X github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=true" \
		-o "build/bin/libstatus.so" ./build/bin/statusgo-lib
	@echo "Android library built"
	@file build/bin/libstatus.so

statusgo-ios-library: generate statusgo-c-bindings $(LIBWAKU) build-libsds-ios ##@cross-compile Build status-go as iOS mobile library
	@echo "Building iOS mobile library..."
	DEVELOPER_DIR="/Applications/Xcode.app/Contents/Developer" \
	CC="$$(xcrun --sdk $(IPHONE_SDK) --find clang)" \
	$(IOS_BUILD_FLAGS) $(CGO_ENV) go build -buildmode=c-archive -tags 'gowaku_no_rln nowatchdog disable_torrent' \
		-ldflags="-checklinkname=0 -X github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=true" \
		-o "build/bin/libstatus.a" ./build/bin/statusgo-lib
	@echo "iOS library built"
	@file build/bin/libstatus.a

docker-image: SHELL := /bin/sh
docker-image: BUILD_TARGET ?= cmd
docker-image: ##@docker Build docker image (use DOCKER_IMAGE_NAME to set the image name)
	@echo "Building docker image..."
	docker build . \
		--build-arg 'build_tags=$(BUILD_TAGS)' \
		--build-arg 'build_flags=$(BUILD_FLAGS)' \
		--build-arg 'build_target=$(BUILD_TARGET)' \
		--label 'commit=$(GIT_COMMIT)' \
		--label 'author=$(GIT_AUTHOR)' \
		-t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG) \
		-t $(DOCKER_IMAGE_NAME):latest

clean-docker-images: SHELL := /bin/sh
clean-docker-images:
	docker rmi -f $$(docker image ls --filter="reference=$(DOCKER_IMAGE_NAME)" --quiet)

setup: ##@setup Install all tools
setup: setup-dev

setup-dev: ##@setup Install all necessary tools for development
setup-dev:
	echo "Replaced by Nix shell. Use 'make shell' or just any target as-is."

clean-generated: CLEANUP_GENERATED_FILES?=true
clean-generated: CLEANUP_GENERATED_FILES_DRY_RUN?=false
clean-generated: ##@generate Remove orphaned generated files
	@if [ "$(CLEANUP_GENERATED_FILES)" = "true" ]; then \
		./scripts/cleanup_generated_files.sh; \
	else \
	  	echo "Skipping cleanup of generated files"; \
	fi

generate: PACKAGES ?= $$(go list -e ./... | grep -v "/contracts/")
generate: GO_GENERATE_CMD ?= go tool go-generate-fast
generate: export GO_GENERATE_FAST_DEBUG ?= false
generate: export GO_GENERATE_FAST_RECACHE ?= false
ifeq ($(NO_NETWORK),)
generate: fetch-storage
endif
generate: clean-generated
generate: ##@ Run generate for all given packages using go-generate-fast, fallback to `go generate` (e.g. for docker)
	@GOROOT=$$(go env GOROOT) $(GO_GENERATE_CMD) $(PACKAGES)
	@go generate -tags "use_logos_storage $(BUILD_TAGS)" ./services/logosstorage

generate-contracts:
	go generate ./contracts

download-tokens:
	echo "Downloading token lists..."; \
	GOROOT=$$(go env GOROOT) GOFLAGS="-mod=mod" go run ./services/wallet/token/local-token-lists/default-lists/downloader/main.go; \
	echo "token list downloaded successfully"; \

analyze-token-stores:
	go run -mod=mod ./services/wallet/token/local-token-lists/analyzer/main.go

prepare-release: clean-release
	mkdir -p $(RELEASE_DIR)
	zip -r $(RELEASE_DIR)/status-go-desktop.zip . -x *.git*
	${MAKE} clean

clean-release:
	rm -rf $(RELEASE_DIR)

docker-test: ##@tests Run tests in a docker container with golang.
	docker run --privileged --rm -it -v "$(PWD):$(DOCKER_TEST_WORKDIR)" -w "$(DOCKER_TEST_WORKDIR)" $(DOCKER_TEST_IMAGE) go test ${ARGS}

test: test-unit ##@tests Run basic, short tests during development

test-unit-prep: fetch-storage $(LIBSDS)
test-unit-prep: generate
test-unit-prep: export BUILD_TAGS ?=
test-unit-prep: export UNIT_TEST_DRY_RUN ?= false
test-unit-prep: export UNIT_TEST_COUNT ?= 1
test-unit-prep: export UNIT_TEST_FAILFAST ?= true
test-unit-prep: export UNIT_TEST_USE_DEVELOPMENT_LOGGER ?= true
test-unit-prep: export UNIT_TEST_REPORT_CODECOV ?= false

test-unit: test-unit-prep
test-unit: export UNIT_TEST_RERUN_FAILS ?= true
test-unit: export UNIT_TEST_PACKAGES ?= $(call sh, go list -tags '$(BUILD_TAGS)' ./... | \
	grep -v /t/e2e | \
	grep -v /t/benchmarks | \
	grep -v /transactions/fake | \
	grep -v /tests-unit-network)
test-unit: ##@tests Run unit and integration tests
	LD_LIBRARY_PATH="$(RUNTIME_LIB_DIRS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" \
	./scripts/run_unit_tests.sh

test-single: test-unit-prep
	LD_LIBRARY_PATH="$(RUNTIME_LIB_DIRS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" \
	go test -v $(PKG) -testify.m $(TEST)

test-unit-network: test-unit-prep
test-unit-network: export UNIT_TEST_RERUN_FAILS ?= false
test-unit-network: export UNIT_TEST_PACKAGES ?= $(call sh, go list ./tests-unit-network/...)
test-unit-network: ##@tests Run unit and integration tests with network access
	./scripts/run_unit_tests.sh

test-unit-race: export GOTEST_EXTRAFLAGS=-race
test-unit-race: test-unit ##@tests Run unit and integration tests with -race flag

test-functional: generate
test-functional: export FUNCTIONAL_TESTS_DOCKER_UID ?= $(call sh, id -u)
test-functional: export FUNCTIONAL_TESTS_REPORT_CODECOV ?= false
test-functional: export FUNCTIONAL_TESTS_USE_LOGOS_STORAGE ?= true
test-functional:
	@./scripts/run_functional_tests.sh

benchmark: export FUNCTIONAL_TESTS_DOCKER_UID ?= $(call sh, id -u)
benchmark:
	@./scripts/run_benchmark.sh

lint-panics: generate
	GOFLAGS=-tags='$(BUILD_TAGS),lint,use_logos_storage,use_torrent' \
	go tool goroutine-defer-guard -test=false -target github.com/status-im/status-go/common.LogOnPanic ./...

lint: generate lint-panics
lint:
	$(CGO_ENV) golangci-lint --build-tags '$(BUILD_TAGS) lint use_logos_storage use_torrent' run ./...

lint-fix: generate
	golangci-lint --build-tags '$(BUILD_TAGS) lint use_logos_storage use_torrent' run --fix ./...

clean: clean-storage ##@other Cleanup
	rm -fr build/bin/*

git-clean:
	git clean -xf

deep-clean: clean git-clean
	rm -Rdf .ethereumtest/StatusChain

tidy:
	go mod tidy

# Temporarily redirect the well-known `vendor` target to `vendor-hash`.
# This target will be removed in the future.
vendor: vendor-hash

# https://gitlab.com/peerdb/peerdb/-/blob/d1dbd0c6533dca16bf57322b57cc8fb6ab897a66/nix-update.sh
# After stopping vendoring https://github.com/status-im/status-go/pull/6951,
# we have to manually update `vendorHash` in the nix derivation.
# This target runs the nix build, extracts the expected vendorHash and updates with it the nix derivation.
vendor-hash:
	@echo "$(GREEN)Running nix build...$(RESET)"; \
	NIX_OUTPUT="$$(nix build --extra-experimental-features 'nix-command flakes' '.?submodules=1#status-go-library' 2>&1)"; \
	CURRENT_VENDOR_HASH="$$(echo $${NIX_OUTPUT} | sed -n 's/.*specified:[[:space:]]*\(sha256-[A-Za-z0-9+/=]*\).*/\1/p' | head -1)"; \
	NEW_VENDOR_HASH="$$(echo $${NIX_OUTPUT} | sed -n 's/.*got:[[:space:]]*\(sha256-[A-Za-z0-9+/=]*\).*/\1/p' | head -1)"; \
	sed -i '' "s|vendorHash = \"$${CURRENT_VENDOR_HASH}\";|vendorHash = \"$${NEW_VENDOR_HASH}\";|g" ./nix/pkgs/status-go/library/default.nix; \
	echo "Replaced vendorHash $${CURRENT_VENDOR_HASH} with $${NEW_VENDOR_HASH}"


migration: DEFAULT_MIGRATION_PATH := appdatabase/migrations/sql
migration:
	touch $(DEFAULT_MIGRATION_PATH)/$$(date '+%s')_$(D).up.sql

migration-check:
	bash scripts/migration_check.sh

commit-check: SHELL := /bin/sh
commit-check:
	@bash scripts/commit_check.sh

version: SHELL := /bin/sh
version:
	@./scripts/version.sh

tag-version:
	bash scripts/tag_version.sh $(TARGET_COMMIT)

migration-wallet: DEFAULT_WALLET_MIGRATION_PATH := walletdatabase/migrations/sql
migration-wallet:
	touch $(DEFAULT_WALLET_MIGRATION_PATH)/$$(date +%s)_$(D).up.sql

install-git-hooks: SHELL := /bin/sh
install-git-hooks:
	@ln -sf $(if $(filter $(detected_OS), Linux),-r,) \
		$(GIT_ROOT)/githooks/* $(GIT_ROOT)/.git/hooks

-include install-git-hooks
.PHONY: install-git-hooks

migration-protocol: DEFAULT_PROTOCOL_PATH := protocol/migrations/sqlite
migration-protocol:
	touch $(DEFAULT_PROTOCOL_PATH)/$$(date +%s)_$(D).up.sql

codecov-validate: SHELL := /bin/sh
codecov-validate:
	curl -X POST --data-binary @.codecov.yml https://codecov.io/validate

.PHONY: pytest-lint
pytest-lint:
	$(MAKE) -C tests-functional lint

generate-db: ##@build Generate fake sqlite DBs in ./build directory for IDE SQL inspections
	LD_LIBRARY_PATH="$(RUNTIME_LIB_DIRS)" CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_CFLAGS="$(CGO_CFLAGS)" \
	go run tools/generate-db/main.go -out-dir build/db
