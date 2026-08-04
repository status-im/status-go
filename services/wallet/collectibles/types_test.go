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
	communityHeader := fullCollectibleDataToHeader(&communityCollectible)

	require.Equal(t, CollectibleDataTypeHeader, communityHeader.DataType)
	require.Equal(t, communityCollectible.CollectibleData.ID, communityHeader.ID)

	require.NotEmpty(t, communityHeader.CollectibleData)
	require.NotEmpty(t, communityHeader.CollectionData)
	require.NotEmpty(t, communityHeader.CommunityData)
	require.NotEmpty(t, communityHeader.Ownership)

	nonCommunityCollectible := getNonCommunityCollectible()
	nonCommunityHeader := fullCollectibleDataToHeader(&nonCommunityCollectible)

	require.Equal(t, CollectibleDataTypeHeader, nonCommunityHeader.DataType)
	require.Equal(t, nonCommunityCollectible.CollectibleData.ID, nonCommunityHeader.ID)

	require.NotEmpty(t, nonCommunityHeader.CollectibleData)
	require.NotEmpty(t, nonCommunityHeader.CollectionData)
	require.Empty(t, nonCommunityHeader.CommunityData)
	require.NotEmpty(t, nonCommunityHeader.Ownership)
}

func TestFullCollectibleToDetails(t *testing.T) {
	communityCollectible := getCommunityCollectible()
	communityDetails := fullCollectibleDataToDetails(&communityCollectible)

	require.Equal(t, CollectibleDataTypeDetails, communityDetails.DataType)
	require.Equal(t, communityCollectible.CollectibleData.ID, communityDetails.ID)

	require.NotEmpty(t, communityDetails.CollectibleData)
	require.NotEmpty(t, communityDetails.CollectionData)
	require.NotEmpty(t, communityDetails.CommunityData)
	require.NotEmpty(t, communityDetails.Ownership)

	nonCommunityCollectible := getNonCommunityCollectible()
	nonCommunityDetails := fullCollectibleDataToDetails(&nonCommunityCollectible)

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

func sizedCollectible(imageSize, thumbnailSize, animationSize int64) thirdparty.FullCollectibleData {
	c := getNonCommunityCollectible()
	c.CollectibleData.ImageURL = "image"
	c.CollectibleData.ThumbnailURL = "thumbnail"
	c.CollectibleData.AnimationURL = "animation"
	c.CollectibleData.AnimationMediaType = "video/mp4"
	c.CollectibleData.ImageSize = imageSize
	c.CollectibleData.ThumbnailSize = thumbnailSize
	c.CollectibleData.AnimationSize = animationSize
	return c
}

func TestApplyAssetSizeLimit(t *testing.T) {
	const cap = 1000

	testCases := []struct {
		name              string
		maxSize           int64
		imageSize         int64
		thumbnailSize     int64
		animationSize     int64
		expectedImage     string
		expectedThumbnail string
		expectedAnimation string
	}{
		{
			name:              "no cap set leaves even an enormous asset alone",
			maxSize:           0,
			imageSize:         cap * 1000,
			thumbnailSize:     cap * 1000,
			animationSize:     cap * 1000,
			expectedImage:     "image",
			expectedThumbnail: "thumbnail",
			expectedAnimation: "animation",
		},
		{
			name:              "a size the provider did not report passes",
			maxSize:           cap,
			expectedImage:     "image",
			expectedThumbnail: "thumbnail",
			expectedAnimation: "animation",
		},
		{
			name:              "an asset exactly at the cap is not over it",
			maxSize:           cap,
			imageSize:         cap,
			thumbnailSize:     cap,
			animationSize:     cap,
			expectedImage:     "image",
			expectedThumbnail: "thumbnail",
			expectedAnimation: "animation",
		},
		{
			name:              "an oversized animation leaves the still behind",
			maxSize:           cap,
			imageSize:         cap - 1,
			thumbnailSize:     cap - 1,
			animationSize:     cap + 1,
			expectedImage:     "image",
			expectedThumbnail: "thumbnail",
			expectedAnimation: "",
		},
		{
			name:              "an oversized image leaves the thumbnail behind",
			maxSize:           cap,
			imageSize:         cap + 1,
			thumbnailSize:     cap - 1,
			animationSize:     cap - 1,
			expectedImage:     "",
			expectedThumbnail: "thumbnail",
			expectedAnimation: "animation",
		},
		{
			name:              "nothing small enough leaves nothing at all",
			maxSize:           cap,
			imageSize:         cap + 1,
			thumbnailSize:     cap + 1,
			animationSize:     cap + 1,
			expectedImage:     "",
			expectedThumbnail: "",
			expectedAnimation: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := []thirdparty.FullCollectibleData{
				sizedCollectible(tc.imageSize, tc.thumbnailSize, tc.animationSize),
			}

			applyAssetSizeLimit(data, tc.maxSize)

			c := data[0].CollectibleData
			require.Equal(t, tc.expectedImage, c.ImageURL)
			require.Equal(t, tc.expectedThumbnail, c.ThumbnailURL)
			require.Equal(t, tc.expectedAnimation, c.AnimationURL)
		})
	}
}

// An animation URL and its media type are one fact: consumers read an empty URL
// as "this collectible is still", and a media type left behind would contradict
// that.
func TestApplyAssetSizeLimitDropsTheAnimationMediaTypeWithTheURL(t *testing.T) {
	data := []thirdparty.FullCollectibleData{sizedCollectible(0, 0, 2000)}

	applyAssetSizeLimit(data, 1000)

	require.Empty(t, data[0].CollectibleData.AnimationURL)
	require.Empty(t, data[0].CollectibleData.AnimationMediaType)
}
