package common

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestThreadPreviewMetadataIsNotSent(t *testing.T) {
	for _, URL := range []string{
		"https://status.app/p/?thread-id=thread",
		"http://status.app/g/?a2=group&thread-id=thread",
		"status-app://cc/channel?thread-id=thread#community",
		"https://status.app/p/?thread-id=%zz",
	} {
		t.Run(URL, func(t *testing.T) {
			message := &Message{LinkPreviews: []LinkPreview{{
				URL: URL, Hostname: "status.app", Title: "PRIVATE TITLE", Description: "PRIVATE ORIGIN",
				ThreadParentName: "PRIVATE PARENT", ThreadAccessible: true,
				Thumbnail: LinkPreviewThumbnail{DataURI: "PRIVATE IMAGE"},
				Favicon:   LinkPreviewThumbnail{DataURI: "PRIVATE ICON"},
			}}}
			links, err := message.ConvertLinkPreviewsToProto()
			require.NoError(t, err)
			require.Len(t, links, 1)
			payload, err := proto.Marshal(links[0])
			require.NoError(t, err)
			require.NotContains(t, string(payload), "PRIVATE")
			require.Equal(t, URL, links[0].Url)
			require.Equal(t, "Unknown thread name", links[0].Title)
			require.Equal(t, "Unknown", links[0].Description)
			require.Empty(t, links[0].ThumbnailPayload)
			require.Empty(t, links[0].FaviconPayload)
			require.Equal(t, "PRIVATE TITLE", message.LinkPreviews[0].Title)
		})
	}
}

func TestLegacyThreadStatusPreviewIsNotSent(t *testing.T) {
	const URL = "https://status.app/cc/channel?thread-id=thread#community"
	message := &Message{StatusLinkPreviews: []StatusLinkPreview{{
		URL: URL, Channel: &StatusCommunityChannelLinkPreview{DisplayName: "PRIVATE CHANNEL"},
	}}}
	links, err := message.ConvertLinkPreviewsToProto()
	require.NoError(t, err)
	require.Len(t, links, 1)
	require.Equal(t, "Unknown thread name", links[0].Title)
	status, err := message.ConvertStatusLinkPreviewsToProto()
	require.NoError(t, err)
	require.Nil(t, status)
}
