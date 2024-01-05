package storenodes

import (
	"testing"

	"github.com/status-im/status-go/services/mailservers"
	"github.com/stretchr/testify/require"
)

func TestReloadFromDB(t *testing.T) {
	db, close := setupTestDB(t, communityID1, communityID2)
	defer close()
	csn := NewCommunityStorenodes(db)
	snodes1 := []Storenode{
		{
			CommunityID: communityID1,
			StorenodeID: "storenode001",
			Name:        "My Mailserver",
			Address:     "enode://...",
			Fleet:       "prod",
			Version:     2,
		},
	}
	snodes2 := []Storenode{
		{
			CommunityID: communityID2,
			StorenodeID: "storenode002",
			Name:        "My Mailserver",
			Address:     "enode://...",
			Fleet:       "prod",
			Version:     2,
		},
	}
	// populate db
	err := db.syncSave(communityID1, snodes1, 0)
	require.NoError(t, err)
	err = db.syncSave(communityID2, snodes2, 0)
	require.NoError(t, err)

	err = csn.ReloadFromDB()
	require.NoError(t, err)

	// check if storenodes are loaded
	ms1, err := csn.GetStorenodeByCommunnityID(communityID1.String())
	require.NoError(t, err)
	matchStoreNode(t, snodes1[0], ms1)

	ms2, err := csn.GetStorenodeByCommunnityID(communityID2.String())
	require.NoError(t, err)
	matchStoreNode(t, snodes2[0], ms2)
}

func matchStoreNode(t *testing.T, sn Storenode, ms mailservers.Mailserver) {
	require.Equal(t, sn.StorenodeID, ms.ID)
	require.Equal(t, sn.Name, ms.Name)
	require.Equal(t, sn.Address, ms.Address)
	require.Equal(t, sn.Fleet, ms.Fleet)
	require.Equal(t, sn.Version, ms.Version)
}
