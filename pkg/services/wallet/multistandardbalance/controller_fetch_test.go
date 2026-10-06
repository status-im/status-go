package multistandardbalance

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/go-wallet-sdk/pkg/balance/multistandardfetcher"
)

type countingTokenListProvider struct {
	calls map[uint64]int
}

func (p *countingTokenListProvider) GetTokenContractAddresses(chainID uint64) ([]common.Address, error) {
	p.calls[chainID]++
	return []common.Address{{byte(chainID)}}, nil
}

func TestBuildFetchConfigsReadsTokenListOncePerChain(t *testing.T) {
	provider := &countingTokenListProvider{calls: make(map[uint64]int)}
	c := NewController(DefaultControllerConfig(), NewStorageMemory(), nil, nil, nil, nil, provider, nil, nil, nil, zap.NewNop())

	erc20 := []multistandardfetcher.ResultType{multistandardfetcher.ResultTypeERC20}
	toFetch := make(map[BalancesKey][]multistandardfetcher.ResultType)
	for _, chainID := range []uint64{1, 10} {
		for i := 0; i < 3; i++ {
			toFetch[BalancesKey{Account: common.Address{19: byte(i + 1)}, ChainID: chainID}] = erc20
		}
	}

	configs := c.buildMultiStandardFetcherFetchConfigs(toFetch)

	require.Equal(t, map[uint64]int{1: 1, 10: 1}, provider.calls)
	for _, chainID := range []uint64{1, 10} {
		require.Len(t, configs[chainID].ERC20, 3)
		for _, tokens := range configs[chainID].ERC20 {
			require.Equal(t, []common.Address{{byte(chainID)}}, tokens)
		}
	}
}
