.PHONY: statusgo all test xgo clean help
.PHONY: statusgo-android statusgo-ios

help: ##@other Show this help
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

ifndef GOPATH
	$(error GOPATH not set. Please set GOPATH and make sure status-go is located at $$GOPATH/src/github.com/status-im/status-go. \
	For more information about the GOPATH environment variable, see https://golang.org/doc/code.html#GOPATH)
endif

CGO_CFLAGS=-I/$(JAVA_HOME)/include -I/$(JAVA_HOME)/include/darwin
BUILD_TAGS =
GOBIN = build/bin
GO ?= latest
XGOVERSION ?= 1.9.2
XGOIMAGE = statusteam/xgo:$(XGOVERSION)
XGOIMAGEIOSSIM = statusteam/xgo-ios-simulator:$(XGOVERSION)

DOCKER_IMAGE_NAME ?= status-go

UNIT_TEST_PACKAGES := $(shell go list ./...  | grep -v /vendor | grep -v /t/e2e | grep -v /cmd | grep -v /lib)

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
		   while(<>) { push @{$$help{$$2 // 'options'}}, [$$1, $$3] if /^([a-zA-Z\-]+)\s*:.*\#\#(?:@([a-zA-Z\-]+))?\s(.*)$$/ }; \
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
	go build -i -o $(GOBIN)/statusd -v -tags '$(BUILD_TAGS)' $(shell _assets/build/testnet-flags.sh) ./cmd/statusd
	@echo "Compilation done."
	@echo "Run \"build/bin/statusd -h\" to view available commands."

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld $(GOBIN)/statusgo-*

statusgo-android: xgo ##@cross-compile Build status-go for Android
	$(GOPATH)/bin/xgo --image $(XGOIMAGE) --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v -tags '$(BUILD_TAGS)' $(shell _assets/build/testnet-flags.sh) ./lib
	@echo "Android cross compilation done."

statusgo-ios: xgo	##@cross-compile Build status-go for iOS
	$(GOPATH)/bin/xgo --image $(XGOIMAGE) --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v -tags '$(BUILD_TAGS)' $(shell _assets/build/testnet-flags.sh) ./lib
	@echo "iOS framework cross compilation done."

statusgo-ios-simulator: xgo	##@cross-compile Build status-go for iOS Simulator
	@docker pull $(XGOIMAGEIOSSIM)
	$(GOPATH)/bin/xgo --image $(XGOIMAGEIOSSIM) --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v -tags '$(BUILD_TAGS)' $(shell _assets/build/testnet-flags.sh) ./lib
	@echo "iOS framework cross compilation done."

statusgo-library: ##@cross-compile Build status-go as static library for current platform
	@echo "Building static library..."
	go build -buildmode=c-archive -o $(GOBIN)/libstatus.a ./lib
	@echo "Static library built:"
	@ls -la $(GOBIN)/libstatus.*

docker-image: ##@docker Build docker image (use DOCKER_IMAGE_NAME to set the image name)
	@echo "Building docker image..."
	docker build --file _assets/build/Dockerfile --build-arg "build_tags=$(BUILD_TAGS)" . -t $(DOCKER_IMAGE_NAME):latest

docker-image-tag: ##@docker Tag DOCKER_IMAGE_NAME:latest with a tag following pattern $GIT_SHA[:8]-$BUILD_TAGS
	@echo "Tagging docker image..."
	docker tag $(DOCKER_IMAGE_NAME):latest $(DOCKER_IMAGE_NAME):$(shell BUILD_TAGS="$(BUILD_TAGS)" ./_assets/ci/get-docker-image-tag.sh)

xgo-docker-images: ##@docker Build xgo docker images
	@echo "Building xgo docker images..."
	docker build _assets/build/xgo/base -t $(XGOIMAGE)
	docker build _assets/build/xgo/ios-simulator -t $(XGOIMAGEIOSSIM)

xgo:
	docker pull $(XGOIMAGE)
	go get github.com/karalabe/xgo

statusgo-mainnet:
	go build -i -o $(GOBIN)/statusgo -v $(shell _assets/build/mainnet-flags.sh) ./cmd/statusd
	@echo "status go compilation done (mainnet)."
	@echo "Run \"build/bin/statusgo\" to view available commands"

statusgo-android-mainnet: xgo
	$(GOPATH)/bin/xgo --image $(XGOIMAGE) --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell _assets/build/mainnet-flags.sh) ./lib
	@echo "Android cross compilation done (mainnet)."

statusgo-ios-mainnet: xgo
	$(GOPATH)/bin/xgo --image $(XGOIMAGE) --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell _assets/build/mainnet-flags.sh) ./lib
	@echo "iOS framework cross compilation done (mainnet)."

statusgo-ios-simulator-mainnet: xgo
	$(GOPATH)/bin/xgo --image $(XGOIMAGEIOSSIM) --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell _assets/build/mainnet-flags.sh) ./lib
	@echo "iOS framework cross compilation done (mainnet)."

generate: ##@other Regenerate assets and other auto-generated stuff
	cp ./node_modules/web3/dist/web3.js ./static/scripts/web3.js
	go generate ./static
	rm ./static/scripts/web3.js

mock-install: ##@other Install mocking tools
	go get -u github.com/golang/mock/mockgen

mock: ##@other Regenerate mocks
	mockgen -source=geth/common/types.go -destination=geth/common/types_mock.go -package=common
	mockgen -source=geth/mailservice/mailservice.go -destination=geth/mailservice/mailservice_mock.go -package=mailservice
	mockgen -source=geth/common/notification.go -destination=geth/common/notification_mock.go -package=common -imports fcm=github.com/NaySoftware/go-fcm
	mockgen -source=geth/notification/fcm/client.go -destination=geth/notification/fcm/client_mock.go -package=fcm -imports fcm=github.com/NaySoftware/go-fcm
	mockgen -source=geth/transactions/fake/txservice.go -destination=geth/transactions/fake/mock.go -package=fake

test: test-unit-coverage ##@tests Run basic, short tests during development

test-unit: ##@tests Run unit and integration tests
	go test $(UNIT_TEST_PACKAGES)

test-unit-coverage: ##@tests Run unit and integration tests with coverage
	go test -coverpkg= $(UNIT_TEST_PACKAGES)

test-e2e: ##@tests Run e2e tests
	# order: reliability then alphabetical
	# TODO(tiabc): make a single command out of them adding `-p 1` flag.
	go test -timeout 5m ./t/e2e/accounts/... -network=$(networkid)
	go test -timeout 5m ./t/e2e/api/... -network=$(networkid)
	go test -timeout 5m ./t/e2e/node/... -network=$(networkid)
	go test -timeout 50m ./t/e2e/jail/... -network=$(networkid)
	go test -timeout 20m ./t/e2e/rpc/... -network=$(networkid)
	go test -timeout 20m ./t/e2e/whisper/... -network=$(networkid)
	go test -timeout 10m ./t/e2e/transactions/... -network=$(networkid)
	# e2e_test tag is required to include some files from ./lib without _test suffix
	go test -timeout 40m -tags e2e_test ./lib -network=$(networkid)

lint-install:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

lint:
	@echo "lint"
	@gometalinter ./...

ci: lint mock test-unit test-e2e ##@tests Run all linters and tests at once

clean: ##@other Cleanup
	rm -fr build/bin/*
	rm -f coverage.out coverage-all.out coverage.html

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
ifdef GETH_BRANCH
	@# escape slashes
	@GETH_BRANCH=$(echo $GETH_BRANCH | sed 's@\/@\\\/@g')
	@# Update go-ethereum contraint branch
	@sed -i 'N;N;s@\(\[\[constraint]]\n  name = "github.com\/ethereum\/go-ethereum"\n  branch =\)\(.*\)@\1 '"\"${GETH_BRANCH}\""'@g' Gopkg.toml
endif
	@dep ensure -v -update github.com/ethereum/go-ethereum
	$(MAKE) dep-ensure
	git add Gopkg.lock Gopkg.toml vendor/ _assets/patches/
ifeq ($(git diff --cached --quiet), 0)
	@echo "No changes to commit. Geth up to date." && exit 1
endif
	@git commit --quiet -m "Updating Geth"
	@echo "Geth updated."

patch: ##@patching Revert and apply all patches
	./_assets/patches/patcher

patch-revert: ##@patching Revert all patches only
	./_assets/patches/patcher -r
