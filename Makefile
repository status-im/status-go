.PHONY: statusgo statusd-prune all test clean help
.PHONY: statusgo-android statusgo-ios

RELEASE_TAG := v$(shell cat VERSION)
RELEASE_BRANCH := develop
RELEASE_DIR := /tmp/release-$(RELEASE_TAG)
PRE_RELEASE := "1"
RELEASE_TYPE := $(shell if [ $(PRE_RELEASE) = "0" ] ; then echo release; else echo pre-release ; fi)
GOLANGCI_BINARY=golangci-lint
IPFS_GATEWAY_URL ?= https://ipfs.status.im/

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

help: ##@other Show this help
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

CGO_CFLAGS = -I/$(JAVA_HOME)/include -I/$(JAVA_HOME)/include/darwin
GOPATH ?= $(HOME)/go

GIT_ROOT := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_AUTHOR ?= $(shell git config user.email || echo $$USER)

ENABLE_METRICS ?= true
BUILD_TAGS ?= gowaku_no_rln
BUILD_FLAGS ?= $(shell echo "-ldflags='\
	-X github.com/status-im/status-go/params.Version=$(RELEASE_TAG:v%=%) \
	-X github.com/status-im/status-go/params.GitCommit=$(GIT_COMMIT) \
	-X github.com/status-im/status-go/params.IpfsGatewayURL=$(IPFS_GATEWAY_URL) \
	-X github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=$(ENABLE_METRICS)'")

BUILD_FLAGS_MOBILE ?= $(shell echo "-ldflags='\
	-X github.com/status-im/status-go/params.Version=$(RELEASE_TAG:v%=%) \
	-X github.com/status-im/status-go/params.GitCommit=$(GIT_COMMIT) \
	-X github.com/status-im/status-go/params.IpfsGatewayURL=$(IPFS_GATEWAY_URL)'")

networkid ?= StatusChain

DOCKER_IMAGE_NAME ?= statusteam/status-go
BOOTNODE_IMAGE_NAME ?= statusteam/bootnode
STATUSD_PRUNE_IMAGE_NAME ?= statusteam/statusd-prune

DOCKER_IMAGE_CUSTOM_TAG ?= $(RELEASE_TAG)

DOCKER_TEST_WORKDIR = /go/src/github.com/status-im/status-go/
DOCKER_TEST_IMAGE = golang:1.13

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

GO_CMD_PATHS := $(filter-out library, $(wildcard cmd/*))
GO_CMD_NAMES := $(notdir $(GO_CMD_PATHS))
GO_CMD_BUILDS := $(addprefix build/bin/, $(GO_CMD_NAMES))

all: $(GO_CMD_NAMES)

.PHONY: $(GO_CMD_NAMES) $(GO_CMD_PATHS) $(GO_CMD_BUILDS)
$(GO_CMD_BUILDS): ##@build Build any Go project from cmd folder
	go build -mod=vendor -v \
		-tags '$(BUILD_TAGS)' $(BUILD_FLAGS) \
		-o ./$@ ./cmd/$(notdir $@)
	@echo "Compilation done."
	@echo "Run \"build/bin/$(notdir $@) -h\" to view available commands."

bootnode: ##@build Build discovery v5 bootnode using status-go deps
bootnode: build/bin/bootnode

node-canary: ##@build Build P2P node canary using status-go deps
node-canary: build/bin/node-canary

statusgo: ##@build Build status-go as statusd server
statusgo: build/bin/statusd
statusd: statusgo

statusd-prune: ##@statusd-prune Build statusd-prune
statusd-prune: build/bin/statusd-prune

spiff-workflow: ##@build Build node for SpiffWorkflow BPMN software
spiff-workflow: build/bin/spiff-workflow

statusd-prune-docker-image: ##@statusd-prune Build statusd-prune docker image
	@echo "Building docker image for ststusd-prune..."
	docker build --file _assets/build/Dockerfile-prune . \
		--label "commit=$(GIT_COMMIT)" \
		--label "author=$(GIT_AUTHOR)" \
		-t $(BOOTNODE_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG) \
		-t $(STATUSD_PRUNE_IMAGE_NAME):latest

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld build/bin/statusgo-*

statusgo-android: ##@cross-compile Build status-go for Android
	@echo "Building status-go for Android..."
	export GO111MODULE=off; \
	gomobile init; \
	gomobile bind -v \
		-target=android -ldflags="-s -w" \
		-tags '$(BUILD_TAGS)' \
		$(BUILD_FLAGS_MOBILE) \
		-o build/bin/statusgo.aar \
		github.com/status-im/status-go/mobile
	@echo "Android cross compilation done in build/bin/statusgo.aar"

statusgo-ios: ##@cross-compile Build status-go for iOS
	@echo "Building status-go for iOS..."
	export GO111MODULE=off; \
	gomobile init; \
	gomobile bind -v \
		-target=ios -ldflags="-s -w" \
		-tags 'nowatchdog $(BUILD_TAGS)' \
		$(BUILD_FLAGS_MOBILE) \
		-o build/bin/Statusgo.xcframework \
		github.com/status-im/status-go/mobile
	@echo "iOS framework cross compilation done in build/bin/Statusgo.xcframework"

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

docker-image: BUILD_TARGET ?= statusd
docker-image: ##@docker Build docker image (use DOCKER_IMAGE_NAME to set the image name)
	@echo "Building docker image..."
	docker build --file _assets/build/Dockerfile . \
		--build-arg "build_tags=$(BUILD_TAGS)" \
		--build-arg "build_flags=$(BUILD_FLAGS)" \
		--build-arg "build_target=$(BUILD_TARGET)" \
		--label "commit=$(GIT_COMMIT)" \
		--label "author=$(GIT_AUTHOR)" \
		-t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG) \
		-t $(DOCKER_IMAGE_NAME):latest

bootnode-image:
	@echo "Building docker image for bootnode..."
	docker build --file _assets/build/Dockerfile-bootnode . \
		--build-arg "build_tags=$(BUILD_TAGS)" \
		--build-arg "build_flags=$(BUILD_FLAGS)" \
		--label "commit=$(GIT_COMMIT)" \
		--label "author=$(GIT_AUTHOR)" \
		-t $(BOOTNODE_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG) \
		-t $(BOOTNODE_IMAGE_NAME):latest

push-docker-images: docker-image bootnode-image
	docker push $(BOOTNODE_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG)
	docker push $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG)

clean-docker-images:
	docker rmi -f $(shell docker image ls --filter="reference=$(DOCKER_IMAGE_NAME)" --quiet)

# See https://www.gnu.org/software/make/manual/html_node/Target_002dspecific.html to understand this magic.
push-docker-images-latest: GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
push-docker-images-latest: GIT_LOCAL  = $(shell git rev-parse @)
push-docker-images-latest: GIT_REMOTE = $(shell git fetch -q && git rev-parse remotes/origin/develop || echo 'NO_DEVELOP')
push-docker-images-latest: docker-image bootnode-image
	@echo "Pushing latest docker images..."
	@echo "Checking git branch..."
ifneq ("$(GIT_BRANCH)", "develop")
	$(error You should only use develop branch to push the latest tag!)
	exit 1
endif
ifneq ("$(GIT_LOCAL)", "$(GIT_REMOTE)")
	$(error The local git commit does not match the remote origin!)
	exit 1
endif
	docker push $(BOOTNODE_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG)
	docker push $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG)

setup: ##@setup Install all tools
setup: setup-check setup-build setup-dev tidy

setup-check: ##@setup Check if Go compiler is installed.
ifeq (, $(shell which go))
	$(error "No Go compiler found! Make sure to install 1.19.0 or newer.")
endif

setup-dev: ##@setup Install all necessary tools for development
setup-dev: install-lint install-mock install-modvendor install-protobuf tidy install-os-deps

setup-build: ##@setup Install all necessary build tools
setup-build: install-lint install-release install-gomobile install-junit-report

install-os-deps: ##@install Operating System Dependencies
	_assets/scripts/install_deps.sh

install-gomobile: install-xtools
install-gomobile: ##@install Go Mobile Build Tools
	GO111MODULE=on  go install golang.org/x/mobile/cmd/...@5d9a3325
	GO111MODULE=on  go mod download golang.org/x/exp@ec7cb31e
	GO111MODULE=off go get -d golang.org/x/mobile/cmd/gobind

install-lint: ##@install Install Linting Tools
	GO111MODULE=on go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2

install-junit-report: ##@install Install Junit Report Tool for Jenkins integration
	GO111MODULE=on go install github.com/jstemmer/go-junit-report/v2@latest

install-mock: ##@install Install Module Mocking Tools
	GO111MODULE=on go install github.com/golang/mock/mockgen@v1.4.4

install-modvendor: ##@install Install Module Vendoring Tool
	GO111MODULE=on go install github.com/goware/modvendor@v0.5.0

install-protobuf: ##@install Install Protobuf Generation Tools
	GO111MODULE=on go install github.com/kevinburke/go-bindata/go-bindata@v3.13.0
	GO111MODULE=on go install github.com/golang/protobuf/protoc-gen-go@v1.3.4

install-release: ##@install Install Github Release Tools
	GO111MODULE=on go install github.com/c4milo/github-release@v1.1.0

install-xtools: ##@install Install Miscellaneous Go Tools
	GO111MODULE=on go install golang.org/x/tools/go/packages/...@v0.1.5

generate-handlers:
	go generate ./_assets/generate_handlers/
generate: ##@other Regenerate assets and other auto-generated stuff
	go generate ./static ./static/mailserver_db_migrations ./t ./multiaccounts/... ./appdatabase/... ./protocol/... ./walletdatabase/... ./_assets/generate_handlers

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
		-and -not -wholename '*/vendor/*' \
		-exec goimports \
		-local 'github.com/ethereum/go-ethereum,github.com/status-im/status-go,github.com/status-im/markdown' \
		-w {} \;
	$(MAKE) vendor

mock: ##@other Regenerate mocks
	mockgen -package=fake         -destination=transactions/fake/mock.go             -source=transactions/fake/txservice.go
	mockgen -package=status       -destination=services/status/account_mock.go       -source=services/status/service.go
	mockgen -package=peer         -destination=services/peer/discoverer_mock.go      -source=services/peer/service.go

docker-test: ##@tests Run tests in a docker container with golang.
	docker run --privileged --rm -it -v "$(shell pwd):$(DOCKER_TEST_WORKDIR)" -w "$(DOCKER_TEST_WORKDIR)" $(DOCKER_TEST_IMAGE) go test ${ARGS}

test: test-unit ##@tests Run basic, short tests during development

test-unit: export BUILD_TAGS ?=
test-unit: export UNIT_TEST_FAILFAST ?= true
# Ensure 'waku' and 'wakuv2' tests are executed first to reduce the impact of flaky tests.
# Otherwise, the entire target might fail at the end, making re-runs time-consuming.
test-unit: export UNIT_TEST_PACKAGES ?= $(shell go list ./... | \
	grep -v /vendor | \
	grep -v /t/e2e | \
	grep -v /t/benchmarks | \
	grep -v /transactions/fake | \
	grep -E -v '/wakuv2(/.*|$$)')
test-unit: export UNIT_TEST_PACKAGES_WITH_EXTENDED_TIMEOUT ?= github.com/status-im/status-go/protocol
test-unit: ##@tests Run unit and integration tests
	./_assets/scripts/run_unit_tests.sh

test-unit-race: export GOTEST_EXTRAFLAGS=-race
test-unit-race: test-unit ##@tests Run unit and integration tests with -race flag

test-e2e: ##@tests Run e2e tests
	# order: reliability then alphabetical
	# TODO(tiabc): make a single command out of them adding `-p 1` flag.

test-e2e-race: export GOTEST_EXTRAFLAGS=-race
test-e2e-race: test-e2e ##@tests Run e2e tests with -race flag

canary-test: node-canary
	# TODO: uncomment that!
	#_assets/scripts/canary_test_mailservers.sh ./config/cli/fleet-eth.prod.json

lint:
	@echo "lint"
	@golangci-lint --exclude=SA1019 run ./... --deadline=5m

ci: lint canary-test test-unit test-e2e ##@tests Run all linters and tests at once

ci-race: lint canary-test test-unit test-e2e-race ##@tests Run all linters and tests at once + race

clean: ##@other Cleanup
	rm -fr build/bin/* mailserver-config.json

git-clean:
	git clean -xf

deep-clean: clean git-clean
	rm -Rdf .ethereumtest/StatusChain

tidy:
	go mod tidy

vendor:
	go mod tidy
	go mod vendor
	modvendor -copy="**/*.c **/*.h" -v
.PHONY: vendor

update-fleet-config: ##@other Update fleets configuration from fleets.status.im
	./_assets/scripts/update-fleet-config.sh
	@echo "Updating static assets..."
	@go generate ./static
	@echo "Done"

run-bootnode-systemd: ##@Easy way to run a bootnode locally with Docker Compose
	@cd _assets/systemd/bootnode && $(MAKE)

run-bootnode-docker: ##@Easy way to run a bootnode locally with Docker Compose
	@cd _assets/compose/bootnode && $(MAKE)

run-mailserver-systemd: ##@Easy Run a mailserver locally with systemd
	@cd _assets/systemd/mailserver && $(MAKE)

run-mailserver-docker: ##@Easy Run a mailserver locally with Docker Compose
	@cd _assets/compose/mailserver && $(MAKE)

clean-bootnode-systemd: ##@Easy Clean your systemd service for running a bootnode
	@cd _assets/systemd/bootnode && $(MAKE) clean

clean-bootnode-docker: ##@Easy Clean your Docker container running a bootnode
	@cd _assets/compose/bootnode && $(MAKE) clean

clean-mailserver-systemd: ##@Easy Clean your systemd service for running a mailserver
	@cd _assets/systemd/mailserver && $(MAKE) clean

clean-mailserver-docker: ##@Easy Clean your Docker container running a mailserver
	@cd _assets/compose/mailserver && $(MAKE) clean

migration: DEFAULT_MIGRATION_PATH := appdatabase/migrations/sql
migration:
	touch $(DEFAULT_MIGRATION_PATH)/$(shell date +%s)_$(D).up.sql

migration-check:
	bash _assets/scripts/migration_check.sh

migration-wallet: DEFAULT_WALLET_MIGRATION_PATH := walletdatabase/migrations/sql
migration-wallet:
	touch $(DEFAULT_WALLET_MIGRATION_PATH)/$(shell date +%s)_$(D).up.sql

install-git-hooks:
	@ln -sf $(if $(filter $(detected_OS), Linux),-r,) \
		$(GIT_ROOT)/_assets/hooks/* $(GIT_ROOT)/.git/hooks

-include install-git-hooks
.PHONY: install-git-hooks

migration-protocol: DEFAULT_PROTOCOL_PATH := protocol/migrations/sqlite
migration-protocol:
	touch $(DEFAULT_PROTOCOL_PATH)/$(shell date +%s)_$(D).up.sql

PROXY_WRAPPER_PATH = $(CURDIR)/vendor/github.com/siphiuel/lc-proxy-wrapper
-include $(PROXY_WRAPPER_PATH)/Makefile.vars

#export VERIF_PROXY_OUT_PATH = $(CURDIR)/vendor/github.com/siphiuel/lc-proxy-wrapper
build-verif-proxy:
	$(MAKE) -C $(NIMBUS_ETH1_PATH) libverifproxy

build-verif-proxy-wrapper:
	$(MAKE) -C $(VERIF_PROXY_OUT_PATH) build-verif-proxy-wrapper

test-verif-proxy-wrapper:
	CGO_CFLAGS="$(CGO_CFLAGS)" go test -v github.com/status-im/status-go/rpc -tags gowaku_skip_migrations,nimbus_light_client -run ^TestProxySuite$$ -testify.m TestRun -ldflags $(LDFLAGS)
