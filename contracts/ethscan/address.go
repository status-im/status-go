package ethscan

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	wallet_common "github.com/status-im/status-go/services/wallet/common"
)

var errorNotAvailableOnChainID = errors.New("not available for chainID")

type ContractData struct {
	Address        common.Address
	CreatedAtBlock uint
}

var contractDataByChainID = map[uint64]ContractData{
	wallet_common.EthereumMainnet:      {common.HexToAddress("0x08A8fDBddc160A7d5b957256b903dCAb1aE512C5"), 12_194_222},
	wallet_common.OptimismMainnet:      {common.HexToAddress("0x9e5076df494fc949abc4461f4e57592b81517d81"), 34_421_097},
	wallet_common.ArbitrumMainnet:      {common.HexToAddress("0xbb85398092b83a016935a17fc857507b7851a071"), 70_031_945},
	wallet_common.BaseMainnet:          {common.HexToAddress("0xc68c1e011cfE059EB94C8915c291502288704D89"), 24_567_587},
	wallet_common.BSCMainnet:           {common.HexToAddress("0x71cfeb2ab5a3505f80b4c86f8ccd0a4b29f62447"), 47_746_468},
	wallet_common.EthereumSepolia:      {common.HexToAddress("0xec21ebe1918e8975fc0cd0c7747d318c00c0acd5"), 4_366_506},
	wallet_common.ArbitrumSepolia:      {common.HexToAddress("0xec21Ebe1918E8975FC0CD0c7747D318C00C0aCd5"), 553_947},
	wallet_common.OptimismSepolia:      {common.HexToAddress("0xec21ebe1918e8975fc0cd0c7747d318c00c0acd5"), 7_362_011},
	wallet_common.BaseSepolia:          {common.HexToAddress("0xc68c1e011cfE059EB94C8915c291502288704D89"), 20_078_235},
	wallet_common.StatusNetworkSepolia: {common.HexToAddress("0xc68c1e011cfE059EB94C8915c291502288704D89"), 1_753_813},
	wallet_common.BSCTestnet:           {common.HexToAddress("0x71cfeb2ab5a3505f80b4c86f8ccd0a4b29f62447"), 49_365_870},
	wallet_common.TestnetChainID:       {common.HexToAddress("0x0000000000000000000000000000000000777333"), 50}, // unit tests
}

func ContractAddress(chainID uint64) (common.Address, error) {
	contract, exists := contractDataByChainID[chainID]
	if !exists {
		return *new(common.Address), errorNotAvailableOnChainID
	}
	return contract.Address, nil
}

func ContractCreatedAt(chainID uint64) (uint, error) {
	contract, exists := contractDataByChainID[chainID]
	if !exists {
		return 0, errorNotAvailableOnChainID
	}
	return contract.CreatedAtBlock, nil
}
