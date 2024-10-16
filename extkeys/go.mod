module github.com/status-im/status-go/extkeys

go 1.21

toolchain go1.21.8

replace github.com/ethereum/go-ethereum v1.10.26 => github.com/status-im/go-ethereum v1.10.25-status.14

require (
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/ethereum/go-ethereum v1.10.26
	golang.org/x/crypto v0.23.0
	golang.org/x/text v0.15.0
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
)
