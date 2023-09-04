package common

import "time"

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

var AverageBlockDurationForChain = map[ChainID]time.Duration{
	ChainID(UnknownChainID):  time.Duration(12000) * time.Millisecond,
	ChainID(EthereumMainnet): time.Duration(12000) * time.Millisecond,
	ChainID(EthereumGoerli):  time.Duration(12000) * time.Millisecond,
	ChainID(OptimismMainnet): time.Duration(400) * time.Millisecond,
	ChainID(OptimismGoerli):  time.Duration(2000) * time.Millisecond,
	ChainID(ArbitrumMainnet): time.Duration(300) * time.Millisecond,
	ChainID(ArbitrumGoerli):  time.Duration(1500) * time.Millisecond,
}
