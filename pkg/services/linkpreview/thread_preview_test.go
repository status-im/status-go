package linkpreview

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/protobuf/proto"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/services/linkpreview/unfurlers"
	mock_unfurlers "github.com/status-im/status-go/pkg/services/linkpreview/unfurlers/mock"
)

type threadHTTPGuard struct{ requests int }

func (g *threadHTTPGuard) RoundTrip(*http.Request) (*http.Response, error) {
	g.requests++
	return nil, fmt.Errorf("thread previews must not use HTTP")
}

func TestThreadPreviewAndMessageRoundTrip(t *testing.T) {
	const link = "https://status.app/g/?a2=group&thread-id=thread"
	provider := mock_unfurlers.NewMockStatusDataProvider(gomock.NewController(t))
	provider.EXPECT().GetThreadPreviewData(link).Return(&unfurlers.ThreadPreviewData{
		Name: "Release planning", ParentName: "Team chat",
	}, nil)
	guard := &threadHTTPGuard{}
	response, err := UnfurlURLs([]string{link}, &http.Client{Transport: guard}, provider, zap.NewNop())
	require.NoError(t, err)
	require.Zero(t, guard.requests)
	require.Empty(t, response.StatusLinkPreviews)
	require.Len(t, response.LinkPreviews, 1)
	preview := response.LinkPreviews[0]
	require.Equal(t, link, preview.URL)
	require.Equal(t, "Release planning", preview.Title)
	require.Equal(t, "Team chat", preview.Description)
	require.Equal(t, "Team chat", preview.ThreadParentName)
	require.Equal(t, "status.app", preview.Hostname)
	message := &common.Message{LinkPreviews: []common.LinkPreview{*preview}}
	wire, err := message.ConvertLinkPreviewsToProto()
	require.NoError(t, err)
	encoded, err := proto.Marshal(wire[0])
	require.NoError(t, err)
	decoded := &protobuf.UnfurledLink{}
	require.NoError(t, proto.Unmarshal(encoded, decoded))
	received := &common.Message{ChatMessage: &protobuf.ChatMessage{UnfurledLinks: []*protobuf.UnfurledLink{decoded}}}
	loaded := received.ConvertFromProtoToLinkPreviews(nil, nil)
	require.Len(t, loaded, 1)
	require.Equal(t, preview.URL, loaded[0].URL)
	require.True(t, preview.ThreadAccessible)
	require.Equal(t, "Unknown thread name", loaded[0].Title)
	require.Equal(t, "Unknown", loaded[0].Description)
	require.Empty(t, loaded[0].ThreadParentName)
	require.False(t, loaded[0].ThreadAccessible)
	plan := GetTextURLsToUnfurl(link, settings.URLUnfurlingEnableAll)
	require.Len(t, plan.URLs, 1)
	require.True(t, plan.URLs[0].IsStatusSharedURL)
	require.Equal(t, URLUnfurlingAllowed, plan.URLs[0].Permission)
}

func TestUnavailableThreadDoesNotFallBackToParentOrHTTP(t *testing.T) {
	const link = "https://status.app/cc/channel?thread-id=missing#community"
	provider := mock_unfurlers.NewMockStatusDataProvider(gomock.NewController(t))
	provider.EXPECT().GetThreadPreviewData(link).Return(nil, fmt.Errorf("inaccessible thread"))
	guard := &threadHTTPGuard{}
	response, err := UnfurlURLs([]string{link}, &http.Client{Transport: guard}, provider, zap.NewNop())
	require.NoError(t, err)
	require.Len(t, response.LinkPreviews, 1)
	require.False(t, response.LinkPreviews[0].ThreadAccessible)
	require.Equal(t, "Unknown thread name", response.LinkPreviews[0].Title)
	require.Empty(t, response.StatusLinkPreviews)
	require.Zero(t, guard.requests)
}

func TestPrivateThreadPhotoRemainsLocal(t *testing.T) {
	for _, tc := range []struct {
		name, link, image string
		hasPhoto          bool
	}{
		{"group photo", "https://status.app/g/?a2=group&thread-id=thread", exampleIdenticonURI, true},
		{"DM photo", "https://status.app/p/?thread-id=thread", exampleIdenticonURI, true},
		{"no photo", "https://status.app/g/?a2=group&thread-id=thread", "", false},
		{"invalid photo", "https://status.app/p/?thread-id=thread", "data:image/png;base64,invalid", false},
		{"remote photo is not fetched", "https://status.app/p/?thread-id=thread", "https://example.org/photo.png", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := mock_unfurlers.NewMockStatusDataProvider(gomock.NewController(t))
			provider.EXPECT().GetThreadPreviewData(tc.link).Return(&unfurlers.ThreadPreviewData{
				Name: "Planning", ParentName: "Chat", ImageDataURI: tc.image,
			}, nil)
			guard := &threadHTTPGuard{}
			response, err := UnfurlURLs([]string{tc.link}, &http.Client{Transport: guard}, provider, zap.NewNop())
			require.NoError(t, err)
			require.Zero(t, guard.requests)
			require.Len(t, response.LinkPreviews, 1)
			message := &common.Message{LinkPreviews: []common.LinkPreview{*response.LinkPreviews[0]}}
			links, err := message.ConvertLinkPreviewsToProto()
			require.NoError(t, err)
			encoded, err := proto.Marshal(links[0])
			require.NoError(t, err)
			decoded := &protobuf.UnfurledLink{}
			require.NoError(t, proto.Unmarshal(encoded, decoded))
			message = &common.Message{ChatMessage: &protobuf.ChatMessage{UnfurledLinks: []*protobuf.UnfurledLink{decoded}}}
			loaded := message.ConvertFromProtoToLinkPreviews(func(_, _ string) string { return "https://localhost:1234/photo" }, nil)
			require.Len(t, loaded, 1)
			require.Equal(t, "Unknown thread name", loaded[0].Title)
			require.Empty(t, decoded.ThumbnailPayload)
			require.Empty(t, loaded[0].Thumbnail.URL)
			if tc.hasPhoto {
				require.NotEmpty(t, response.LinkPreviews[0].Thumbnail.DataURI)
			} else {
				require.Empty(t, response.LinkPreviews[0].Thumbnail.DataURI)
			}

		})
	}
}

func TestUnavailablePrivateThreadDoesNotLogURL(t *testing.T) {
	const link = "https://status.app/p/?thread-id=private-root"
	provider := mock_unfurlers.NewMockStatusDataProvider(gomock.NewController(t))
	provider.EXPECT().GetThreadPreviewData(link).Return(nil, fmt.Errorf("cannot resolve %s", link))
	core, logs := observer.New(zap.DebugLevel)
	guard := &threadHTTPGuard{}
	response, err := UnfurlURLs([]string{link}, &http.Client{Transport: guard}, provider, zap.New(core))
	require.NoError(t, err)
	require.Len(t, response.LinkPreviews, 1)
	require.False(t, response.LinkPreviews[0].ThreadAccessible)
	require.Equal(t, "Unknown thread name", response.LinkPreviews[0].Title)
	require.Empty(t, response.StatusLinkPreviews)
	require.Zero(t, guard.requests)
	require.NotEmpty(t, logs.All())
	require.NotContains(t, fmt.Sprint(logs.All()), "private-root")
}

func (s *LinkPreviewsTestSuite) TestThreadPreviewCommunityBannerRemainsLocal() {
	const link = "https://status.app/cc/channel?thread-id=thread#community"
	community := s.fakeCommunity()
	provider := mock_unfurlers.NewMockStatusDataProvider(s.ctrl)
	provider.EXPECT().GetThreadPreviewData(link).Return(&unfurlers.ThreadPreviewData{
		Name: "Release planning", ParentName: "general", Community: community,
	}, nil)
	response, err := UnfurlURLs([]string{link}, nil, provider, s.logger)
	s.Require().NoError(err)
	s.Require().Len(response.LinkPreviews, 1)
	s.Require().Empty(response.StatusLinkPreviews)
	preview := response.LinkPreviews[0]
	s.Require().NotEmpty(preview.Thumbnail.DataURI)
	s.Require().Empty(preview.Favicon.DataURI)
	message := &common.Message{LinkPreviews: []common.LinkPreview{*preview}}
	wire, err := message.ConvertLinkPreviewsToProto()
	s.Require().NoError(err)
	s.Require().Empty(wire[0].ThumbnailPayload)
	s.Require().Empty(wire[0].FaviconPayload)
	received := &common.Message{ChatMessage: &protobuf.ChatMessage{UnfurledLinks: wire}}
	loaded := received.ConvertFromProtoToLinkPreviews(
		func(_, _ string) string { return "https://localhost:1234/banner" },
		nil)
	s.Require().Len(loaded, 1)
	s.Require().Equal(link, loaded[0].URL)
	s.Require().Equal("Unknown thread name", loaded[0].Title)
	s.Require().Equal("Unknown", loaded[0].Description)
	s.Require().Empty(loaded[0].ThreadParentName)
	s.Require().Empty(loaded[0].Thumbnail.URL)
	s.Require().True(preview.ThreadAccessible)
	s.Require().Equal(community.Name(), preview.Description)
	s.Require().Equal("#general", preview.ThreadParentName)
	s.Require().Empty(loaded[0].Favicon.URL)
}
