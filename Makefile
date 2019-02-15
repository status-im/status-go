.PHONY: statusgo statusd-prune all test clean help
.PHONY: statusgo-android statusgo-ios

RELEASE_TAG := $(shell cat VERSION)
RELEASE_BRANCH := "develop"
RELEASE_DIRECTORY := /tmp/release-$(RELEASE_TAG)
PRE_RELEASE := "1"

# sets Makefile GOPATH2 var to the first entry in the GOPATH env var
GOPATH2 := $(word 1, $(subst :, ,$(shell go env GOPATH)))

help: ##@other Show this help
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

ifndef GOPATH2
	$(error GOPATH not set. Please set GOPATH and make sure status-go is located at $$GOPATH/src/github.com/status-im/status-go. \
	For more information about the GOPATH environment variable, see https://golang.org/doc/code.html#GOPATH)
endif


EXPECTED_PATH=$(GOPATH2)/src/github.com/status-im/status-go
ifneq ($(CURDIR),$(EXPECTED_PATH))
define NOT_IN_GOPATH_ERROR

Current dir is $(CURDIR), which seems to be different from your GOPATH.
Please, build status-go from GOPATH for proper build.
If your GOPATH has multiple entries, build status-go from the first entry in your GOPATH.
  GOPATH       = $(shell go env GOPATH)
  Current dir  = $(CURDIR)
  Expected dir = $(EXPECTED_PATH)
See https://golang.org/doc/code.html#GOPATH for more info

endef
$(error $(NOT_IN_GOPATH_ERROR))
endif

CGO_CFLAGS = -I/$(JAVA_HOME)/include -I/$(JAVA_HOME)/include/darwin
GOBIN = $(dir $(realpath $(firstword $(MAKEFILE_LIST))))build/bin
GIT_COMMIT = $(shell git rev-parse --short HEAD)
AUTHOR = $(shell echo $$USER)

ENABLE_METRICS ?= true
BUILD_FLAGS ?= $(shell echo "-ldflags '\
	-X main.buildStamp=`date -u '+%Y-%m-%d.%H:%M:%S'` \
	-X github.com/status-im/status-go/params.Version=$(RELEASE_TAG) \
	-X github.com/status-im/status-go/params.GitCommit=$(GIT_COMMIT) \
	-X github.com/status-im/status-go/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=$(ENABLE_METRICS)'")

networkid ?= StatusChain
gotest_extraflags =

DOCKER_IMAGE_NAME ?= statusteam/status-go
BOOTNODE_IMAGE_NAME ?= statusteam/bootnode
PROXY_IMAGE_NAME ?= statusteam/discovery-proxy
STATUSD_PRUNE_IMAGE_NAME ?= statusteam/statusd-prune

DOCKER_IMAGE_CUSTOM_TAG ?= $(RELEASE_TAG)

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

node-canary: ##@build Build P2P node canary using status-go deps
	go build -i -o $(GOBIN)/node-canary -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./cmd/node-canary/
	@echo "Compilation done."

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld $(GOBIN)/statusgo-*

statusgo-android: ##@cross-compile Build status-go for Android
	@echo "Building status-go for Android..."
	@gomobile bind -target=android -ldflags="-s -w" -o build/bin/statusgo.aar github.com/status-im/status-go/mobile
	@echo "Android cross compilation done in build/bin/statusgo.aar"

statusgo-ios: ##@cross-compile Build status-go for iOS
	@echo "Building status-go for iOS..."
	@gomobile bind -target=ios -ldflags="-s -w" -o build/bin/Statusgo.framework github.com/status-im/status-go/mobile
	@echo "iOS framework cross compilation done in build/bin/Statusgo.framework"

statusgo-linux: xgo ##@cross-compile Build status-go for Linux
	./_assets/patches/patcher -b . -p geth-xgo
	$(GOPATH)/bin/xgo --image $(XGOIMAGE) --go=$(XGO_GO) -out statusgo --dest=$(GOBIN) --targets=linux/amd64 -v -tags '$(BUILD_TAGS)' $(BUILD_FLAGS) ./cmd/statusd
	./_assets/patches/patcher -b . -p geth-xgo -r
	@echo "Linux cross compilation done."

statusgo-library: ##@cross-compile Build status-go as static library for current platform
	@echo "Building static library..."
	go build -buildmode=c-archive -o $(GOBIN)/libstatus.a $(BUILD_FLAGS) ./lib
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

install-os-dependencies:
	_assets/scripts/install_deps.sh

setup-dev: setup-build install-os-dependencies gen-install ##@other Prepare project for development

setup-build: dep-install lint-install mock-install release-install gomobile-install ##@other Prepare project for build

setup: setup-build setup-dev ##@other Prepare project for development and building

generate: ##@other Regenerate assets and other auto-generated stuff
	go generate ./static ./static/migrations
	$(shell cd ./services/shhext/chat && exec protoc --go_out=. ./*.proto)

prepare-release: clean-release
	mkdir -p $(RELEASE_DIRECTORY)
	mv build/bin/statusgo.aar $(RELEASE_DIRECTORY)/status-go-android.aar
	zip -r build/bin/Statusgo.framework.zip build/bin/Statusgo.framework
	mv build/bin/Statusgo.framework.zip $(RELEASE_DIRECTORY)/status-go-ios.zip
	zip -r $(RELEASE_DIRECTORY)/status-go-desktop.zip . -x *.git*
	${MAKE} clean

clean-release:
	rm -rf $(RELEASE_DIRECTORY)

release:
	@read -p "Are you sure you want to create a new GitHub $(shell if [ $(PRE_RELEASE) = "0" ] ; then echo release; else echo pre-release ; fi) against $(RELEASE_BRANCH) branch? (y/n): " REPLY; \
	if [ $$REPLY = "y" ]; then \
		latest_tag=$$(git describe --tags `git rev-list --tags --max-count=1`); \
		comparison="$$latest_tag..HEAD"; \
		if [ -z "$$latest_tag" ]; then comparison=""; fi; \
		changelog=$$(git log $$comparison --oneline --no-merges --format="* %h %s"); \
	    github-release $(shell if [ $(PRE_RELEASE) != "0" ] ; then echo "-prerelease" ; fi) "status-im/status-go" "$(RELEASE_TAG)" "$(RELEASE_BRANCH)" "$(changelog)" "$(RELEASE_DIRECTORY)/*" ; \
	else \
	    echo "Aborting." && exit 1; \
	fi

gomobile-install:
	go get -u golang.org/x/mobile/cmd/gomobile
ifdef NDK_GOMOBILE
	gomobile init -ndk $(NDK_GOMOBILE)
else
	gomobile init
endif

release-install:
	go get -u github.com/c4milo/github-release

gen-install:
	go get -u github.com/jteeuwen/go-bindata
	go get -u github.com/jteeuwen/go-bindata/go-bindata
	go get -u github.com/golang/protobuf/protoc-gen-go

mock-install: ##@other Install mocking tools
	go get -u github.com/golang/mock/mockgen
	dep ensure -update github.com/golang/mock

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
	go test -v -failfast $(UNIT_TEST_PACKAGES) $(gotest_extraflags)

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

canary-test: node-canary
	# TODO: uncomment that!
	#_assets/scripts/canary_test_mailservers.sh ./config/cli/fleet-eth.beta.json

lint-install:
	@# The following installs a specific version of golangci-lint, which is appropriate for a CI server to avoid different results from build to build
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b $(GOPATH)/bin v1.10.2

lint:
	@echo "lint"
	@golangci-lint run ./...

ci: lint mock dep-ensure canary-test test-unit test-e2e ##@tests Run all linters and tests at once

ci-race: lint mock dep-ensure canary-test test-unit test-e2e-race ##@tests Run all linters and tests at once + race

clean: ##@other Cleanup
	rm -fr build/bin/*

deep-clean: clean
	rm -Rdf .ethereumtest/StatusChain

dep-ensure: ##@dependencies Ensure all dependencies are in place with dep
	@dep ensure

dep-install: ##@dependencies Install vendoring tool
	go get -u github.com/golang/dep/cmd/dep

update-fleet-config: ##@other Update fleets configuration from fleets.status.im
	./_assets/ci/update-fleet-config.sh
	@echo "Updating static assets..."
	@go generate ./static
	@echo "Done"

run-mailserver: ##@Easy way to run a mailserver locally with Docker
	cd _assets/compose/mailserver/ && $(MAKE)
