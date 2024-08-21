package storenodes

import (
	"testing"

	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/mailservers"
)

func TestSerialization(t *testing.T) {
	maddr, err := multiaddr.NewMultiaddr("/dns4/test.net/tcp/30303/p2p/16Uiu2HAmMELCo218hncCtTvC2Dwbej3rbyHQcR8erXNnKGei7WPZ")
	require.NoError(t, err)
	snodes := Storenodes{
		{
			CommunityID: communityID1,
			StorenodeID: "storenode001",
			Name:        "My Mailserver",
			Address:     maddr,
			Fleet:       "prod",
			Version:     2,
		},
	}

	snodesProtobuf := snodes.ToProtobuf()

	snodes2 := FromProtobuf(snodesProtobuf, 0)

	require.Equal(t, snodes[0].Address.String(), snodes2[0].Address.String())
}

func TestUpdateStorenodesInDB(t *testing.T) {
	db, close := setupTestDB(t, communityID1, communityID2)
	defer close()

	maddr, err := multiaddr.NewMultiaddr("/dns4/test.net/tcp/30303/p2p/16Uiu2HAmMELCo218hncCtTvC2Dwbej3rbyHQcR8erXNnKGei7WPZ")
	require.NoError(t, err)

	csn := NewCommunityStorenodes(db, nil)
	snodes1 := []Storenode{
		{
			CommunityID: communityID1,
			StorenodeID: "storenode001",
			Name:        "My Mailserver",
			Address:     maddr,
			Fleet:       "prod",
			Version:     2,
		},
	}
	snodes2 := []Storenode{
		{
			CommunityID: communityID2,
			StorenodeID: "storenode002",
			Name:        "My Mailserver",
			Address:     maddr,
			Fleet:       "prod",
			Version:     2,
		},
	}
	// populate db
	err = csn.UpdateStorenodesInDB(communityID1, snodes1, 0)
	require.NoError(t, err)
	err = csn.UpdateStorenodesInDB(communityID2, snodes2, 0)
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
	require.Equal(t, sn.Address.String(), (*ms.Addr).String())
	require.Equal(t, sn.Fleet, ms.Fleet)
}
