package storenodes

import (
	"testing"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/stretchr/testify/require"
)

var (
	communityID1 = types.HexBytes("community001")
	communityID2 = types.HexBytes("community002")
)

func TestSyncSave(t *testing.T) {
	db, close := setupTestDB(t, communityID1)
	defer close()
	snodes := []Storenode{
		{
			CommunityID: communityID1,
			StorenodeID: "storenode001",
			Name:        "My Mailserver",
			Address:     "enode://...",
			Fleet:       "prod",
			Version:     2,
		},
	}

	// ========
	// Save

	err := db.syncSave(communityID1, snodes, 0)
	require.NoError(t, err)

	dbNodes, err := db.getByCommunityID(communityID1)
	require.NoError(t, err)

	require.Len(t, dbNodes, 1)
	require.ElementsMatch(t, dbNodes, snodes)

	// ========
	// Update

	updated := []Storenode{
		{
			CommunityID: communityID1,
			StorenodeID: "storenode001",
			Name:        "My Mailserver 2",
			Address:     "enode://...",
			Fleet:       "prod",
			Version:     2,
		},
	}
	err = db.syncSave(communityID1, updated, 0)
	require.NoError(t, err)

	dbNodes, err = db.getByCommunityID(communityID1)
	require.NoError(t, err)

	require.Len(t, dbNodes, 1)
	require.ElementsMatch(t, dbNodes, updated)

	// ========
	// Remove

	err = db.syncSave(communityID1, []Storenode{}, 0)
	require.NoError(t, err)

	dbNodes, err = db.getByCommunityID(communityID1)
	require.NoError(t, err)

	require.Len(t, dbNodes, 0)
}
