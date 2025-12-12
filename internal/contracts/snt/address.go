package snt

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

var contractAddressByChainID = map[uint64]common.Address{
	1:          common.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"), // mainnet
	10:         common.HexToAddress("0x650af3c15af43dcb218406d30784416d64cfb6b2"), // optimism
	8453:       common.HexToAddress("0x662015ec830df08c0fc45896fab726542e8ac09e"), // base
	42161:      common.HexToAddress("0x707f635951193ddafbb40971a0fcaab8a6415160"), // arbitrum
	11155111:   common.HexToAddress("0xE452027cdEF746c7Cd3DB31CB700428b16cD8E51"), // sepolia
	84532:      common.HexToAddress("0xfdb3b57944943a7724fcc0520ee2b10659969a06"), // base testnet
	1660990954: common.HexToAddress("0x1c3ac2a186c6149ae7cb4d716ebbd0766e4f898a"), // status testnet
}

func ContractAddress(chainID uint64) (common.Address, error) {
	addr, exists := contractAddressByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return addr, nil
}
