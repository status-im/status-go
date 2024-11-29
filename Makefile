.PHONY: statusgo all test clean help
.PHONY: statusgo-android statusgo-ios

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

ifeq ($(detected_OS),Darwin)
 GOBIN_SHARED_LIB_EXT := dylib
 GOBIN_SHARED_LIB_CFLAGS := CGO_ENABLED=1 GOOS=darwin
else ifeq ($(detected_OS),Windows)
 GOBIN_SHARED_LIB_EXT := dll
 GOBIN_SHARED_LIB_CGO_LDFLAGS := CGO_LDFLAGS=""
else
 GOBIN_SHARED_LIB_EXT := so
 GOBIN_SHARED_LIB_CGO_LDFLAGS := CGO_LDFLAGS="-Wl,-soname,libstatus.so.0"
endif

CGO_CFLAGS = -I/$(JAVA_HOME)/include -I/$(JAVA_HOME)/include/darwin
export GOPATH ?= $(HOME)/go

GIT_ROOT ?= $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_AUTHOR ?= $(shell git config user.email || echo $$USER)

ENABLE_METRICS ?= true
BUILD_TAGS ?= gowaku_no_rln

BUILD_FLAGS ?= -ldflags="-X github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=$(ENABLE_METRICS)"
BUILD_FLAGS_MOBILE ?=

networkid ?= StatusChain

DOCKER_IMAGE_NAME ?= statusteam/status-go
DOCKER_IMAGE_CUSTOM_TAG ?= $(RELEASE_TAG)
DOCKER_TEST_WORKDIR = /go/src/github.com/status-im/status-go/
DOCKER_TEST_IMAGE = golang:1.13

GO_CMD_PATHS := $(filter-out library, $(wildcard cmd/*))
GO_CMD_NAMES := $(notdir $(GO_CMD_PATHS))
GO_CMD_BUILDS := $(addprefix build/bin/, $(GO_CMD_NAMES))

# Our custom config is located in nix/nix.conf
export NIX_USER_CONF_FILES = $(PWD)/nix/nix.conf
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
TARGET := default
ifneq ($(detected_OS),Windows)
 SHELL := ./nix/scripts/shell.sh
endif
shell: export TARGET ?= default
shell: ##@prepare Enter into a pre-configured shell
ifndef IN_NIX_SHELL
	@ENTER_NIX_SHELL
else
	@echo "${YELLOW}Nix shell is already active$(RESET)"
endif

nix-repl: SHELL := /bin/sh
nix-repl: ##@nix Start an interactive Nix REPL
	nix repl shell.nix

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
		-o ./$@ ./cmd/$(notdir $@) ;\
	echo "Compilation done." ;\
	echo "Run \"build/bin/$(notdir $@) -h\" to view available commands."

statusgo: ##@build Build status-go as statusd server
statusgo: build/bin/statusd
statusd: statusgo

status-cli: ##@build Build status-cli to send messages
status-cli: build/bin/status-cli

status-backend: ##@build Build status-backend to run status-go as HTTP server
status-backend: build/bin/status-backend

run-status-backend: PORT ?= 0
run-status-backend: generate
run-status-backend: ##@run Start status-backend server listening to localhost:PORT
	go run ./cmd/status-backend --address localhost:${PORT}

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld build/bin/statusgo-*

status-go-deps:
	go install go.uber.org/mock/mockgen@v0.4.0
	go install github.com/kevinburke/go-bindata/v4/...@v4.0.2
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1

statusgo-android: generate
statusgo-android: ##@cross-compile Build status-go for Android
	@echo "Building status-go for Android..."
	export GO111MODULE=off; \
	gomobile init; \
	gomobile bind -v \
		-target=android -ldflags="-s -w" \
		-tags '$(BUILD_TAGS) disable_torrent' \
		$(BUILD_FLAGS_MOBILE) \
		--androidapi="23" \
		-o build/bin/statusgo.aar \
		github.com/status-im/status-go/mobile
	@echo "Android cross compilation done in build/bin/statusgo.aar"

statusgo-ios: generate
statusgo-ios: ##@cross-compile Build status-go for iOS
	@echo "Building status-go for iOS..."
	export GO111MODULE=off; \
	gomobile init; \
	gomobile bind -v \
		-target=ios -ldflags="-s -w" \
		-tags 'nowatchdog $(BUILD_TAGS) disable_torrent' \
		$(BUILD_FLAGS_MOBILE) \
		-o build/bin/Statusgo.xcframework \
		github.com/status-im/status-go/mobile
	@echo "iOS framework cross compilation done in build/bin/Statusgo.xcframework"

statusgo-library: generate
statusgo-library: ##@cross-compile Build status-go as static library for current platform
	## cmd/library/README.md explains the magic incantation behind this
	mkdir -p build/bin/statusgo-lib
	go run cmd/library/*.go > build/bin/statusgo-lib/main.go
	@echo "Building static library..."
	go build \
		-tags '$(BUILD_TAGS)' \
		$(BUILD_FLAGS) \
		-buildmode=c-archive \
		-o build/bin/libstatus.a \
		./build/bin/statusgo-lib
	@echo "Static library built:"
	@ls -la build/bin/libstatus.*

statusgo-shared-library: generate
statusgo-shared-library: ##@cross-compile Build status-go as shared library for current platform
	## cmd/library/README.md explains the magic incantation behind this
	mkdir -p build/bin/statusgo-lib
	go run cmd/library/*.go > build/bin/statusgo-lib/main.go
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

docker-image: SHELL := /bin/sh
docker-image: BUILD_TARGET ?= statusd
docker-image: ##@docker Build docker image (use DOCKER_IMAGE_NAME to set the image name)
	@echo "Building docker image..."
	docker build --file _assets/build/Dockerfile . \
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
download-uniswap-tokens:
	go run ./services/wallet/token/downloader/main.go

prepare-release: clean-release
	mkdir -p $(RELEASE_DIR)
	mv build/bin/statusgo.aar $(RELEASE_DIR)/status-go-android.aar
	zip -r build/bin/Statusgo.xcframework.zip build/bin/Statusgo.xcframework
	mv build/bin/Statusgo.xcframework.zip $(RELEASE_DIR)/status-go-ios.zip
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

test-e2e: ##@tests Run e2e tests
	# order: reliability then alphabetical
	# TODO(tiabc): make a single command out of them adding `-p 1` flag.

test-e2e-race: export GOTEST_EXTRAFLAGS=-race
test-e2e-race: test-e2e ##@tests Run e2e tests with -race flag

test-functional: generate
test-functional: export FUNCTIONAL_TESTS_DOCKER_UID ?= $(call sh, id -u)
test-functional: export FUNCTIONAL_TESTS_REPORT_CODECOV ?= false
test-functional:
	@./_assets/scripts/run_functional_tests.sh

lint-panics: generate
	go run ./cmd/lint-panics -root="$(call sh, pwd)" -skip=./cmd -test=false ./...

lint: generate lint-panics
	golangci-lint run ./...

ci: generate lint test-unit test-e2e ##@tests Run all linters and tests at once

ci-race: generate lint test-unit test-e2e-race ##@tests Run all linters and tests at once + race

clean: ##@other Cleanup
	rm -fr build/bin/* mailserver-config.json

git-clean:
	git clean -xf

deep-clean: clean git-clean
	rm -Rdf .ethereumtest/StatusChain

tidy:
	go mod tidy

vendor: generate
	go mod tidy
	go mod vendor
	modvendor -copy="**/*.c **/*.h" -v
.PHONY: vendor

update-fleet-config: ##@other Update fleets configuration from fleets.status.im
	./_assets/scripts/update-fleet-config.sh
	@echo "Updating static assets..."
	@go generate ./static
	@echo "Done"

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

PROXY_WRAPPER_PATH = $(CURDIR)/vendor/github.com/siphiuel/lc-proxy-wrapper
-include $(PROXY_WRAPPER_PATH)/Makefile.vars

#export VERIF_PROXY_OUT_PATH = $(CURDIR)/vendor/github.com/siphiuel/lc-proxy-wrapper
build-verif-proxy:
	$(MAKE) -C $(NIMBUS_ETH1_PATH) libverifproxy

build-verif-proxy-wrapper:
	$(MAKE) -C $(VERIF_PROXY_OUT_PATH) build-verif-proxy-wrapper

test-verif-proxy-wrapper:
	CGO_CFLAGS="$(CGO_CFLAGS)" go test -v github.com/status-im/status-go/rpc -tags gowaku_skip_migrations,nimbus_light_client -run ^TestProxySuite$$ -testify.m TestRun -ldflags $(LDFLAGS)

run-anvil: SHELL := /bin/sh
run-anvil:
	docker-compose -f tests-functional/docker-compose.anvil.yml up --remove-orphans

codecov-validate: SHELL := /bin/sh
codecov-validate:
	curl -X POST --data-binary @.codecov.yml https://codecov.io/validate
