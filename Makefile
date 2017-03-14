.PHONY: statusgo all test xgo clean
.PHONY: statusgo-android statusgo-ios

GOBIN = build/bin
GO ?= latest

statusgo:
	build/env.sh go build -i -o $(GOBIN)/statusgo -v $(shell build/testnet-flags.sh) ./cmd/status
	@echo "status go compilation done."
	@echo "Run \"build/bin/statusgo\" to view available commands"

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld $(GOBIN)/statusgo-*

statusgo-android: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/testnet-flags.sh) ./cmd/status
	@echo "Android cross compilation done."

statusgo-ios: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/testnet-flags.sh) ./cmd/status
	@echo "iOS framework cross compilation done."

statusgo-ios-simulator: xgo
	@build/env.sh docker pull farazdagi/xgo-ios-simulator
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo-ios-simulator --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/testnet-flags.sh) ./cmd/status
	@echo "iOS framework cross compilation done."

xgo:
	build/env.sh docker pull farazdagi/xgo
	build/env.sh go get github.com/karalabe/xgo

statusgo-mainnet:
	build/env.sh go build -i -o $(GOBIN)/statusgo -v $(shell build/mainnet-flags.sh) ./cmd/status
	@echo "status go compilation done (mainnet)."
	@echo "Run \"build/bin/statusgo\" to view available commands"

statusgo-android-mainnet: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/mainnet-flags.sh) ./cmd/status
	@echo "Android cross compilation done (mainnet)."

statusgo-ios-mainnet: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/mainnet-flags.sh) ./cmd/status
	@echo "iOS framework cross compilation done (mainnet)."

statusgo-ios-simulator-mainnet: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo-ios-simulator --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/mainnet-flags.sh) ./cmd/status
	@echo "iOS framework cross compilation done (mainnet)."

ci:
	build/env.sh go test -v -cover ./geth
	build/env.sh go test -v -cover ./extkeys

test-all:
	@build/env.sh echo "mode: set" > coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./geth/...
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./extkeys
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./cmd/status
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	@build/env.sh go tool cover -html=coverage-all.out -o coverage.html
	@build/env.sh go tool cover -func=coverage-all.out

test: test-all

test-geth:
	build/env.sh go test -v -coverprofile=coverage.out ./geth/...
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

test-extkeys:
	build/env.sh go test -v -coverprofile=coverage.out ./extkeys
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

test-cmd:
	build/env.sh go test -v -coverprofile=coverage.out ./cmd/status
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

clean:
	rm -fr build/bin/*
	rm coverage.out coverage-all.out coverage.html
