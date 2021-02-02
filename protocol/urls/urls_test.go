package urls

import (
	"strings"
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

// split at "." and ignore the first item
func thumbnailURLWithoutSubdomain(url string) []string {
	return strings.Split(url, ".")[1:]
}

func TestGetGiphyPreviewData(t *testing.T) {
	validGiphyLink := "https://giphy.com/gifs/FullMag-robot-boston-dynamics-dance-lcG3qwtTKSNI2i5vst"
	previewData, err := GetGiphyPreviewData(validGiphyLink)
	bostonDynamicsEthGifData := LinkPreviewData{
		Site:         "GIPHY",
		Title:        "Boston Dynamics Yes GIF by FullMag - Find & Share on GIPHY",
		ThumbnailURL: "https://media1.giphy.com/media/lcG3qwtTKSNI2i5vst/giphy.gif",
		Height:       480,
		Width:        480,
	}
	require.NoError(t, err)
	require.Equal(t, bostonDynamicsEthGifData.Site, previewData.Site)
	require.Equal(t, bostonDynamicsEthGifData.Title, previewData.Title)
	require.Equal(t, bostonDynamicsEthGifData.Height, previewData.Height)
	require.Equal(t, bostonDynamicsEthGifData.Width, previewData.Width)

	// Giphy oembed returns links to different servers: https://media1.giphy.com, https://media2.giphy.com and so on
	// We don't care about the server as long as other parts are equal, so we split at "." and ignore the first item
	require.Equal(t, thumbnailURLWithoutSubdomain(bostonDynamicsEthGifData.ThumbnailURL), thumbnailURLWithoutSubdomain(previewData.ThumbnailURL))

	invalidGiphyLink := "https://giphy.com/gifs/this-gif-does-not-exist-44444"
	_, err = GetGiphyPreviewData(invalidGiphyLink)
	require.Error(t, err)

	mediaLink := "https://media.giphy.com/media/lcG3qwtTKSNI2i5vst/giphy.gif"

	mediaLinkData, _ := GetGiphyPreviewData(mediaLink)

	require.Equal(t, thumbnailURLWithoutSubdomain(mediaLinkData.ThumbnailURL), thumbnailURLWithoutSubdomain(previewData.ThumbnailURL))
}

func TestGetGiphyLongURL(t *testing.T) {
	shortURL := "https://gph.is/g/aXLyK7P"
	computedLongURL, _ := GetGiphyLongURL(shortURL)
	actualLongURL := "https://giphy.com/gifs/FullMag-robot-boston-dynamics-dance-lcG3qwtTKSNI2i5vst"

	require.Equal(t, computedLongURL, actualLongURL)

	_, err := GetGiphyLongURL("http://this-giphy-site-doesn-not-exist.se/bogus-url")
	require.Error(t, err)

	_, err = GetGiphyLongURL("http://gph.is/bogus-url-but-correct-domain")
	require.Error(t, err)
}

func TestGetGiphyShortURLPreviewData(t *testing.T) {
	shortURL := "https://gph.is/g/aXLyK7P"
	previewData, err := GetGiphyShortURLPreviewData(shortURL)

	bostonDynamicsEthGifData := LinkPreviewData{
		Site:         "GIPHY",
		Title:        "Boston Dynamics Yes GIF by FullMag - Find & Share on GIPHY",
		ThumbnailURL: "https://media1.giphy.com/media/lcG3qwtTKSNI2i5vst/giphy.gif",
	}

	require.NoError(t, err)
	require.Equal(t, bostonDynamicsEthGifData.Site, previewData.Site)
	require.Equal(t, bostonDynamicsEthGifData.Title, previewData.Title)
}

func TestGetTenorPreviewData(t *testing.T) {
	validTenorLink := "https://tenor.com/view/robot-dance-do-you-love-me-boston-boston-dynamics-dance-gif-19998728"
	previewData, err := GetTenorPreviewData(validTenorLink)

	gifData := LinkPreviewData{
		Site:         "Tenor",
		Title:        "Annihere",
		ThumbnailURL: "https://media.tenor.com/images/975f6b95d188c277ebba62d9b5511685/tenor.gif",
		Height:       400,
		Width:        600,
	}
	require.NoError(t, err)
	require.Equal(t, gifData.Site, previewData.Site)
	require.Equal(t, gifData.Title, previewData.Title)
	require.Equal(t, gifData.ThumbnailURL, previewData.ThumbnailURL)
	require.Equal(t, gifData.Height, previewData.Height)
	require.Equal(t, gifData.Width, previewData.Width)

	invalidTenorLink := "https://giphy.com/gifs/this-gif-does-not-exist-44444"
	_, err = GetTenorPreviewData(invalidTenorLink)
	require.Error(t, err)
}

func TestStatusLinkPreviewData(t *testing.T) {

	statusSecurityAudit := LinkPreviewData{
		Site:         "Our Status",
		Title:        "What is a Security Audit, When You Should Get One, and How to Prepare.",
		ThumbnailURL: "https://our.status.im/content/images/2021/02/Security-Audit-Header.png",
	}

	previewData, err := GetLinkPreviewData("https://our.status.im/what-is-a-security-audit-when-you-should-get-one-and-how-to-prepare/")
	require.NoError(t, err)
	require.Equal(t, statusSecurityAudit.Site, previewData.Site)
	require.Equal(t, statusSecurityAudit.Title, previewData.Title)
	require.Equal(t, statusSecurityAudit.ThumbnailURL, previewData.ThumbnailURL)
}

func TestMediumLinkPreviewData(t *testing.T) {

	statusSecurityAudit := LinkPreviewData{
		Site:         "Medium",
		Title:        "A Look at the Status.im ICO Token Distribution",
		ThumbnailURL: "https://miro.medium.com/max/700/1*Smc0y_TOL1XsofS1wxa3rg.jpeg",
	}

	previewData, err := GetLinkPreviewData("https://medium.com/the-bitcoin-podcast-blog/a-look-at-the-status-im-ico-token-distribution-f5bcf7f00907")
	require.NoError(t, err)
	require.Equal(t, statusSecurityAudit.Site, previewData.Site)
	require.Equal(t, statusSecurityAudit.Title, previewData.Title)
	require.Equal(t, statusSecurityAudit.ThumbnailURL, previewData.ThumbnailURL)
}
