.PHONY: statusgo all test clean help
.PHONY: statusgo-ios-library statusgo-android-library
.PHONY: build-libwaku test-libwaku clean-libwaku rebuild-libwaku

# Clear any GOROOT set outside of the Nix shell
export GOROOT=

# This is a code for automatic help generator.
# It supports ANSI colors and categories.
# To add new item into help output, simply add comments
# starting with '##'. To add category, use @category.
GREEN  := $(shell echo "\e[32m")
WHITE  := $(shell echo "\e[37m")
YELLOW := $(shell echo "\e[33m")
RESET  := $(shell echo "\e[0m")
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

RELEASE_TAG ?= $(shell ./_assets/scripts/version.sh)
RELEASE_DIR ?= /tmp/release-$(RELEASE_TAG)
GOLANGCI_BINARY = golangci-lint

ifeq ($(OS),Windows_NT)     # is Windows_NT on XP, 2000, 7, Vista, 10...
 detected_OS := Windows
else
 detected_OS := $(strip $(shell uname))
endif

ifeq ($(MAKECMDGOALS),statusgo-android-library)
    ARCH ?= arm64
    ANDROID_NDK_ROOT ?= $(shell find /nix/store -path "*android-sdk-ndk-27.2.12479018/libexec/android-sdk/ndk/27.2.12479018" -type d 2>/dev/null | head -1)
    ANDROID_API ?= 28
    HOST_OS ?= linux
    ifeq ($(ARCH),x86_64)
        MOBILE_GOARCH := amd64
    else
        MOBILE_GOARCH := $(ARCH)
    endif
    ANDROID_BUILD_FLAGS := CC="$(ANDROID_NDK_ROOT)/toolchains/llvm/prebuilt/$(HOST_OS)-x86_64/bin/clang --target=aarch64-linux-android$(ANDROID_API) --sysroot=$(ANDROID_NDK_ROOT)/toolchains/llvm/prebuilt/$(HOST_OS)-x86_64/sysroot" CGO_CFLAGS="-Os -flto -fembed-bitcode" CGO_LDFLAGS="-Os -flto" CGO_ENABLED=1 GOOS=android GOARCH=$(MOBILE_GOARCH)
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
    IOS_BUILD_FLAGS := CC="$(shell xcrun --sdk $(IPHONE_SDK) --find clang)" CGO_CFLAGS="-Os -flto -arch $(ARCH) -isysroot $(shell xcrun --sdk $(IPHONE_SDK) --show-sdk-path) -miphoneos-version-min=$(IOS_TARGET) -fembed-bitcode" CGO_LDFLAGS="-Os -flto" CGO_ENABLED=1 GOOS=ios GOARCH=$(MOBILE_GOARCH)
endif

ifeq ($(detected_OS),Darwin)
 GOBIN_SHARED_LIB_EXT := dylib
 LIBWAKU_EXT := so
 GOBIN_SHARED_LIB_CFLAGS := CGO_ENABLED=1 GOOS=darwin
else ifeq ($(detected_OS),Windows)
 GOBIN_SHARED_LIB_EXT := dll
 LIBWAKU_EXT := dll
 GOBIN_SHARED_LIB_CGO_LDFLAGS := CGO_LDFLAGS=""
else
 GOBIN_SHARED_LIB_EXT := so
 LIBWAKU_EXT := so
 GOBIN_SHARED_LIB_CGO_LDFLAGS := CGO_LDFLAGS="-Wl,-soname,libstatus.so.0"
endif

CGO_CFLAGS = -I/$(JAVA_HOME)/include -I/$(JAVA_HOME)/include/darwin
export GOPATH ?= $(HOME)/go

GIT_ROOT ?= $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_AUTHOR ?= $(shell git config user.email || echo $$USER)

BUILD_TAGS ?= gowaku_no_rln

ifeq ($(USE_NWAKU), true)
BUILD_TAGS += use_nwaku
endif

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

.PHONY: $(GO_CMD_NAMES) $(GO_CMD_PATHS) $(GO_CMD_BUILDS)
$(GO_CMD_BUILDS): generate
$(GO_CMD_BUILDS): ##@build Build any Go project from cmd folder
	go build -mod=vendor -v \
		-tags '$(BUILD_TAGS)' $(BUILD_FLAGS) \
		-o ./$@ ./cmd/$(notdir $@)
	@echo "Compilation done."
	@echo "Run \"build/bin/$(notdir $@) -h\" to view available commands."

LIBWAKU := $(CURDIR)/vendor/github.com/waku-org/waku-go-bindings/third_party/nwaku/build/libwaku.$(LIBWAKU_EXT)
$(LIBWAKU):
ifeq ($(USE_NWAKU),true)
	@echo "Building libwaku"
	$(MAKE) -C $(CURDIR)/vendor/github.com/waku-org/waku-go-bindings/waku SHELL=/bin/bash
endif

statusgo: ##@build Build status-go as status-backend server
statusgo: build/bin/status-backend

status-backend: ##@build Build status-backend to run status-go as HTTP server
status-backend: build/bin/status-backend

run-status-backend: PORT ?= 0
run-status-backend: generate
run-status-backend: ##@run Start status-backend server listening to localhost:PORT
	go run ./cmd/status-backend --address localhost:${PORT}

push-notification-server: ##@build Build push-notification-server
push-notification-server: build/bin/push-notification-server

cmd: ##@build Build all public apps in ./cmd
cmd: status-backend push-notification-server


status-go-deps:
	go clean -cache || true
	go clean -modcache || true
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1



statusgo-c-bindings:
	## cmd/library/README.md explains the magic incantation behind this
	mkdir -p build/bin/statusgo-lib
	go run cmd/library/*.go > build/bin/statusgo-lib/main.go

statusgo-library: generate
statusgo-library: statusgo-c-bindings $(LIBWAKU) ##@cross-compile Build status-go as static library for current platform
	@echo "Building static library..."
	go build \
		-tags '$(BUILD_TAGS)' \
		$(BUILD_FLAGS) \
		-buildmode=c-archive \
		-o build/bin/libstatus.a \
		./build/bin/statusgo-lib
	@echo "Static library built:"
	@ls -la build/bin/libstatus.*

build-libwaku: $(LIBWAKU)

statusgo-shared-library: generate
statusgo-shared-library: statusgo-c-bindings $(LIBWAKU) ##@cross-compile Build status-go as shared library for current platform
	@echo "Building shared library..."
	@echo "Tags: $(BUILD_TAGS)"
	$(GOBIN_SHARED_LIB_CFLAGS) $(GOBIN_SHARED_LIB_CGO_LDFLAGS) go build \
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

statusgo-android-library: generate statusgo-c-bindings $(LIBWAKU) ##@cross-compile Build status-go as Android mobile library
	@echo "Building Android mobile library..."
	@echo "MOBILE_GOARCH: $(MOBILE_GOARCH)"
	@echo "Android build flags: $(ANDROID_BUILD_FLAGS)"
	$(ANDROID_BUILD_FLAGS) go build -buildmode=c-shared -tags 'gowaku_no_rln nowatchdog disable_torrent' \
		-ldflags="-checklinkname=0 -X github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=true" \
		-o "build/bin/libstatus.so" ./build/bin/statusgo-lib
	@echo "Android library built"
	@file build/bin/libstatus.so

statusgo-ios-library: generate statusgo-c-bindings $(LIBWAKU) ##@cross-compile Build status-go as iOS mobile library
	@echo "Building iOS mobile library..."
	@echo "MOBILE_GOARCH: $(MOBILE_GOARCH)"
	@echo "iOS build flags: $(IOS_BUILD_FLAGS)"
	$(IOS_BUILD_FLAGS) go build -buildmode=c-archive -tags 'gowaku_no_rln nowatchdog disable_torrent' \
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

generate: PACKAGES ?= $$(go list -e ./... | grep -v "/contracts/")
generate: GO_GENERATE_CMD ?= $$(which go-generate-fast || echo 'go generate')
generate: export GO_GENERATE_FAST_DEBUG ?= false
generate: export GO_GENERATE_FAST_RECACHE ?= false
generate:  ##@ Run generate for all given packages using go-generate-fast, fallback to `go generate` (e.g. for docker)
	@GOROOT=$$(go env GOROOT) $(GO_GENERATE_CMD) $(PACKAGES)

generate-contracts:
	go generate ./contracts
download-tokens:
	go run ./services/wallet/token/token-lists/default-lists/downloader/main.go
analyze-token-stores:
	go run ./services/wallet/token/token-lists/analyzer/main.go

prepare-release: clean-release
	mkdir -p $(RELEASE_DIR)
	zip -r $(RELEASE_DIR)/status-go-desktop.zip . -x *.git*
	${MAKE} clean

clean-release:
	rm -rf $(RELEASE_DIR)

lint-fix:
	find . \
		-name '*.go' \
		-and -not -name '*.pb.go' \
		-and -not -name 'bindata*' \
		-and -not -name 'migrations.go' \
		-and -not -name 'messenger_handlers.go' \
		-and -not -name '*/mock/*' \
		-and -not -name 'mock.go' \
		-and -not -wholename '*/vendor/*' \
		-exec goimports \
		-local 'github.com/ethereum/go-ethereum,github.com/status-im/status-go,github.com/status-im/markdown' \
		-w {} \;
	$(MAKE) vendor

docker-test: ##@tests Run tests in a docker container with golang.
	docker run --privileged --rm -it -v "$(PWD):$(DOCKER_TEST_WORKDIR)" -w "$(DOCKER_TEST_WORKDIR)" $(DOCKER_TEST_IMAGE) go test ${ARGS}

test-libwaku: | $(LIBWAKU)
	go test -tags '$(BUILD_TAGS) use_nwaku' -run TestDial ./wakuv2/... -count 1 -v -json | jq -r '.Output'

clean-libwaku:
	@echo "Removing libwaku"
	rm $(LIBWAKU)

rebuild-libwaku: | clean-libwaku $(LIBWAKU)

test: test-unit ##@tests Run basic, short tests during development

test-unit-prep: generate
test-unit-prep: export BUILD_TAGS ?=
test-unit-prep: export UNIT_TEST_DRY_RUN ?= false
test-unit-prep: export UNIT_TEST_COUNT ?= 1
test-unit-prep: export UNIT_TEST_FAILFAST ?= true
test-unit-prep: export UNIT_TEST_USE_DEVELOPMENT_LOGGER ?= true
test-unit-prep: export UNIT_TEST_REPORT_CODECOV ?= false

test-unit: test-unit-prep
test-unit: export UNIT_TEST_RERUN_FAILS ?= true
test-unit: export UNIT_TEST_PACKAGES ?= $(call sh, go list ./... | \
	grep -v /vendor | \
	grep -v /t/e2e | \
	grep -v /t/benchmarks | \
	grep -v /transactions/fake | \
	grep -v /tests-unit-network)
test-unit: ##@tests Run unit and integration tests
	./_assets/scripts/run_unit_tests.sh

test-unit-network: test-unit-prep
test-unit-network: export UNIT_TEST_RERUN_FAILS ?= false
test-unit-network: export UNIT_TEST_PACKAGES ?= $(call sh, go list ./tests-unit-network/...)
test-unit-network: ##@tests Run unit and integration tests with network access
	./_assets/scripts/run_unit_tests.sh

test-unit-race: export GOTEST_EXTRAFLAGS=-race
test-unit-race: test-unit ##@tests Run unit and integration tests with -race flag

test-functional: generate
test-functional: export FUNCTIONAL_TESTS_DOCKER_UID ?= $(call sh, id -u)
test-functional: export FUNCTIONAL_TESTS_REPORT_CODECOV ?= false
test-functional:
	@./_assets/scripts/run_functional_tests.sh

benchmark: export FUNCTIONAL_TESTS_DOCKER_UID ?= $(call sh, id -u)
benchmark:
	@./_assets/scripts/run_benchmark.sh

lint-panics: export GOFLAGS ?= -tags='$(BUILD_TAGS)'
lint-panics: generate
	go run ./cmd/lint-panics -root="$(PWD)" -skip=./cmd -test=false ./...

lint: generate lint-panics
	golangci-lint --build-tags '$(BUILD_TAGS)' run ./...

clean: ##@other Cleanup
	rm -fr build/bin/*

git-clean:
	git clean -xf

deep-clean: clean git-clean
	rm -Rdf .ethereumtest/StatusChain

tidy:
	go mod tidy

vendor: generate
	go mod tidy
	go mod vendor
	go tool modvendor -copy="**/*.c **/*.h" -v
.PHONY: vendor

migration: DEFAULT_MIGRATION_PATH := appdatabase/migrations/sql
migration:
	touch $(DEFAULT_MIGRATION_PATH)/$$(date '+%s')_$(D).up.sql

migration-check:
	bash _assets/scripts/migration_check.sh

commit-check: SHELL := /bin/sh
commit-check:
	@bash _assets/scripts/commit_check.sh

version: SHELL := /bin/sh
version:
	@./_assets/scripts/version.sh

tag-version:
	bash _assets/scripts/tag_version.sh $(TARGET_COMMIT)

migration-wallet: DEFAULT_WALLET_MIGRATION_PATH := walletdatabase/migrations/sql
migration-wallet:
	touch $(DEFAULT_WALLET_MIGRATION_PATH)/$$(date +%s)_$(D).up.sql

install-git-hooks: SHELL := /bin/sh
install-git-hooks:
	@ln -sf $(if $(filter $(detected_OS), Linux),-r,) \
		$(GIT_ROOT)/_assets/hooks/* $(GIT_ROOT)/.git/hooks

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

generate-db: build/bin/generate-db
generate-db: ##@build Generate fake sqlite DBs in ./build directory for IDE SQL inspections
	./build/bin/generate-db -out-dir build/db
