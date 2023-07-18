package common

type ChainID uint64

const (
	UnknownChainID  uint64 = 0
	EthereumMainnet uint64 = 1
	EthereumGoerli  uint64 = 5
	EthereumSepolia uint64 = 11155111
	OptimismMainnet uint64 = 10
	OptimismGoerli  uint64 = 420
	ArbitrumMainnet uint64 = 42161
	ArbitrumGoerli  uint64 = 421613
)
