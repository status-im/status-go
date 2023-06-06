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

	ts := []struct {
		URL        string
		ShouldFail bool
	}{
		{"https://www.youtube.com/watch?v=mzOyYtfXkb0", false},
		{"https://youtu.be/mzOyYtfXkb0", false},
		{"https://www.test.com/unknown", true},
	}

	for _, u := range ts {
		previewData, err := GetLinkPreviewData(u.URL)
		if u.ShouldFail {
			require.Error(t, err)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, statusTownhall.Site, previewData.Site)
		require.Equal(t, statusTownhall.Title, previewData.Title)
		require.Equal(t, statusTownhall.ThumbnailURL, previewData.ThumbnailURL)
	}
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

// Medium unfurling is failing - https://github.com/status-im/status-go/issues/2192
//
// func TestMediumLinkPreviewData(t *testing.T) {

// 	statusSecurityAudit := LinkPreviewData{
// 		Site:         "Medium",
// 		Title:        "A Look at the Status.im ICO Token Distribution",
// 		ThumbnailURL: "https://miro.medium.com/max/700/1*Smc0y_TOL1XsofS1wxa3rg.jpeg",
// 	}

// 	previewData, err := GetLinkPreviewData("https://medium.com/the-bitcoin-podcast-blog/a-look-at-the-status-im-ico-token-distribution-f5bcf7f00907")
// 	require.NoError(t, err)
// 	require.Equal(t, statusSecurityAudit.Site, previewData.Site)
// 	require.Equal(t, statusSecurityAudit.Title, previewData.Title)
// 	require.Equal(t, statusSecurityAudit.ThumbnailURL, previewData.ThumbnailURL)
// }

// Flaky test, gives the following error:
// Error: Received unexpected error: invalid character '<' looking for beginning of value
//
// func TestTwitterLinkPreviewData(t *testing.T) {
// 	statusTweet1 := LinkPreviewData{
// 		Site:  "Twitter",
// 		Title: "Crypto isn't going anywhere.— Status (@ethstatus) July 26, 2021",
// 	}
// 	statusTweet2 := LinkPreviewData{
// 		Site: "Twitter",
// 		Title: "🎉 Status v1.15 is a go! 🎉\n\n📌 Pin important messages in chats and groups" +
// 			"\n✏️ Edit messages after sending\n🔬 Scan QR codes with the browser\n⚡️ FASTER app navigation!" +
// 			"\nhttps://t.co/qKrhDArVKb— Status (@ethstatus) July 27, 2021",
// 	}
// 	statusProfile := LinkPreviewData{
// 		Site:  "Twitter",
// 		Title: "Tweets by ethstatus",
// 	}

// 	ts := []struct {
// 		URL        string
// 		Expected   LinkPreviewData
// 		ShouldFail bool
// 	}{
// 		{"https://twitter.com/ethstatus/status/1419674733885407236", statusTweet1, false},
// 		{"https://twitter.com/ethstatus/status/1420035091997278214", statusTweet2, false},
// 		{"https://twitter.com/ethstatus", statusProfile, false},
// 		{"https://www.test.com/unknown", LinkPreviewData{}, true},
// 	}

// 	for _, u := range ts {
// 		previewData, err := GetLinkPreviewData(u.URL)
// 		if u.ShouldFail {
// 			require.Error(t, err)
// 			continue
// 		}

// 		require.NoError(t, err)
// 		require.Equal(t, u.Expected.Site, previewData.Site)
// 		require.Equal(t, u.Expected.Title, previewData.Title)
// 		require.Equal(t, u.Expected.ThumbnailURL, previewData.ThumbnailURL)
// 	}
// }
