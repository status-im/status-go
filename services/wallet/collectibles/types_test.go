package collectibles

import (
	"testing"

	"github.com/status-im/status-go/services/wallet/thirdparty"

	"github.com/stretchr/testify/require"
)

func getCommunityCollectible() thirdparty.FullCollectibleData {
	return thirdparty.GenerateTestFullCollectiblesData(1)[0]
}

func getNonCommunityCollectible() thirdparty.FullCollectibleData {
	c := thirdparty.GenerateTestFullCollectiblesData(1)[0]
	c.CollectibleData.CommunityID = ""
	c.CollectionData.CommunityID = ""
	c.CommunityInfo = nil
	c.CollectibleCommunityInfo = nil
	return c
}

func TestFullCollectibleToHeader(t *testing.T) {
	communityCollectible := getCommunityCollectible()
	communityHeader := fullCollectibleDataToHeader(communityCollectible)

	require.Equal(t, CollectibleDataTypeHeader, communityHeader.DataType)
	require.Equal(t, communityCollectible.CollectibleData.ID, communityHeader.ID)

	require.NotEmpty(t, communityHeader.CollectibleData)
	require.NotEmpty(t, communityHeader.CollectionData)
	require.NotEmpty(t, communityHeader.CommunityData)
	require.NotEmpty(t, communityHeader.Ownership)

	nonCommunityCollectible := getNonCommunityCollectible()
	nonCommunityHeader := fullCollectibleDataToHeader(nonCommunityCollectible)

	require.Equal(t, CollectibleDataTypeHeader, nonCommunityHeader.DataType)
	require.Equal(t, nonCommunityCollectible.CollectibleData.ID, nonCommunityHeader.ID)

	require.NotEmpty(t, nonCommunityHeader.CollectibleData)
	require.NotEmpty(t, nonCommunityHeader.CollectionData)
	require.Empty(t, nonCommunityHeader.CommunityData)
	require.NotEmpty(t, nonCommunityHeader.Ownership)
}

func TestFullCollectibleToDetails(t *testing.T) {
	communityCollectible := getCommunityCollectible()
	communityDetails := fullCollectibleDataToDetails(communityCollectible)

	require.Equal(t, CollectibleDataTypeDetails, communityDetails.DataType)
	require.Equal(t, communityCollectible.CollectibleData.ID, communityDetails.ID)

	require.NotEmpty(t, communityDetails.CollectibleData)
	require.NotEmpty(t, communityDetails.CollectionData)
	require.NotEmpty(t, communityDetails.CommunityData)
	require.NotEmpty(t, communityDetails.Ownership)

	nonCommunityCollectible := getNonCommunityCollectible()
	nonCommunityDetails := fullCollectibleDataToDetails(nonCommunityCollectible)

	require.Equal(t, CollectibleDataTypeDetails, nonCommunityDetails.DataType)
	require.Equal(t, nonCommunityCollectible.CollectibleData.ID, nonCommunityDetails.ID)

	require.NotEmpty(t, nonCommunityDetails.CollectibleData)
	require.NotEmpty(t, nonCommunityDetails.CollectionData)
	require.Empty(t, nonCommunityDetails.CommunityData)
	require.NotEmpty(t, nonCommunityDetails.Ownership)
}

func TestFullCollectiblesToCommunityHeader(t *testing.T) {
	collectibles := make([]thirdparty.FullCollectibleData, 0, 10)
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			collectibles = append(collectibles, getCommunityCollectible())
		} else {
			collectibles = append(collectibles, getNonCommunityCollectible())
		}
	}

	communityHeaders := fullCollectiblesDataToCommunityHeader(collectibles)
	require.Equal(t, 5, len(communityHeaders))
}
