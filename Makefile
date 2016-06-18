GOBIN = build/bin
GO ?= latest

statusgo:
	build/env.sh go build -i -o $(GOBIN)/statusgo ./src
	@echo "status go compilation done."
	@echo "Run \"build/bin/statusgo\" to view available commands"

statusgo-android: xgo
	build/env.sh $(GOBIN)/xgo --go=$(GO) --dest=$(GOBIN) --targets=android-16/aar ./src
	@echo "Android cross compilation done:"

xgo:
	build/env.sh go get github.com/karalabe/xgo
