package communities

import (
	"context"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	_ "github.com/mutecomm/go-sqlcipher/v4" // require go-sqlcipher that overrides default implementation

	"github.com/status-im/status-go/protocol/protobuf"
)

func (s *ManagerSuite) Test_calculatePermissionedBalances() {
	var mainnetID uint64 = 1
	var arbitrumID uint64 = 42161
	var gnosisID uint64 = 100
	chainIDs := []uint64{mainnetID, arbitrumID}

	mainnetSNTContractAddress := gethcommon.HexToAddress("0xC")
	mainnetETHContractAddress := gethcommon.HexToAddress("0xA")
	arbitrumETHContractAddress := gethcommon.HexToAddress("0xB")
	mainnetTMasterAddress := gethcommon.HexToAddress("0x123")
	mainnetOwnerAddress := gethcommon.HexToAddress("0x1234")

	account1Address := gethcommon.HexToAddress("0x1")
	account2Address := gethcommon.HexToAddress("0x2")
	account3Address := gethcommon.HexToAddress("0x3")
	accountAddresses := []gethcommon.Address{account1Address, account2Address, account3Address}

	erc20Balances := make(BalancesByChain)
	erc721Balances := make(CollectiblesByChain)

	erc20Balances[mainnetID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	erc20Balances[arbitrumID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	erc20Balances[gnosisID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	erc721Balances[mainnetID] = make(map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)

	// Account 1 balances
	erc20Balances[mainnetID][account1Address] = make(map[gethcommon.Address]*hexutil.Big)
	erc20Balances[mainnetID][account1Address][mainnetETHContractAddress] = intToBig(10)
	erc20Balances[arbitrumID][account1Address] = make(map[gethcommon.Address]*hexutil.Big)
	erc20Balances[arbitrumID][account1Address][arbitrumETHContractAddress] = intToBig(25)

	// Account 2 balances
	erc20Balances[mainnetID][account2Address] = make(map[gethcommon.Address]*hexutil.Big)
	erc20Balances[mainnetID][account2Address][mainnetSNTContractAddress] = intToBig(120)
	erc721Balances[mainnetID][account2Address] = make(thirdparty.TokenBalancesPerContractAddress)
	erc721Balances[mainnetID][account2Address][mainnetTMasterAddress] = []thirdparty.TokenBalance{
		thirdparty.TokenBalance{
			TokenID: uintToDecBig(456),
			Balance: uintToDecBig(1),
		},
		thirdparty.TokenBalance{
			TokenID: uintToDecBig(123),
			Balance: uintToDecBig(2),
		},
	}
	erc721Balances[mainnetID][account2Address][mainnetOwnerAddress] = []thirdparty.TokenBalance{
		thirdparty.TokenBalance{
			TokenID: uintToDecBig(100),
			Balance: uintToDecBig(6),
		},
		thirdparty.TokenBalance{
			TokenID: uintToDecBig(101),
			Balance: uintToDecBig(1),
		},
	}

	erc20Balances[arbitrumID][account2Address] = make(map[gethcommon.Address]*hexutil.Big)
	erc20Balances[arbitrumID][account2Address][arbitrumETHContractAddress] = intToBig(2)

	// Account 3 balances. This account is used to assert zeroed balances are
	// removed from the final response.
	erc20Balances[mainnetID][account3Address] = make(map[gethcommon.Address]*hexutil.Big)
	erc20Balances[mainnetID][account3Address][mainnetETHContractAddress] = intToBig(0)

	// A balance that should be ignored because the list of wallet addresses don't
	// contain any wallet in the Gnosis chain.
	erc20Balances[gnosisID][gethcommon.HexToAddress("0xF")] = make(map[gethcommon.Address]*hexutil.Big)
	erc20Balances[gnosisID][gethcommon.HexToAddress("0xF")][gethcommon.HexToAddress("0x99")] = intToBig(5)

	tokenPermissions := []*CommunityTokenPermission{
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
				TokenCriteria: []*protobuf.TokenCriteria{
					&protobuf.TokenCriteria{
						Type:              protobuf.CommunityTokenType_ERC721,
						Symbol:            "TMTEST",
						Name:              "TMaster-Test",
						Amount:            "1",
						TokenIds:          []uint64{123, 456},
						ContractAddresses: map[uint64]string{mainnetID: mainnetTMasterAddress.Hex()},
					},
				},
			},
		},
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Type: protobuf.CommunityTokenPermission_BECOME_TOKEN_OWNER,
				TokenCriteria: []*protobuf.TokenCriteria{
					&protobuf.TokenCriteria{
						Type:   protobuf.CommunityTokenType_ERC721,
						Symbol: "OWNTEST",
						Name:   "Owner-Test",
						Amount: "5",
						// No account has a positive balance for these token IDs, so we
						// expect this collectible to not be present in the final result.
						TokenIds:          []uint64{666},
						ContractAddresses: map[uint64]string{mainnetID: mainnetOwnerAddress.Hex()},
					},
				},
			},
		},
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Type: protobuf.CommunityTokenPermission_BECOME_ADMIN,
				TokenCriteria: []*protobuf.TokenCriteria{
					&protobuf.TokenCriteria{
						Type:     protobuf.CommunityTokenType_ERC20,
						Symbol:   "ETH",
						Name:     "Ethereum",
						Amount:   "20",
						Decimals: 18,
						ContractAddresses: map[uint64]string{
							arbitrumID: arbitrumETHContractAddress.Hex(),
							mainnetID:  mainnetETHContractAddress.Hex(),
						},
					},
					&protobuf.TokenCriteria{
						Type:              protobuf.CommunityTokenType_ERC20,
						Symbol:            "ETH",
						Name:              "Ethereum",
						Amount:            "4",
						Decimals:          18,
						ContractAddresses: map[uint64]string{arbitrumID: arbitrumETHContractAddress.Hex()},
					},
				},
			},
		},
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				Type: protobuf.CommunityTokenPermission_BECOME_MEMBER,
				TokenCriteria: []*protobuf.TokenCriteria{
					&protobuf.TokenCriteria{
						Type:              protobuf.CommunityTokenType_ERC20,
						Symbol:            "SNT",
						Name:              "Status",
						Amount:            "1000",
						Decimals:          16,
						ContractAddresses: map[uint64]string{mainnetID: mainnetSNTContractAddress.Hex()},
					},
				},
			},
		},
		&CommunityTokenPermission{
			CommunityTokenPermission: &protobuf.CommunityTokenPermission{
				// Unknown permission should be ignored.
				Type: protobuf.CommunityTokenPermission_UNKNOWN_TOKEN_PERMISSION,
				TokenCriteria: []*protobuf.TokenCriteria{
					&protobuf.TokenCriteria{
						Type:              protobuf.CommunityTokenType_ERC20,
						Symbol:            "DAI",
						Name:              "Dai",
						Amount:            "7",
						Decimals:          12,
						ContractAddresses: map[uint64]string{mainnetID: "0x1234567"},
					},
				},
			},
		},
	}

	actual := calculatePermissionedBalances(
		chainIDs,
		accountAddresses,
		erc20Balances,
		erc721Balances,
		tokenPermissions,
	)

	expected := make(map[gethcommon.Address][]PermissionedBalance)
	expected[account1Address] = []PermissionedBalance{
		PermissionedBalance{
			Type:     protobuf.CommunityTokenType_ERC20,
			Symbol:   "ETH",
			Name:     "Ethereum",
			Decimals: 18,
			Amount:   &bigint.BigInt{Int: big.NewInt(35)},
		},
	}
	expected[account2Address] = []PermissionedBalance{
		PermissionedBalance{
			Type:     protobuf.CommunityTokenType_ERC20,
			Symbol:   "ETH",
			Name:     "Ethereum",
			Decimals: 18,
			Amount:   &bigint.BigInt{Int: big.NewInt(2)},
		},
		PermissionedBalance{
			Type:     protobuf.CommunityTokenType_ERC20,
			Symbol:   "SNT",
			Name:     "Status",
			Decimals: 16,
			Amount:   &bigint.BigInt{Int: big.NewInt(120)},
		},
		PermissionedBalance{
			Type:   protobuf.CommunityTokenType_ERC721,
			Symbol: "TMTEST",
			Name:   "TMaster-Test",
			Amount: &bigint.BigInt{Int: big.NewInt(1)},
		},
	}

	_, ok := actual[account1Address]
	s.Require().True(ok, "not found account1Address='%s'", account1Address)
	_, ok = actual[account1Address]
	s.Require().True(ok, "not found account2Address='%s'", account2Address)

	for accountAddress, permissionedTokens := range actual {
		s.Require().ElementsMatch(expected[accountAddress], permissionedTokens, "accountAddress='%s'", accountAddress)
	}
}

func (s *ManagerSuite) Test_GetPermissionedBalances() {
	m, collectiblesManager, tokenManager := s.setupManagerForTokenPermissions()
	s.Require().NotNil(m)
	s.Require().NotNil(collectiblesManager)

	request := &requests.CreateCommunity{
		Membership: protobuf.CommunityPermissions_AUTO_ACCEPT,
	}
	community, err := m.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	accountAddress := gethcommon.HexToAddress("0x1")
	accountAddresses := []gethcommon.Address{accountAddress}

	var chainID uint64 = 5
	erc20ETHAddress := gethcommon.HexToAddress("0xA")
	erc721Address := gethcommon.HexToAddress("0x123")

	permissionRequest := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				Symbol:            "ETH",
				Name:              "Ethereum",
				Amount:            "3",
				Decimals:          18,
				ContractAddresses: map[uint64]string{chainID: erc20ETHAddress.Hex()},
			},
		},
	}
	_, changes, err := m.CreateCommunityTokenPermission(permissionRequest)
	s.Require().NoError(err)
	s.Require().Len(changes.TokenPermissionsAdded, 1)

	permissionRequest = &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC721,
				Symbol:            "TMTEST",
				Name:              "TMaster-Test",
				Amount:            "1",
				TokenIds:          []uint64{666},
				ContractAddresses: map[uint64]string{chainID: erc721Address.Hex()},
			},
		},
	}
	_, changes, err = m.CreateCommunityTokenPermission(permissionRequest)
	s.Require().NoError(err)
	s.Require().Len(changes.TokenPermissionsAdded, 1)

	tokenManager.setResponse(chainID, accountAddress, erc20ETHAddress, 42)
	collectiblesManager.setResponse(chainID, accountAddress, erc721Address, []thirdparty.TokenBalance{
		thirdparty.TokenBalance{
			TokenID: uintToDecBig(666),
			Balance: uintToDecBig(15),
		},
	})

	actual, err := m.GetPermissionedBalances(context.Background(), community.ID(), accountAddresses)
	s.Require().NoError(err)

	expected := make(map[gethcommon.Address][]PermissionedBalance)
	expected[accountAddress] = []PermissionedBalance{
		PermissionedBalance{
			Type:     protobuf.CommunityTokenType_ERC20,
			Symbol:   "ETH",
			Name:     "Ethereum",
			Decimals: 18,
			Amount:   &bigint.BigInt{Int: big.NewInt(42)},
		},
		PermissionedBalance{
			Type:   protobuf.CommunityTokenType_ERC721,
			Symbol: "TMTEST",
			Name:   "TMaster-Test",
			Amount: &bigint.BigInt{Int: big.NewInt(1)},
		},
	}

	_, ok := actual[accountAddress]
	s.Require().True(ok, "not found accountAddress='%s'", accountAddress)

	for address, permissionedBalances := range actual {
		s.Require().ElementsMatch(expected[address], permissionedBalances)
	}
}
