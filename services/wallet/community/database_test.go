package community

import (
	"fmt"
	"testing"

	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func setupCommunityDataDBTest(t *testing.T) (*DataDB, func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return NewDataDB(db), func() {
		require.NoError(t, db.Close())
	}
}

func generateTestCommunityInfo(count int) map[string]thirdparty.CommunityInfo {
	result := make(map[string]thirdparty.CommunityInfo)
	for i := 0; i < count; i++ {
		communityID := fmt.Sprintf("communityid-%d", i)
		newCommunity := thirdparty.CommunityInfo{
			CommunityName:  fmt.Sprintf("communityname-%d", i),
			CommunityColor: fmt.Sprintf("communitycolor-%d", i),
			CommunityImage: fmt.Sprintf("communityimage-%d", i),
		}
		result[communityID] = newCommunity
	}

	return result
}

func TestUpdateCommunityInfo(t *testing.T) {
	db, cleanup := setupCommunityDataDBTest(t)
	defer cleanup()

	communityData := generateTestCommunityInfo(10)
	for communityID, communityInfo := range communityData {
		err := db.SetCommunityInfo(communityID, communityInfo)
		require.NoError(t, err)
	}

	for communityID, communityInfo := range communityData {
		communityInfoFromDB, err := db.GetCommunityInfo(communityID)
		require.NoError(t, err)
		require.Equal(t, communityInfo, *communityInfoFromDB)
	}
}
