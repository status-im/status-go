test: constants.go
	go test -v

constants.go: src/iso4217-table.xml src/update.go
	go run src/update.go < src/iso4217-table.xml | gofmt > $@
