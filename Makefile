.PHONY: statusgo statusd-prune all test xgo clean help
.PHONY: statusgo-android statusgo-ios

help: ##@other Show this help
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

ifndef GOPATH
	$(error GOPATH not set. Please set GOPATH and make sure status-go is located at $$GOPATH/src/github.com/status-im/status-go. \
	For more information about the GOPATH environment variable, see https://golang.org/doc/code.html#GOPATH)
endif


EXPECTED_PATH=$(shell go env GOPATH)/src/github.com/status-im/status-go
ifneq ($(CURDIR),$(EXPECTED_PATH))
define NOT_IN_GOPATH_ERROR

Current dir is $(CURDIR), which seems to be different from your GOPATH.
Please, build status-go from GOPATH for proper build.
  GOPATH       = $(shell go env GOPATH)
  Current dir  = $(CURDIR)
  Expected dir = $(EXPECTED_PATH))
See https://golang.org/doc/code.html#GOPATH for more info

endef
$(error $(NOT_IN_GOPATH_ERROR))
endif

CGO_CFLAGS = -I/$(JAVA_HOME)/include -I/$(JAVA_HOME)/include/darwin
GOBIN = $(dir $(realpath $(firstword $(MAKEFILE_LIST))))build/bin
GIT_COMMIT = $(shell git describe --exact-match --tag 2>/dev/null || git rev-parse --short HEAD)
AUTHOR = $(shell echo $$USER)

BUILD_FLAGS ?= $(shell echo "-ldflags '-X main.buildStamp=`date -u '+%Y-%m-%d.%H:%M:%S'` -X github.com/status-im/status-go/params.Version=$(GIT_COMMIT)'")

XGO_GO ?= latest
XGOVERSION ?= 1.10.x
XGOIMAGE = statusteam/xgo:$(XGOVERSION)
XGOIMAGEIOSSIM = statusteam/xgo-ios-simulator:$(XGOVERSION)

networkid ?= StatusChain
gotest_extraflags =

DOCKER_IMAGE_NAME ?= statusteam/status-go
BOOTNODE_IMAGE_NAME ?= statusteam/bootnode
PROXY_IMAGE_NAME ?= statusteam/discovery-proxy
STATUSD_PRUNE_IMAGE_NAME ?= statusteam/statusd-prune

DOCKER_IMAGE_CUSTOM_TAG ?= $(GIT_COMMIT)

DOCKER_TEST_WORKDIR = /go/src/github.com/status-im/status-go/
DOCKER_TEST_IMAGE = golang:1.10

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

statusgo: ##@build Build status-go as statusd server
	go build -i -o $(GOBIN)/statusd -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./cmd/statusd
	@echo "Compilation done."
	@echo "Run \"build/bin/statusd -h\" to view available commands."

statusd-prune: ##@statusd-prune Build statusd-prune
	go build -o $(GOBIN)/statusd-prune -v ./cmd/statusd-prune
	@echo "Compilation done."
	@echo "Run \"build/bin/statusd-prune -h\" to view available commands."

statusd-prune-docker-image: ##@statusd-prune Build statusd-prune docker image
	@echo "Building docker image for ststusd-prune..."
	docker build --file _assets/build/Dockerfile-prune . \
		--label "commit=$(GIT_COMMIT)" \
		--label "author=$(AUTHOR)" \
		-t $(BOOTNODE_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG) \
		-t $(STATUSD_PRUNE_IMAGE_NAME):latest

bootnode: ##@build Build discovery v5 bootnode using status-go deps
	go build -i -o $(GOBIN)/bootnode -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./cmd/bootnode/
	@echo "Compilation done."

proxy: ##@build Build proxy for rendezvous servers using status-go deps
	go build -i -o $(GOBIN)/proxy -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./cmd/proxy/
	@echo "Compilation done."

mailserver-canary: ##@build Build mailserver canary using status-go deps
	go build -i -o $(GOBIN)/mailserver-canary -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./cmd/mailserver-canary/
	@echo "Compilation done."

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld $(GOBIN)/statusgo-*

statusgo-linux: xgo ##@cross-compile Build status-go for Linux
	./_assets/patches/patcher -b . -p geth-xgo
	$(GOPATH)/bin/xgo --image $(XGOIMAGE) --go=$(XGO_GO) -out statusgo --dest=$(GOBIN) --targets=linux/amd64 -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./cmd/statusd
	./_assets/patches/patcher -b . -p geth-xgo -r
	@echo "Android cross compilation done."

statusgo-android: xgo ##@cross-compile Build status-go for Android
	./_assets/patches/patcher -b . -p geth-xgo
	$(GOPATH)/bin/xgo --image $(XGOIMAGE) --go=$(XGO_GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./lib
	./_assets/patches/patcher -b . -p geth-xgo -r
	@echo "Android cross compilation done."

statusgo-ios: xgo	##@cross-compile Build status-go for iOS
	./_assets/patches/patcher -b . -p geth-xgo
	$(GOPATH)/bin/xgo --image $(XGOIMAGE) --go=$(XGO_GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./lib
	./_assets/patches/patcher -b . -p geth-xgo -r
	@echo "iOS framework cross compilation done."

statusgo-ios-simulator: xgo	##@cross-compile Build status-go for iOS Simulator
	@docker pull $(XGOIMAGEIOSSIM)
	./_assets/patches/patcher -b . -p geth-xgo
	$(GOPATH)/bin/xgo --image $(XGOIMAGEIOSSIM) --go=$(XGO_GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./lib
	./_assets/patches/patcher -b . -p geth-xgo -r
	@echo "iOS framework cross compilation done."

statusgo-library: ##@cross-compile Build status-go as static library for current platform
	@echo "Building static library..."
	go build -buildmode=c-archive -o $(GOBIN)/libstatus.a ./lib
	@echo "Static library built:"
	@ls -la $(GOBIN)/libstatus.*

docker-image: ##@docker Build docker image (use DOCKER_IMAGE_NAME to set the image name)
	@echo "Building docker image..."
	docker build --file _assets/build/Dockerfile . \
		--build-arg "build_tags=$(BUILD_TAGS)" \
		--build-arg "build_flags=$(BUILD_FLAGS)" \
		--label "commit=$(GIT_COMMIT)" \
		--label "author=$(AUTHOR)" \
		-t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG) \
		-t $(DOCKER_IMAGE_NAME):latest

bootnode-image:
	@echo "Building docker image for bootnode..."
	docker build --file _assets/build/Dockerfile-bootnode . \
		--build-arg "build_tags=$(BUILD_TAGS)" \
		--build-arg "build_flags=$(BUILD_FLAGS)" \
		--label "commit=$(GIT_COMMIT)" \
		--label "author=$(AUTHOR)" \
		-t $(BOOTNODE_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG) \
		-t $(BOOTNODE_IMAGE_NAME):latest

proxy-image:
	@echo "Building docker image for proxy..."
	docker build --file _assets/build/Dockerfile-proxy . \
		--build-arg "build_tags=$(BUILD_TAGS)" \
		--build-arg "build_flags=$(BUILD_FLAGS)" \
		-t $(PROXY_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG) \
		-t $(PROXY_IMAGE_NAME):latest

push-docker-images: docker-image bootnode-image
	docker push $(BOOTNODE_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG)
	docker push $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_CUSTOM_TAG)

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
	docker push $(BOOTNODE_IMAGE_NAME):latest
	docker push $(DOCKER_IMAGE_NAME):latest

xgo-docker-images: ##@docker Build xgo docker images
	@echo "Building xgo docker images..."
	docker build _assets/build/xgo/base -t $(XGOIMAGE)
	docker build _assets/build/xgo/ios-simulator -t $(XGOIMAGEIOSSIM)

xgo:
	docker pull $(XGOIMAGE)
	go get github.com/karalabe/xgo

setup: dep-install lint-install mock-install ##@other Prepare project for first build

mock-install: ##@other Install mocking tools
	go get -u github.com/golang/mock/mockgen

mock: ##@other Regenerate mocks
	mockgen -package=fcm          -destination=notifications/push/fcm/client_mock.go -source=notifications/push/fcm/client.go
	mockgen -package=fake         -destination=transactions/fake/mock.go             -source=transactions/fake/txservice.go
	mockgen -package=account      -destination=account/accounts_mock.go              -source=account/accounts.go
	mockgen -package=status       -destination=services/status/account_mock.go       -source=services/status/service.go
	mockgen -package=peer         -destination=services/peer/discoverer_mock.go      -source=services/peer/service.go

docker-test: ##@tests Run tests in a docker container with golang.
	docker run --privileged --rm -it -v "$(shell pwd):$(DOCKER_TEST_WORKDIR)" -w "$(DOCKER_TEST_WORKDIR)" $(DOCKER_TEST_IMAGE) go test ${ARGS}

test: test-unit ##@tests Run basic, short tests during development

test-unit: UNIT_TEST_PACKAGES = $(shell go list ./...  | \
	grep -v /vendor | \
	grep -v /t/e2e | \
	grep -v /t/benchmarks | \
	grep -v /lib)
test-unit: ##@tests Run unit and integration tests
	go test -v $(UNIT_TEST_PACKAGES) $(gotest_extraflags)

test-unit-race: gotest_extraflags=-race
test-unit-race: test-unit ##@tests Run unit and integration tests with -race flag

test-e2e: ##@tests Run e2e tests
	# order: reliability then alphabetical
	# TODO(tiabc): make a single command out of them adding `-p 1` flag.
	go test -timeout 5m ./t/e2e/accounts/... -network=$(networkid) $(gotest_extraflags)
	go test -timeout 5m ./t/e2e/api/... -network=$(networkid) $(gotest_extraflags)
	go test -timeout 5m ./t/e2e/node/... -network=$(networkid) $(gotest_extraflags)
	go test -timeout 20m ./t/e2e/rpc/... -network=$(networkid) $(gotest_extraflags)
	go test -timeout 20m ./t/e2e/whisper/... -network=$(networkid) $(gotest_extraflags)
	go test -timeout 10m ./t/e2e/transactions/... -network=$(networkid) $(gotest_extraflags)
	go test -timeout 10m ./t/e2e/services/... -network=$(networkid) $(gotest_extraflags)
	# e2e_test tag is required to include some files from ./lib without _test suffix
	go test -timeout 40m -tags e2e_test ./lib -network=$(networkid) $(gotest_extraflags)

test-e2e-race: gotest_extraflags=-race
test-e2e-race: test-e2e ##@tests Run e2e tests with -race flag

lint-install:
	@# The following installs a specific version of golangci-lint, which is appropriate for a CI server to avoid different results from build to build
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b $(GOPATH)/bin v1.9.1

lint:
	@echo "lint"
	@golangci-lint run ./...

ci: lint mock dep-ensure test-unit test-e2e ##@tests Run all linters and tests at once

clean: ##@other Cleanup
	rm -fr build/bin/*

deep-clean: clean
	rm -Rdf .ethereumtest/StatusChain

vendor-check: ##@dependencies Require all new patches and disallow other changes
	./_assets/patches/patcher -c
	./_assets/ci/isolate-vendor-check.sh

dep-ensure: ##@dependencies Dep ensure and apply all patches
	@dep ensure
	./_assets/patches/patcher

dep-install: ##@dependencies Install vendoring tool
	go get -u github.com/golang/dep/cmd/dep

update-geth: ##@dependencies Update geth (use GETH_BRANCH to optionally set the geth branch name)
	./_assets/ci/update-geth.sh $(GETH_BRANCH)
	@echo "**************************************************************"
	@echo "NOTE: Don't forget to:"
	@echo "- update the goleveldb dependency revision in Gopkg.toml to match the version used in go-ethereum"
	@echo "- reconcile any changes to interfaces in transactions/fake (such as PublicTransactionPoolAPI), which are copies from internal geth interfaces"
	@echo "**************************************************************"

patch: ##@patching Revert and apply all patches
	./_assets/patches/patcher

patch-revert: ##@patching Revert all patches only
	./_assets/patches/patcher -r
