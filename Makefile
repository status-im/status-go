.PHONY: statusgo all test xgo clean
.PHONY: statusgo-android statusgo-ios

GOBIN = build/bin
GO ?= latest

statusgo:
	build/env.sh go build -i -o $(GOBIN)/statusd -v $(shell build/testnet-flags.sh) ./cmd/statusd
	@echo "\nCompilation done.\nRun \"build/bin/statusd help\" to view available commands."

statusgo-cross: statusgo-android statusgo-ios
	@echo "Full cross compilation done."
	@ls -ld $(GOBIN)/statusgo-*

statusgo-android: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=android-16/aar -v $(shell build/testnet-flags.sh) ./cmd/statusd
	@echo "Android cross compilation done."

statusgo-ios: xgo
	build/env.sh $(GOBIN)/xgo --image farazdagi/xgo --go=$(GO) -out statusgo --dest=$(GOBIN) --targets=ios-9.3/framework -v $(shell build/testnet-flags.sh) ./cmd/statusd
	@echo "iOS framework cross compilation done."

statusgo-ios-simulator: xgo
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

ci: mock
	build/env.sh go test -timeout 40m -v ./geth/api/...
	build/env.sh go test -timeout 40m -v ./geth/common
	build/env.sh go test -timeout 40m -v ./geth/jail
	build/env.sh go test -timeout 40m -v ./geth/node
	build/env.sh go test -timeout 40m -v ./geth/params
	build/env.sh go test -timeout 40m -v ./extkeys
	build/env.sh go test -timeout 1m -v ./helpers/...

generate:
	cp ./node_modules/web3/dist/web3.js ./static/scripts/web3.js
	build/env.sh go generate ./static
	rm ./static/scripts/web3.js

lint-deps:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

lint-cur:
	gometalinter --disable-all --enable=deadcode extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"

lint:
	@echo "Linter: go vet\n--------------------"
	@gometalinter --disable-all --enable=vet extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: go vet --shadow\n--------------------"
	@gometalinter --disable-all --enable=vetshadow extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: gofmt\n--------------------"
	@gometalinter --disable-all --enable=gofmt extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: goimports\n--------------------"
	@gometalinter --disable-all --enable=goimports extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: golint\n--------------------"
	@gometalinter --disable-all --enable=golint extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: deadcode\n--------------------"
	@gometalinter --disable-all --enable=deadcode extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: misspell\n--------------------"
	@gometalinter --disable-all --enable=misspell extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: unparam\n--------------------"
	@gometalinter --disable-all --deadline 45s --enable=unparam extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: unused\n--------------------"
	@gometalinter --disable-all --deadline 45s --enable=unused extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: gocyclo\n--------------------"
	@gometalinter --disable-all --enable=gocyclo extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: errcheck\n--------------------"
	@gometalinter --disable-all --enable=errcheck extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: dupl\n--------------------"
	@gometalinter --disable-all --enable=dupl extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: ineffassign\n--------------------"
	@gometalinter --disable-all --enable=ineffassign extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: interfacer\n--------------------"
	@gometalinter --disable-all --enable=interfacer extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: unconvert\n--------------------"
	@gometalinter --disable-all --enable=unconvert extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: goconst\n--------------------"
	@gometalinter --disable-all --enable=goconst extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: staticcheck\n--------------------"
	@gometalinter --disable-all --deadline 45s --enable=staticcheck extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: gas\n--------------------"
	@gometalinter --disable-all --enable=gas extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: varcheck\n--------------------"
	@gometalinter --disable-all --deadline 60s --enable=varcheck extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: structcheck\n--------------------"
	@gometalinter --disable-all --enable=structcheck extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"
	@echo "Linter: gosimple\n--------------------"
	@gometalinter --disable-all --deadline 45s --enable=gosimple extkeys cmd/... geth/... | grep -v -f ./static/config/linter_exclude_list.txt || echo "OK!"

mock-install:
	go get -u github.com/golang/mock/mockgen

mock: mock-install
	mockgen -source=geth/common/types.go -destination=geth/common/types_mock.go -package=common

test:
	@build/env.sh echo "mode: set" > coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./geth/api
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./geth/common
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./geth/jail
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./geth/node
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./geth/params
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./extkeys
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	build/env.sh go test -coverprofile=coverage.out -covermode=set ./cmd/statusd
	@build/env.sh tail -n +2 coverage.out >> coverage-all.out
	@build/env.sh go tool cover -html=coverage-all.out -o coverage.html
	@build/env.sh go tool cover -func=coverage-all.out

test-api:
	build/env.sh go test -v -coverprofile=coverage.out  -coverpkg=./geth/node ./geth/api
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

test-common:
	build/env.sh go test -v -coverprofile=coverage.out ./geth/common
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

test-jail:
	build/env.sh go test -v -coverprofile=coverage.out ./geth/jail
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

test-node:
	build/env.sh go test -v -coverprofile=coverage.out ./geth/node
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

test-params:
	build/env.sh go test -v -coverprofile=coverage.out ./geth/params
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

test-extkeys:
	build/env.sh go test -v -coverprofile=coverage.out ./extkeys
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

test-cmd:
	build/env.sh go test -v -coverprofile=coverage.out ./cmd/statusd
	@build/env.sh go tool cover -html=coverage.out -o coverage.html
	@build/env.sh go tool cover -func=coverage.out

clean:
	rm -fr build/bin/*
	rm coverage.out coverage-all.out coverage.html
