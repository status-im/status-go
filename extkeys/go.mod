module github.com/status-im/status-go/extkeys

go 1.13

replace github.com/ethereum/go-ethereum v1.9.5 => github.com/status-im/go-ethereum v1.9.5-status.7

require (
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/ethereum/go-ethereum v1.9.25
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/text v0.3.3
)
