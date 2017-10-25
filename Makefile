.PHONY: statusgo all test xgo clean help
.PHONY: statusgo-android statusgo-ios

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

help: ##@other Show this help
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

# Main targets

UNIT_TEST_PACKAGES := $(shell go list ./...  | grep -v /vendor | grep -v /e2e | grep -v /cmd)

statusgo: ##@build Build status-go as statusd server
	build/env.sh go build -i -o $(GOBIN)/statusd -v $(shell build/testnet-flags.sh) ./cmd/statusd
	@echo "\nCompilation done.\nRun \"build/bin/statusd help\" to view available commands."

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld $(GOBIN)/statusgo-*

statusgo-android: xgo ##@cross-compile Build status-go for Android
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/testnet-flags.sh) ./cmd/statusd
	@echo "Android cross compilation done."

statusgo-ios: xgo	##@cross-compile Build status-go for iOS
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/testnet-flags.sh) ./cmd/statusd
	@echo "iOS framework cross compilation done."

statusgo-ios-simulator: xgo	##@cross-compile Build status-go for iOS Simulator
	@build/env.sh docker pull farazdagi/xgo-ios-simulator
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo-ios-simulator --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/testnet-flags.sh) ./cmd/statusd
	@echo "iOS framework cross compilation done."

xgo:
	build/env.sh docker pull farazdagi/xgo
	build/env.sh go get github.com/karalabe/xgo

statusgo-mainnet:
	build/env.sh go build -i -o $(GOBIN)/statusgo -v $(shell build/mainnet-flags.sh) ./cmd/statusd
	@echo "status go compilation done (mainnet)."
	@echo "Run \"build/bin/statusgo\" to view available commands"

statusgo-android-mainnet: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/mainnet-flags.sh) ./cmd/statusd
	@echo "Android cross compilation done (mainnet)."

statusgo-ios-mainnet: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/mainnet-flags.sh) ./cmd/statusd
	@echo "iOS framework cross compilation done (mainnet)."

statusgo-ios-simulator-mainnet: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo-ios-simulator --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/mainnet-flags.sh) ./cmd/statusd
	@echo "iOS framework cross compilation done (mainnet)."

generate: ##@other Regenerate assets and other auto-generated stuff
	cp ./node_modules/web3/dist/web3.js ./static/scripts/web3.js
	build/env.sh go generate ./static
	rm ./static/scripts/web3.js

lint-deps:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

lint-cur:
	gometalinter --disable-all --enable=deadcode $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"

lint: ##@tests Run meta linter on code
	@echo "Linter: go vet\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=vet $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: go vet --shadow\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=vetshadow $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: gofmt\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=gofmt $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: goimports\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=goimports $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: golint\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=golint $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: deadcode\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=deadcode $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: misspell\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=misspell $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: unparam\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --deadline 45s --enable=unparam $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: unused\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --deadline 45s --enable=unused $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: gocyclo\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=gocyclo --cyclo-over=20 $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: errcheck\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=errcheck $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: dupl\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --exclude='.*_test.go' --disable-all --enable=dupl --dupl-threshold=100  extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: ineffassign\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=ineffassign $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: interfacer\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=interfacer $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: unconvert\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=unconvert $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: goconst\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=goconst $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: staticcheck\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --deadline 45s --enable=staticcheck $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: gas\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=gas $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: varcheck\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --deadline 60s --enable=varcheck $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: structcheck\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=structcheck $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: gosimple\n--------------------"
	@gometalinter $(LINT_EXCLUDE) --disable-all --deadline 45s --enable=gosimple $(LINT_FOLDERS) | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"

mock-install: ##@other Install mocking tools
	go get -u github.com/golang/mock/mockgen

mock: ##@other Regenerate mocks
	mockgen -source=geth/common/types.go -destination=geth/common/types_mock.go -package=common

test: ##@tests Run unit and integration tests
	build/env.sh go test $(UNIT_TEST_PACKAGES)

test-coverage: ##@tests Run unit and integration tests with covevare
	build/env.sh go test -coverpkg= $(UNIT_TEST_PACKAGES)

test-e2e: ##@tests Run e2e tests
	# order: reliability then alphabetical
	# TODO(tiabc): make a single command out of them adding `-p 1` flag.
	build/env.sh go test -timeout 5m ./e2e/accounts/...
	build/env.sh go test -timeout 5m ./e2e/api/... -network=4
	build/env.sh go test -timeout 5m ./e2e/node/...
	build/env.sh go test -timeout 15m ./e2e/jail/...
	build/env.sh go test -timeout 20m ./e2e/rpc/... -network=4
	build/env.sh go test -timeout 20m ./e2e/whisper/...
	build/env.sh go test -timeout 10m ./e2e/transactions/...
	build/env.sh go test -timeout 40m ./cmd/statusd

ci: mock-install mock test-coverage test-e2e ##@tests Run all tests in CI

clean: ##@other Cleanup
	rm -fr build/bin/*
	rm coverage.out coverage-all.out coverage.html


LINT_EXCLUDE := --exclude='.*_mock.go' --exclude='geth/jail/doc.go'
LINT_FOLDERS := extkeys cmd/... geth/... e2e/...