
.PHONY:
build-spec-tests:
	go run github.com/ferranbt/fastssz/sszgen --path ./spectests/structs.go --exclude-objs Hash,Uint256
	go run github.com/ferranbt/fastssz/sszgen --path ./tests

.PHONY:
get-spec-tests:
	./scripts/download-spec-tests.sh v1.4.0-beta.5

.PHONY:
generate-testcases:
	go generate ./...

.PHONY:
benchmark:
	go test -v ./spectests/... -run=XXX -bench=.
