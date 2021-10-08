module github.com/status-im/go-multiaddr-ethv4

go 1.11

replace github.com/ethereum/go-ethereum v1.10.4 => github.com/status-im/go-ethereum v1.10.4-status.2

require (
	github.com/ethereum/go-ethereum v1.10.4
	github.com/libp2p/go-libp2p-core v0.9.0
	github.com/multiformats/go-multiaddr v0.4.0
	github.com/multiformats/go-multihash v0.0.14
	github.com/stretchr/testify v1.7.0
)
