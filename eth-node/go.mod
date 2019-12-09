module github.com/status-im/status-go/eth-node

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.6

replace github.com/status-im/status-go/extkeys => ../extkeys

replace github.com/status-im/status-go/whisper => ../whisper

require (
	github.com/ethereum/go-ethereum v1.9.5
	github.com/mattn/go-pointer v0.0.0-20190911064623-a0a44394634f
	github.com/status-im/doubleratchet v3.0.0+incompatible
	github.com/status-im/status-go/extkeys v1.0.0 // indirect
	github.com/status-im/status-go/whisper v1.0.0 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/wealdtech/go-ens/v3 v3.0.9
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20191119213627-4f8c1d86b1ba
)
