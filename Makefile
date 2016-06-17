GO ?= latest

statusgo:
	go install
	@echo "Done installing status go."
	@echo "Run \"statusgo\" to view available commands"

statusgo-android: xgo
	xgo --go=$(GO) --targets=android-16/aar ./
	@echo "Android cross compilation done:"

xgo:
	go get github.com/karalabe/xgo
