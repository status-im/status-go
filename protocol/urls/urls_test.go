package urls

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLinkPreviewData(t *testing.T) {

	statusTownhall := LinkPreviewData{
		Site:         "YouTube",
		Title:        "Status Town Hall #67 - 12 October 2020",
		ThumbnailURL: "https://i.ytimg.com/vi/mzOyYtfXkb0/hqdefault.jpg",
	}

	previewData, err := GetLinkPreviewData("https://www.youtube.com/watch?v=mzOyYtfXkb0")
	require.NoError(t, err)
	require.Equal(t, statusTownhall.Site, previewData.Site)
	require.Equal(t, statusTownhall.Title, previewData.Title)
	require.Equal(t, statusTownhall.ThumbnailURL, previewData.ThumbnailURL)

	previewData, err = GetLinkPreviewData("https://youtu.be/mzOyYtfXkb0")
	require.NoError(t, err)
	require.Equal(t, statusTownhall.Site, previewData.Site)
	require.Equal(t, statusTownhall.Title, previewData.Title)
	require.Equal(t, statusTownhall.ThumbnailURL, previewData.ThumbnailURL)

	_, err = GetLinkPreviewData("https://www.test.com/unknown")
	require.Error(t, err)

}

func TestGetGiphyPreviewData(t *testing.T) {
	validGiphyLink := "https://giphy.com/gifs/FullMag-robot-boston-dynamics-dance-lcG3qwtTKSNI2i5vst"
	previewData, err := GetGiphyPreviewData(validGiphyLink)
	bostonDynamicsEthGifData := LinkPreviewData{
		Site:         "GIPHY",
		Title:        "Boston Dynamics Yes GIF by FullMag - Find & Share on GIPHY",
		ThumbnailURL: "https://media1.giphy.com/media/lcG3qwtTKSNI2i5vst/giphy.gif",
	}
	require.NoError(t, err)
	require.Equal(t, bostonDynamicsEthGifData.Site, previewData.Site)
	require.Equal(t, bostonDynamicsEthGifData.Title, previewData.Title)
	require.Equal(t, bostonDynamicsEthGifData.ThumbnailURL, previewData.ThumbnailURL)

	invalidGiphyLink := "https://giphy.com/gifs/this-gif-does-not-exist-44444"
	_, err = GetGiphyPreviewData(invalidGiphyLink)
	require.Error(t, err)
}
