package stickers

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var stickerTypeByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0x0577215622f43a39f4bc9640806dfea9b10d2a36"), // mainnet
	3: common.HexToAddress("0x8cc272396be7583c65bee82cd7b743c69a87287d"), // ropsten
}

var stickerMarketByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0x12824271339304d3a9f7e096e62a2a7e73b4a7e7"), // mainnet
	3: common.HexToAddress("0x6CC7274aF9cE9572d22DFD8545Fb8c9C9Bcb48AD"), // ropsten
}

var stickerPackByChainID = map[uint64]common.Address{
	1: common.HexToAddress("0x110101156e8F0743948B2A61aFcf3994A8Fb172e"), // mainnet
	3: common.HexToAddress("0xf852198d0385c4b871e0b91804ecd47c6ba97351"), // ropsten
}

func StickerTypeContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := stickerTypeByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}

func StickerMarketContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := stickerMarketByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}

func StickerPackContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := stickerPackByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
