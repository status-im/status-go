package token

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/go-wallet-sdk/pkg/tokens/parsers"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"

	walletcommon "github.com/status-im/status-go/pkg/services/wallet/common"
)

// Token keys are persisted (token preferences, ManageTokens group keys): types.TokenKey must keep producing
// exactly the historical format for every token we ship.
func TestTokenKeyMatchesPersistedFormatForEmbeddedLists(t *testing.T) {
	chains := walletcommon.AllChainIDsAsUint64()
	checked := 0
	for _, id := range initialListIDsFromEmbedded() {
		data, err := initialListProviderFromEmbedded(id)
		require.NoError(t, err, id)

		var parser parsers.TokenListParser = &parsers.StandardTokenListParser{}
		if id == walletcommon.StatusTokenListID {
			parser = &parsers.StatusTokenListParser{}
		}
		list, err := parser.Parse(data, chains)
		require.NoError(t, err, id)

		for _, token := range list.Tokens {
			want := fmt.Sprintf("%d-%s", token.ChainID, strings.ToLower(token.Address.Hex()))
			require.Equal(t, want, types.TokenKey(token.ChainID, token.Address), "list %s", id)
			require.Equal(t, want, token.Key(), "list %s", id)
			checked++
		}
	}
	t.Logf("checked %d token keys", checked)
	require.Greater(t, checked, 10000)
}
