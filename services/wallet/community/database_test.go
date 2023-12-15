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
			CommunityName:         fmt.Sprintf("communityname-%d", i),
			CommunityColor:        fmt.Sprintf("communitycolor-%d", i),
			CommunityImage:        fmt.Sprintf("communityimage-%d", i),
			CommunityImagePayload: []byte(fmt.Sprintf("communityimagepayload-%d", i)),
		}
		result[communityID] = newCommunity
	}

	return result
}

func TestUpdateCommunityInfo(t *testing.T) {
	db, cleanup := setupCommunityDataDBTest(t)
	defer cleanup()

	communityData := generateTestCommunityInfo(10)
	extraCommunityID := "extra-community-id"

	for communityID, communityInfo := range communityData {
		communityInfo := communityInfo // Prevent lint warning G601: Implicit memory aliasing in for loop.
		err := db.SetCommunityInfo(communityID, &communityInfo)
		require.NoError(t, err)
	}
	err := db.SetCommunityInfo(extraCommunityID, nil)
	require.NoError(t, err)

	for communityID, communityInfo := range communityData {
		info, state, err := db.GetCommunityInfo(communityID)
		require.NoError(t, err)
		require.Equal(t, communityInfo, *info)
		require.True(t, state.LastUpdateSuccesful)
	}
	info, state, err := db.GetCommunityInfo(extraCommunityID)
	require.NoError(t, err)
	require.Empty(t, info)
	require.False(t, state.LastUpdateSuccesful)

	randomCommunityID := "random-community-id"
	info, state, err = db.GetCommunityInfo(randomCommunityID)
	require.NoError(t, err)
	require.Empty(t, info)
	require.Empty(t, state)
}
