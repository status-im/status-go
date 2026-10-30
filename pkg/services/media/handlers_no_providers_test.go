package media

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// After SetDataProviders(nil, nil, nil) on logout, every provider-backed
// handler must reply 503 instead of dereferencing a nil provider.
func TestHandlersReturnUnavailableWithoutProviders(t *testing.T) {
	s := newReproMediaServer()
	s.SetDataProviders(nil, nil, nil)

	handlers := map[string]http.HandlerFunc{
		"contactImages":                 s.handleContactImages,
		"discordAuthorAvatar":           s.handleDiscordAuthorAvatar,
		"discordAttachment":             s.handleDiscordAttachment,
		"image":                         s.handleImage,
		"audio":                         s.handleAudio,
		"ipfs":                          s.handleIPFS,
		"communityTokenImages":          s.handleCommunityTokenImages,
		"communityDescriptionImages":    s.handleCommunityDescriptionImages,
		"communityDescriptionTokenImgs": s.handleCommunityDescriptionTokenImages,
		"walletCommunityImages":         s.handleWalletCommunityImages,
		"walletCollectionImages":        s.handleWalletCollectionImages,
		"walletCollectibleImages":       s.handleWalletCollectibleImages,
		"linkPreviewThumbnail":          s.handleLinkPreviewThumbnail,
		"linkPreviewFavicon":            s.handleLinkPreviewFavicon,
		"statusLinkPreviewThumbnail":    s.handleStatusLinkPreviewThumbnail,
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler(rr, httptest.NewRequest(http.MethodGet, "/", nil))
			require.Equal(t, http.StatusServiceUnavailable, rr.Code)
		})
	}
}
