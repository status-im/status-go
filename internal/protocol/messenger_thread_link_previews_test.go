package protocol

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
)

func TestReceivedThreadPreviewUsesOnlyAuthorizedLocalData(t *testing.T) {
	const URL = "https://status.app/g/?a2=group&thread-id=thread"
	for _, state := range []string{"no resolver", "inaccessible", "nil result", "wrong destination", "unverified result", "accessible"} {
		t.Run(state, func(t *testing.T) {
			messenger := &Messenger{}
			if state != "no resolver" {
				messenger.SetThreadLinkPreviewResolver(func(rawURL string) (*common.LinkPreview, error) {
					require.Equal(t, URL, rawURL)
					if state == "inaccessible" {
						return nil, fmt.Errorf("unavailable")
					}
					if state == "nil result" {
						return nil, nil
					}
					result := &common.LinkPreview{URL: URL, Title: "LOCAL TITLE", Description: "LOCAL ORIGIN", ThreadParentName: "LOCAL PARENT", ThreadAccessible: true,
						Thumbnail: common.LinkPreviewThumbnail{DataURI: "LOCAL IMAGE"}}
					if state == "wrong destination" {
						result.URL = "https://status.app/g/?a2=other&thread-id=other"
					}
					if state == "unverified result" {
						result.ThreadAccessible = false
					}
					return result, nil
				})
			}
			message := &common.Message{LinkPreviews: []common.LinkPreview{{URL: URL, Title: "UNTRUSTED TITLE", Description: "UNTRUSTED ORIGIN", ThreadParentName: "UNTRUSTED PARENT", ThreadAccessible: true,
				Thumbnail: common.LinkPreviewThumbnail{DataURI: "UNTRUSTED IMAGE"}},
				{URL: "https://example.org/", Title: "Ordinary preview"}},
				StatusLinkPreviews: []common.StatusLinkPreview{{URL: URL, Channel: &common.StatusCommunityChannelLinkPreview{DisplayName: "UNTRUSTED CHANNEL"}}},
			}
			messenger.resolveThreadLinkPreviews(message)
			require.Len(t, message.LinkPreviews, 2)
			require.Empty(t, message.StatusLinkPreviews)
			preview := message.LinkPreviews[0]
			if state == "accessible" {
				require.True(t, preview.ThreadAccessible)
				require.Equal(t, "LOCAL TITLE", preview.Title)
				require.Equal(t, "LOCAL ORIGIN", preview.Description)
				require.Equal(t, "LOCAL IMAGE", preview.Thumbnail.DataURI)
			} else {
				require.False(t, preview.ThreadAccessible)
				require.Equal(t, "Unknown thread name", preview.Title)
				require.Equal(t, "Unknown", preview.Description)
				require.Equal(t, "Unknown", preview.ThreadParentName)
				require.Empty(t, preview.Thumbnail.DataURI)
			}
			require.Equal(t, "Ordinary preview", message.LinkPreviews[1].Title)
		})
	}
}

func TestThreadPreviewPrivacyWithoutMediaServer(t *testing.T) {
	const URL = "https://status.app/p/?thread-id=thread"
	message := common.NewMessage()
	message.ChatMessage = &protobuf.ChatMessage{UnfurledLinks: []*protobuf.UnfurledLink{{Url: URL, Title: "PRIVATE TITLE"}}}
	messenger := &Messenger{}
	require.NoError(t, messenger.prepareMessage(message, nil))
	require.Len(t, message.LinkPreviews, 1)
	require.Equal(t, "Unknown thread name", message.LinkPreviews[0].Title)
	legacy := common.NewMessage()
	legacy.UnfurledStatusLinks = &protobuf.UnfurledStatusLinks{UnfurledStatusLinks: []*protobuf.UnfurledStatusLink{{Url: URL}}}
	require.NoError(t, messenger.prepareMessage(legacy, nil))
	require.Len(t, legacy.LinkPreviews, 1)
	require.False(t, legacy.LinkPreviews[0].ThreadAccessible)
}
