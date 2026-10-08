package tokenbalances_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/pkg/services/wallet/multistandardbalance"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
	"github.com/status-im/status-go/pkg/services/wallet/tokenbalances"
)

// BenchmarkGetBalances is one reader refresh after a fetch of 2 accounts x 8500
// tokens, 1 in 97 of them held.
func BenchmarkGetBalances(b *testing.B) {
	storage := multistandardbalance.NewStorageMemory()
	chainID := uint64(1)
	tokens := make([]*tokentypes.Token, 8500)
	for i := range tokens {
		tokens[i] = erc20Token(chainID, common.BigToAddress(big.NewInt(int64(0x700000+i))).Hex())
	}
	accounts := []common.Address{{19: 1}, {19: 2}}
	for _, account := range accounts {
		stored := multistandardbalance.ERC20Balances{Balances: map[common.Address]*big.Int{}, Answered: map[common.Address]struct{}{}}
		for i, token := range tokens {
			stored.Answered[token.Address] = struct{}{}
			if i%97 == 3 {
				stored.Balances[token.Address] = big.NewInt(int64(i))
			}
		}
		_, _, err := storage.UpdateERC20Balances(context.Background(), multistandardbalance.BalancesKey{Account: account, ChainID: chainID}, stored, fetchedState())
		require.NoError(b, err)
	}
	reader := tokenbalances.NewStorageMultistandardBalance(storage)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := reader.GetBalances(context.Background(), tokens, accounts)
		require.NoError(b, err)
	}
}
