module github.com/status-im/status-go/whisper

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.6

require (
	github.com/deckarep/golang-set v0.0.0-20180603214616-504e848d77ea
	github.com/ethereum/go-ethereum v1.9.5
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/syndtr/goleveldb v0.0.0-20181128100959-b001fa50d6b2
	github.com/tsenart/tb v0.0.0-20181025101425-0d2499c8b6e9
	golang.org/x/crypto v0.0.0-20191029031824-8986dd9e96cf
	golang.org/x/sync v0.0.0-20180314180146-1d60e4601c6f
)
