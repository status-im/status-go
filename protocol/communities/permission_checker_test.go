package communities

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestPermissionCheckerSuite(t *testing.T) {
	suite.Run(t, new(PermissionCheckerSuite))
}

type PermissionCheckerSuite struct {
	suite.Suite
}

func (s *PermissionCheckerSuite) TestMergeValidCombinations() {

	permissionChecker := DefaultPermissionChecker{}

	combination1 := &AccountChainIDsCombination{
		Address:  gethcommon.HexToAddress("0xA"),
		ChainIDs: []uint64{1},
	}

	combination2 := &AccountChainIDsCombination{
		Address:  gethcommon.HexToAddress("0xB"),
		ChainIDs: []uint64{5},
	}

	combination3 := &AccountChainIDsCombination{
		Address:  gethcommon.HexToAddress("0xA"),
		ChainIDs: []uint64{5},
	}

	combination4 := &AccountChainIDsCombination{
		Address:  gethcommon.HexToAddress("0xB"),
		ChainIDs: []uint64{5},
	}

	mergedCombination := permissionChecker.MergeValidCombinations([]*AccountChainIDsCombination{combination1, combination2},
		[]*AccountChainIDsCombination{combination3, combination4})

	s.Require().Len(mergedCombination, 2)
	chains1 := mergedCombination[0].ChainIDs
	chains2 := mergedCombination[1].ChainIDs

	if len(chains1) == 2 {
		s.Equal([]uint64{1, 5}, chains1)
		s.Equal([]uint64{5}, chains2)
	} else {
		s.Equal([]uint64{1, 5}, chains2)
		s.Equal([]uint64{5}, chains1)
	}

}

func (s *PermissionCheckerSuite) TestCheckPermissions() {
	testCases := []struct {
		name                string
		amountInWei         func(t protobuf.CommunityTokenType) string
		requiredAmountInWei func(t protobuf.CommunityTokenType) string
		shouldSatisfy       bool
	}{
		{
			name: "account does not meet criteria",
			amountInWei: func(t protobuf.CommunityTokenType) string {
				if t == protobuf.CommunityTokenType_ERC721 {
					return "1"
				}
				return "1000000000000000000"
			},
			requiredAmountInWei: func(t protobuf.CommunityTokenType) string {
				if t == protobuf.CommunityTokenType_ERC721 {
					return "2"
				}
				return "2000000000000000000"
			},
			shouldSatisfy: false,
		},
		{
			name: "account does exactly meet criteria",
			amountInWei: func(t protobuf.CommunityTokenType) string {
				if t == protobuf.CommunityTokenType_ERC721 {
					return "2"
				}
				return "2000000000000000000"
			},
			requiredAmountInWei: func(t protobuf.CommunityTokenType) string {
				if t == protobuf.CommunityTokenType_ERC721 {
					return "2"
				}
				return "2000000000000000000"
			},
			shouldSatisfy: true,
		},
		{
			name: "account does meet criteria",
			amountInWei: func(t protobuf.CommunityTokenType) string {
				if t == protobuf.CommunityTokenType_ERC721 {
					return "3"
				}
				return "3000000000000000000"
			},
			requiredAmountInWei: func(t protobuf.CommunityTokenType) string {
				if t == protobuf.CommunityTokenType_ERC721 {
					return "2"
				}
				return "2000000000000000000"
			},
			shouldSatisfy: true,
		},
	}

	permissionChecker := DefaultPermissionChecker{}
	chainID := uint64(1)
	contractAddress := gethcommon.HexToAddress("0x3d6afaa395c31fcd391fe3d562e75fe9e8ec7e6a")
	walletAddress := gethcommon.HexToAddress("0xD6b912e09E797D291E8D0eA3D3D17F8000e01c32")

	for _, tc := range testCases {
		for _, tokenType := range [](protobuf.CommunityTokenType){protobuf.CommunityTokenType_ERC20, protobuf.CommunityTokenType_ERC721} {
			s.Run(fmt.Sprintf("%s_%s", tc.name, tokenType.String()), func() {
				decimals := uint64(0)
				if tokenType == protobuf.CommunityTokenType_ERC20 {
					decimals = 18
				}
				permissions := map[string]*CommunityTokenPermission{
					"p1": {
						CommunityTokenPermission: &protobuf.CommunityTokenPermission{
							Id:   "p1",
							Type: protobuf.CommunityTokenPermission_BECOME_MEMBER,
							TokenCriteria: []*protobuf.TokenCriteria{
								{
									ContractAddresses: map[uint64]string{
										chainID: contractAddress.String(),
									},
									Type:        tokenType,
									Symbol:      "STT",
									TokenIds:    []uint64{},
									Decimals:    decimals,
									AmountInWei: tc.requiredAmountInWei(tokenType),
								},
							},
						},
					},
				}

				permissionsData, _ := PreParsePermissionsData(permissions)
				accountsAndChainIDs := []*AccountChainIDsCombination{
					{
						Address:  walletAddress,
						ChainIDs: []uint64{chainID},
					},
				}

				var getOwnedERC721Tokens ownedERC721TokensGetter = func(walletAddresses []gethcommon.Address, tokenRequirements map[uint64]map[string]*protobuf.TokenCriteria, chainIDs []uint64) (CollectiblesByChain, error) {
					amount, err := strconv.ParseUint(tc.amountInWei(protobuf.CommunityTokenType_ERC721), 10, 64)
					if err != nil {
						return nil, err
					}

					balances := []thirdparty.TokenBalance{}
					for i := uint64(0); i < amount; i++ {
						balances = append(balances, thirdparty.TokenBalance{
							TokenID: &bigint.BigInt{
								Int: new(big.Int).SetUint64(i + 1),
							},
							Balance: &bigint.BigInt{
								Int: new(big.Int).SetUint64(1),
							},
						})
					}

					return CollectiblesByChain{
						chainID: {
							walletAddress: {
								contractAddress: balances,
							},
						},
					}, nil
				}

				var getBalancesByChain balancesByChainGetter = func(ctx context.Context, accounts, tokens []gethcommon.Address, chainIDs []uint64) (BalancesByChain, error) {
					balance, ok := new(big.Int).SetString(tc.amountInWei(protobuf.CommunityTokenType_ERC20), 10)
					if !ok {
						return nil, errors.New("invalid conversion")
					}

					return BalancesByChain{
						chainID: {
							walletAddress: {
								contractAddress: (*hexutil.Big)(balance),
							},
						},
					}, nil
				}

				response, err := permissionChecker.checkPermissions(permissionsData[protobuf.CommunityTokenPermission_BECOME_MEMBER], accountsAndChainIDs, true, getOwnedERC721Tokens, getBalancesByChain)
				s.Require().NoError(err)
				s.Require().Equal(tc.shouldSatisfy, response.Satisfied)
			})
		}
	}
}
