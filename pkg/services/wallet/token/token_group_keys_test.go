package token

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

type TokensByGroupKeysSuite struct {
	suite.Suite
	tm *Manager
}

func TestTokensByGroupKeysSuite(t *testing.T) {
	suite.Run(t, new(TokensByGroupKeysSuite))
}

func (s *TokensByGroupKeysSuite) SetupTest() {
	s.tm = setupMarketTestManager(s.T(), false)
}

// groupKeyOf is the client's group key: the cross-chain id when set, else the token key.
func groupKeyOf(token *tokentypes.Token) string {
	if strings.TrimSpace(token.CrossChainID) == "" {
		return token.Key()
	}
	return token.CrossChainID
}

// legacyTokensByGroupKey is the client's former index: every GetAllTokens token
// grouped by groupKeyOf, in GetAllTokens order.
func (s *TokensByGroupKeysSuite) legacyTokensByGroupKey() map[string][]tokentypes.Token {
	all, err := s.tm.GetAllTokens()
	s.Require().NoError(err)
	result := make(map[string][]tokentypes.Token)
	for _, token := range all {
		key := groupKeyOf(token)
		result[key] = append(result[key], *token)
	}
	return result
}

func groupTokens(tokens []*tokentypes.Token) map[string][]tokentypes.Token {
	result := make(map[string][]tokentypes.Token)
	for _, token := range tokens {
		key := groupKeyOf(token)
		result[key] = append(result[key], *token)
	}
	return result
}

func (s *TokensByGroupKeysSuite) addCustom(chainID uint64, address string, crossChainID string) *tokentypes.Token {
	token := &tokentypes.Token{
		Token:         &types.Token{ChainID: chainID, Address: common.HexToAddress(address), Symbol: "COMM", Name: "Community", Decimals: 18, CrossChainID: crossChainID},
		CommunityData: &tokentypes.CommunityData{ID: "community"},
	}
	upsertCommunityToken(s.T(), token, s.tm)
	return token
}

func (s *TokensByGroupKeysSuite) TestEachGroupKeyMatchesAllTokensGrouping() {
	s.addCustom(walletcommon.EthereumMainnet, "0xc0ffee", "usd-coin")
	custom := s.addCustom(walletcommon.EthereumMainnet, "0xbeef", "")
	legacy := s.legacyTokensByGroupKey()
	s.Require().Greater(len(legacy), 1000)
	s.Require().Contains(legacy, custom.Key())

	for key, want := range legacy {
		got, err := s.tm.GetTokensByGroupKeys([]string{key})
		s.Require().NoError(err)
		s.Require().Equal(map[string][]tokentypes.Token{key: want}, groupTokens(got), key)
	}
}

func (s *TokensByGroupKeysSuite) TestBatchMatchesAllTokensGrouping() {
	legacy := s.legacyTokensByGroupKey()
	keys := make([]string, 0, len(legacy))
	for key := range legacy {
		keys = append(keys, key)
	}
	got, err := s.tm.GetTokensByGroupKeys(keys)
	s.Require().NoError(err)
	s.Require().Equal(legacy, groupTokens(got))
}

func (s *TokensByGroupKeysSuite) TestTokenKeyOfCrossChainTokenIsNotAGroupKey() {
	legacy := s.legacyTokensByGroupKey()
	usdc := legacy["usd-coin"]
	s.Require().NotEmpty(usdc)

	got, err := s.tm.GetTokensByGroupKeys([]string{usdc[0].Key()})
	s.Require().NoError(err)
	s.Require().Empty(got)
}

func (s *TokensByGroupKeysSuite) TestUnknownKeysReturnEmpty() {
	for _, keys := range [][]string{nil, {}, {""}, {"not-a-group"}, {"999999-0x0000000000000000000000000000000000000001"}} {
		got, err := s.tm.GetTokensByGroupKeys(keys)
		s.Require().NoError(err)
		s.Require().NotNil(got, keys)
		s.Require().Empty(got, keys)
	}
}
