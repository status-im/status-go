.PHONY: regen fetch test

regen:
	go run cmd/main.go --from-json-file=exchanges.json
	gofmt -w exchanges.go

fetch:
	go run cmd/main.go
	gofmt -w exchanges.go

test:
	go test -v -cover ./...
