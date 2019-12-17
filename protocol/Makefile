GO111MODULE = on

ENABLE_METRICS ?= true
BUILD_FLAGS ?= $(shell echo "-ldflags '\
	-X github.com/status-im/status-go/protocol/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=$(ENABLE_METRICS)'")

test:
	go test ./...
.PHONY: test

test-race:
	go test -race ./...
.PHONY: test-race

lint:
	golangci-lint run -v
.PHONY: lint

install-linter:
	# install linter
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.21.0
.PHONY: install-linter

install-dev:
	# a tool to vendor non-go files
	go get github.com/goware/modvendor@latest

	go get github.com/golang/mock/gomock@latest
	go install github.com/golang/mock/mockgen

	go get github.com/kevinburke/go-bindata/go-bindata@v3.13.0
	go get github.com/golang/protobuf/protoc-gen-go@v1.3.1
.PHONY: install-dev

generate:
	go generate ./...
.PHONY: generate

new-migration:
	@if [ -z "$$DIR" ]; then \
		echo 'missing DIR var'; \
		exit 1; \
	fi

	@if [ -z "$$NAME" ]; then \
		echo 'missing NAME var'; \
		exit 1; \
	fi

	mkdir -p $(DIR)
	touch $(DIR)/`date +"%s"`_$(NAME).down.sql ./$(DIR)/`date +"%s"`_$(NAME).up.sql
.PHONY: create-migration
