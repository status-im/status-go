package urls

import (
	"github.com/stretchr/testify/require"
	"testing"
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

	metabolize := LinkPreviewData{
		Site:         "GitHub",
		Title:        "qfrank/metabolize",
		ThumbnailURL: "https://avatars3.githubusercontent.com/u/12406719?s=400&v=4",
	}

	previewData, err = GetLinkPreviewData("https://github.com/qfrank/metabolize")
	require.NoError(t, err)
	require.Equal(t, metabolize.Site, previewData.Site)
	require.Equal(t, metabolize.Title, previewData.Title)
	require.Equal(t, metabolize.ThumbnailURL, previewData.ThumbnailURL)

}
