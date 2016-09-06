.PHONY: statusgo all test xgo clean
.PHONY: statusgo-android statusgo-ios

GOBIN = build/bin
GO ?= latest

statusgo:
	build/env.sh go build -i -o $(GOBIN)/statusgo -v $(shell build/flags.sh) ./src
	@echo "status go compilation done."
	@echo "Run \"build/bin/statusgo\" to view available commands"

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/statusgo-*

statusgo-android: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/flags.sh) ./src
	@echo "Android cross compilation done:"

statusgo-ios: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/flags.sh) ./src
	@echo "iOS framework cross compilation done:"

statusgo-ios-simulator: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo-ios-simulator --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/flags.sh) ./src
	@echo "iOS framework cross compilation done:"

xgo:
	build/env.sh go get github.com/karalabe/xgo

test:
	build/env.sh go test -v -coverprofile=cover.out ./src

test-cover: test
	build/env.sh go tool cover -html=cover.out -o cover.html

clean:
	rm -fr build/bin/*
