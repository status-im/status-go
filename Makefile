.PHONY: statusgo all test xgo clean help
.PHONY: statusgo-android statusgo-ios

help: ##@other Show this help
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

include ./static/tools/mk/lint.mk

ifndef GOPATH
$(error GOPATH not set. Please set GOPATH and make sure status-go is located at $$GOPATH/src/github.com/status-im/status-go. For more information about the GOPATH environment variable, see https://golang.org/doc/code.html#GOPATH)
endif

CGO_CFLAGS=-I/$(JAVA_HOME)/include -I/$(JAVA_HOME)/include/darwin
GOBIN = build/bin
GO ?= latest

# This is a code for automatic help generator.
# It supports ANSI colors and categories.
# To add new item into help output, simply add comments
# starting with '##'. To add category, use @category.
GREEN  := $(shell tput -Txterm setaf 2)
WHITE  := $(shell tput -Txterm setaf 7)
YELLOW := $(shell tput -Txterm setaf 3)
RESET  := $(shell tput -Txterm sgr0)
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

# Main targets

UNIT_TEST_PACKAGES := $(shell go list ./...  | grep -v /vendor | grep -v /e2e | grep -v /cmd | grep -v /lib)

statusgo: ##@build Build status-go as statusd server
	go build -i -o $(GOBIN)/statusd -v $(shell build/testnet-flags.sh) ./cmd/statusd
	@echo "\nCompilation done.\nRun \"build/bin/statusd -h\" to view available commands."

wnode-status: ##@build Build wnode-status (Whisper 5 debug tool)
	go build -i -o $(GOBIN)/wnode-status -v $(shell build/testnet-flags.sh) ./cmd/wnode-status

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld $(GOBIN)/statusgo-*

statusgo-android: xgo ##@cross-compile Build status-go for Android
	$(GOPATH)/bin/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/testnet-flags.sh) ./lib
	@echo "Android cross compilation done."

statusgo-ios: xgo	##@cross-compile Build status-go for iOS
	$(GOPATH)/bin/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/testnet-flags.sh) ./lib
	@echo "iOS framework cross compilation done."

statusgo-ios-simulator: xgo	##@cross-compile Build status-go for iOS Simulator
	@docker pull farazdagi/xgo-ios-simulator
	$(GOPATH)/bin/xgo --image farazdagi/xgo-ios-simulator --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/testnet-flags.sh) ./lib
	@echo "iOS framework cross compilation done."

statusgo-library: ##@cross-compile Build status-go as static library for current platform
	@echo "Building static library..."
	go build -buildmode=c-archive -o $(GOBIN)/libstatus.a ./lib
	@echo "Static library built:"
	@ls -la $(GOBIN)/libstatus.*

xgo:
	docker pull farazdagi/xgo
	go get github.com/karalabe/xgo

statusgo-mainnet:
	go build -i -o $(GOBIN)/statusgo -v $(shell build/mainnet-flags.sh) ./cmd/statusd
	@echo "status go compilation done (mainnet)."
	@echo "Run \"build/bin/statusgo\" to view available commands"

statusgo-android-mainnet: xgo
	$(GOPATH)/bin/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/mainnet-flags.sh) ./lib
	@echo "Android cross compilation done (mainnet)."

statusgo-ios-mainnet: xgo
	$(GOPATH)/bin/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/mainnet-flags.sh) ./lib
	@echo "iOS framework cross compilation done (mainnet)."

statusgo-ios-simulator-mainnet: xgo
	$(GOPATH)/bin/xgo --image farazdagi/xgo-ios-simulator --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/mainnet-flags.sh) ./lib
	@echo "iOS framework cross compilation done (mainnet)."

generate: ##@other Regenerate assets and other auto-generated stuff
	cp ./node_modules/web3/dist/web3.js ./static/scripts/web3.js
	go generate ./static
	rm ./static/scripts/web3.js

mock-install: ##@other Install mocking tools
	go get -u github.com/golang/mock/mockgen

mock: ##@other Regenerate mocks
	mockgen -source=geth/common/types.go -destination=geth/common/types_mock.go -package=common
	mockgen -source=geth/common/notification.go -destination=geth/common/notification_mock.go -package=common -imports fcm=github.com/NaySoftware/go-fcm
	mockgen -source=geth/notification/fcm/client.go -destination=geth/notification/fcm/client_mock.go -package=fcm -imports fcm=github.com/NaySoftware/go-fcm

test: test-unit-coverage ##@tests Run basic, short tests during development

test-unit: ##@tests Run unit and integration tests
	go test $(UNIT_TEST_PACKAGES)

test-unit-coverage: ##@tests Run unit and integration tests with coverage
	go test -coverpkg= $(UNIT_TEST_PACKAGES)

test-e2e: ##@tests Run e2e tests
	# order: reliability then alphabetical
	# TODO(tiabc): make a single command out of them adding `-p 1` flag.
	go test -timeout 5m ./e2e/accounts/... -network=$(networkid)
	go test -timeout 5m ./e2e/api/... -network=$(networkid)
	go test -timeout 5m ./e2e/node/... -network=$(networkid)
	go test -timeout 50m ./e2e/jail/... -network=$(networkid)
	go test -timeout 20m ./e2e/rpc/... -network=$(networkid)
	go test -timeout 20m ./e2e/whisper/... -network=$(networkid)
	go test -timeout 10m ./e2e/transactions/... -network=$(networkid)
	# e2e_test tag is required to include some files from ./lib without _test suffix
	go test -timeout 40m -tags e2e_test ./lib -network=$(networkid)

ci: lint mock-install mock test-unit test-e2e ##@tests Run all linters and tests at once

clean: ##@other Cleanup
	rm -fr build/bin/*
	rm -f coverage.out coverage-all.out coverage.html

deep-clean: clean
	rm -Rdf .ethereumtest/StatusChain
