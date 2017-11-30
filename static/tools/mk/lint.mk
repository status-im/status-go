LINT_EXCLUDE := --exclude='.*_mock.go' --exclude='geth/jail/doc.go'
LINT_FOLDERS := extkeys cmd/... geth/... e2e/...
LINT_FOLDERS_WITHOUT_TESTS := extkeys cmd/... geth/...

lint-install:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

lint: lint-vet lint-gofmt lint-deadcode lint-misspell lint-unparam lint-unused lint-gocyclo lint-errcheck lint-ineffassign lint-interfacer lint-unconvert lint-staticcheck lint-goconst lint-gas lint-varcheck lint-structcheck lint-gosimple

lint-vet:
	@echo "lint-vet"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=vet --deadline=45s  $(LINT_FOLDERS)
lint-golint:
	@echo "lint-golint"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=golint --deadline=45s  $(LINT_FOLDERS)
lint-gofmt:
	@echo "lint-gofmt"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=gofmt --deadline=45s  $(LINT_FOLDERS)
lint-deadcode:
	@echo "lint-deadcode"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=deadcode --deadline=45s  $(LINT_FOLDERS)
lint-misspell:
	@echo "lint-misspell"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=misspell --deadline=45s  $(LINT_FOLDERS)
lint-unparam:
	@echo "lint-unparam"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=unparam --deadline=45s  $(LINT_FOLDERS)
lint-unused:
	@echo "lint-unused"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=unused --deadline=45s  $(LINT_FOLDERS)
lint-gocyclo:
	@echo "lint-gocyclo"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=gocyclo --cyclo-over=16 --deadline=45s  $(LINT_FOLDERS)
lint-errcheck:
	@echo "lint-errcheck"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=errcheck --deadline=1m  $(LINT_FOLDERS)
lint-ineffassign:
	@echo "lint-ineffassign"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=ineffassign --deadline=45s  $(LINT_FOLDERS)
lint-interfacer:
	@echo "lint-interfacer"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=interfacer --deadline=45s  $(LINT_FOLDERS)
lint-unconvert:
	@echo "lint-unconvert"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=unconvert --deadline=45s  $(LINT_FOLDERS)
lint-staticcheck:
	@echo "lint-staticcheck"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=staticcheck --deadline=45s  $(LINT_FOLDERS)
lint-goconst:
	@echo "lint-goconst"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=goconst --deadline=45s  $(LINT_FOLDERS)
lint-gas:
	@echo "lint-gas"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=gas --deadline=45s  $(LINT_FOLDERS)
lint-varcheck:
	@echo "lint-varcheck"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=varcheck --deadline=45s  $(LINT_FOLDERS)
lint-structcheck:
	@echo "lint-structcheck"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=structcheck --deadline=45s  $(LINT_FOLDERS)
lint-gosimple:
	@echo "lint-gosimple"
	@gometalinter $(LINT_EXCLUDE) --disable-all --enable=gosimple --deadline=45s  $(LINT_FOLDERS)
