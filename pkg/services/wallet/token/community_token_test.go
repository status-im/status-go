package token

import (
	"database/sql"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

func TestCustomCommunityPersistence(t *testing.T) {
	m, cleanup := setupTestTokenDB(t)
	defer cleanup()
	testCustomCommunityPersistence(t, m)
}

func testCustomCommunityPersistence(t *testing.T, m *Manager) {
	t.Helper()
	row := tokentypes.Token{Token: &types.Token{ChainID: 1, Address: common.HexToAddress("0x1234"), Symbol: "CUSTOM", Decimals: 18}}
	for _, community := range []*tokentypes.CommunityData{nil, {ID: "community"}, {ID: ""}, {ID: "community"}, nil} {
		row.CommunityData = community
		require.NoError(t, m.UpsertCustom(row))
		isCommunity := community != nil && community.ID != ""
		var stored sql.NullString
		require.NoError(t, m.walletDB.QueryRow("SELECT community_id FROM tokens WHERE network_id=? AND address=?", row.ChainID, row.Address).Scan(&stored))
		require.Equal(t, isCommunity, stored.Valid)
		if isCommunity {
			require.Equal(t, community.ID, stored.String)
		}
		for _, communityOnly := range []bool{false, true} {
			rows, err := m.GetCustoms(communityOnly)
			require.NoError(t, err)
			if communityOnly == isCommunity {
				require.Len(t, rows, 1)
				require.Equal(t, row.Address, rows[0].Address)
			} else {
				require.Empty(t, rows)
			}
		}
	}
}
