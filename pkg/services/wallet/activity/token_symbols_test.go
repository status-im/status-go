package activity

import (
	"math/big"
	"testing"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	wCommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

func TestFillInTokenSymbolsUsesOneLookup(t *testing.T) {
	token := func(address string) *ac.Token {
		return &ac.Token{TokenType: ac.Erc20, ChainID: wCommon.ChainID(1), Address: eth.HexToAddress(address)}
	}
	nft := token("0x3")
	nft.TokenID = (*hexutil.Big)(big.NewInt(1))
	entries := []Entry{{tokenOut: token("0x1"), tokenIn: token("0x2")}, {tokenOut: nft}, {tokenIn: token("0x9")}, {}}
	calls := 0
	deps := FilterDependencies{tokenSymbols: func(tokens []ac.Token) []string {
		calls++
		symbols := make([]string, len(tokens))
		for i, token := range tokens {
			require.Nil(t, token.TokenID)
			switch token.Address {
			case eth.HexToAddress("0x1"):
				symbols[i] = "ONE"
			case eth.HexToAddress("0x2"):
				symbols[i] = "TWO"
			}
		}
		return symbols
	}}
	fillInTokenSymbols(deps, entries)
	require.Equal(t, 1, calls)
	require.Equal(t, "ONE", *entries[0].symbolOut)
	require.Equal(t, "TWO", *entries[0].symbolIn)
	require.Nil(t, entries[1].symbolOut)
	require.Nil(t, entries[2].symbolIn)
	require.Nil(t, entries[3].symbolOut)
	fillInTokenSymbols(FilterDependencies{}, entries[3:])
}
