.PHONY: statusgo all test xgo clean
.PHONY: statusgo-android statusgo-ios

GOBIN = build/bin
GO ?= latest

statusgo:
	build/env.sh go build -i -o $(GOBIN)/statusgo ./src
	@echo "status go compilation done."
	@echo "Run \"build/bin/statusgo\" to view available commands"

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/statusgo-*

statusgo-android: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/flags.sh) ./src
	@echo "Android cross compilation done:"

statusgo-ios: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-7.0/framework -v $(shell build/flags.sh) ./src
	@echo "iOS framework cross compilation done:"

xgo:
	build/env.sh go get github.com/karalabe/xgo

test:
	build/env.sh go test ./...

clean:
	rm -fr $(GOBIN)/*
