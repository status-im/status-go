GO111MODULE = on

ENABLE_METRICS ?= true
BUILD_FLAGS ?= $(shell echo "-ldflags '\
	-X github.com/status-im/status-go/eth-node/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=$(ENABLE_METRICS)'")

test:
	go test ./...
.PHONY: test

lint:
	golangci-lint run -v
.PHONY: lint

install-linter:
	# install linter
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.21.0
.PHONY: install-linter
