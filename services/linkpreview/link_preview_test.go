package linkpreview

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/services/linkpreview/unfurlers"
	mock_unfurlers "github.com/status-im/status-go/services/linkpreview/unfurlers/mock"
	"github.com/status-im/status-go/services/sharedurls"
)

const (
	exampleIdenticonURI = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAYAAAAeP4ixA" +
		"AAAiklEQVR4nOzWwQmFQAwG4ffEXmzLIizDImzLarQBhSwSGH7mO+9hh0DI9AthCI0hNIbQGEJjCI0hNIbQxITM1YfHfl69X3m2bsu/8i5mI" +
		"obQGEJjCI0hNIbQlG+tUW83UtfNFjMRQ2gMofm8tUa3U9c2i5mIITSGqEnMRAyhMYTGEBpDaO4AAAD//5POEGncqtj1AAAAAElFTkSuQmCC"
)

func TestLinkPreviews(t *testing.T) {
	suite.Run(t, new(LinkPreviewsTestSuite))
}

type LinkPreviewsTestSuite struct {
	suite.Suite
	logger *zap.Logger

	ctrl *gomock.Controller
}

func (s *LinkPreviewsTestSuite) SetupSuite() {
	var err error
	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)

	s.ctrl = gomock.NewController(s.T())
}

// assertContainsLongString verifies if actual contains a slice of expected and
// correctly prints the cause of the failure. The default behavior of
// require.Contains with long strings is to not print the formatted message
// (varargs to require.Contains).
func (s *LinkPreviewsTestSuite) assertContainsLongString(expected string, actual string, maxLength int) {
	var safeIdx float64
	var actualShort string
	var expectedShort string

	if len(actual) > 0 {
		safeIdx = math.Min(float64(maxLength), float64(len(actual)-1))
		actualShort = actual[:int(safeIdx)]
	}

	if len(expected) > 0 {
		safeIdx = math.Min(float64(maxLength), float64(len(expected)-1))
		expectedShort = expected[:int(safeIdx)]
	}

	s.Require().Contains(
		actual, expected,
		"'%s' should contain '%s'",
		actualShort,
		expectedShort,
	)
}

func (s *LinkPreviewsTestSuite) Test_GetLinks() {
	examples := []struct {
		args     string
		expected *URLsUnfurlPlan
	}{
		// Invalid URLs are not taken in consideration.
		{
			args:     "",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{}},
		},
		{
			args:     "  ",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{}},
		},
		{
			args:     "https",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{}},
		},
		{
			args:     "https://",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{}},
		},
		{
			args:     "https://status",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{}},
		},
		{
			args:     "https://status.",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{}},
		},
		// URLs must include the sheme.
		{
			args:     "status.com",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{}},
		},
		{
			args: "https://status.im",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://status.im",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},
		// Only the host should be lowercased.
		{
			args: "HTTPS://STATUS.IM/path/to?Q=AbCdE",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://status.im/path/to?Q=AbCdE",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},
		// Remove trailing forward slash.
		{
			args: "https://github.com/",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://github.com",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},
		{
			args: "https://www.youtube.com/watch?v=mzOyYtfXkb0/",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://www.youtube.com/watch?v=mzOyYtfXkb0",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},
		// Valid URL.
		{
			args: "https://status.c",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://status.c",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},
		{
			args: "https://status.im/test",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://status.im/test",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},
		{
			args: "https://192.168.0.100:9999/xyz",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://192.168.0.100:9999/xyz",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},

		// There is a bug in the code that builds the AST from markdown text,
		// because it removes the closing parenthesis, which means it won't be
		// possible to unfurl this URL.
		{
			args: "https://en.wikipedia.org/wiki/Status_message_(instant_messaging)",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://en.wikipedia.org/wiki/Status_message_(instant_messaging",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},

		// Multiple URLs.
		{
			args: "https://status.im/test https://www.youtube.com/watch?v=mzOyYtfXkb0",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://status.im/test",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
				{
					URL:               "https://www.youtube.com/watch?v=mzOyYtfXkb0",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},
		{
			args: "status.im https://www.youtube.com/watch?v=mzOyYtfXkb0",
			expected: &URLsUnfurlPlan{URLs: []URLUnfurlingMetadata{
				{
					URL:               "https://www.youtube.com/watch?v=mzOyYtfXkb0",
					Permission:        URLUnfurlingAllowed,
					IsStatusSharedURL: false,
				},
			}},
		},
	}

	for _, ex := range examples {
		s.Run(ex.args, func() {
			links := GetTextURLsToUnfurl(ex.args, settings.URLUnfurlingEnableAll)
			s.Require().Equal(ex.expected, links, "Failed for args: '%s'", ex.args)
		})
	}
}

func (s *LinkPreviewsTestSuite) Test_GetFavicon() {
	goodHTMLPNG := []byte(
		`
	<html>
		<head>
			<link rel="shortcut icon" href="https://www.somehost.com/favicon.png">
		</head>
	</html>`)

	goodHTMLSVG := []byte(
		`
	<html>
		<head>
			<link rel="shortcut icon" href="https://www.somehost.com/favicon.svg">
		</head>
	</html>`)

	goodHTMLICO := []byte(
		`
	<html>
		<head>
			<link rel="shortcut icon" href="https://www.somehost.com/favicon.ico">
		</head>
	</html>`)

	badHTMLNoRelAttr := []byte(
		`
	<html>
		<head>
			<link href="https://www.somehost.com/favicon.png">
		</head>
	</html>`)

	GoodHTMLRelAttributeIcon := []byte(
		`
	<html>
		<head>
			<link rel="icon" href="https://www.somehost.com/favicon.png">
		</head>
	</html>`)

	faviconPath := unfurlers.GetFavicon(goodHTMLPNG)
	s.Require().Equal("https://www.somehost.com/favicon.png", faviconPath)

	faviconPath = unfurlers.GetFavicon(goodHTMLSVG)
	s.Require().Equal("https://www.somehost.com/favicon.svg", faviconPath)

	faviconPath = unfurlers.GetFavicon(goodHTMLICO)
	s.Require().Equal("https://www.somehost.com/favicon.ico", faviconPath)

	faviconPath = unfurlers.GetFavicon(GoodHTMLRelAttributeIcon)
	s.Require().Equal("https://www.somehost.com/favicon.png", faviconPath)

	faviconPath = unfurlers.GetFavicon(badHTMLNoRelAttr)
	s.Require().Equal("", faviconPath)
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_YouTube() {
	u := "https://www.youtube.com/watch?v=lE4UXdJSJM4"
	thumbnailURL := "https://i.ytimg.com/vi/lE4UXdJSJM4/maxresdefault.jpg"
	expected := common.LinkPreview{
		Type:        protobuf.UnfurledLink_LINK,
		URL:         u,
		Hostname:    "www.youtube.com",
		Title:       "Interview with a GNU/Linux user - Partition 1",
		Description: "GNU/Linux Operating SystemInterview with a GNU/Linux user with Richie Guix - aired on © The GNU Linux.Programmer humorLinux humorProgramming jokesProgramming...",
		Thumbnail: common.LinkPreviewThumbnail{
			Width:   1,
			Height:  1,
			DataURI: "data:image/webp;base64,UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAQAaJaQAA3AA/vpMgAA",
		},
	}
	favicon := "https://www.youtube.com/s/desktop/87423d78/img/favicon.ico"
	transport := StubTransport{}
	transport.AddURLMatcher(
		u,
		[]byte(fmt.Sprintf(`
			<html>
				<head>
					<meta property="og:title" content="%s">
					<meta property="og:description" content="%s">
					<meta property="og:image" content="%s">
					<link rel="shortcut icon" href="%s">
				</head>
			</html>
		`, expected.Title, expected.Description, thumbnailURL, favicon)),
		nil,
	)
	thumbnail, err := os.ReadFile("testdata/1.jpg")
	s.Require().NoError(err)

	transport.AddURLMatcher(thumbnailURL, thumbnail, nil)
	stubbedClient := http.Client{Transport: &transport}

	response, err := UnfurlURLs([]string{u}, &stubbedClient, nil, s.logger)
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Len(response.LinkPreviews, 1)
	preview := response.LinkPreviews[0]

	s.Require().Equal(expected.Type, preview.Type)
	s.Require().Equal(expected.URL, preview.URL)
	s.Require().Equal(expected.Hostname, preview.Hostname)
	s.Require().Equal(expected.Title, preview.Title)
	s.Require().Equal(expected.Description, preview.Description)
	s.Require().Equal(expected.Thumbnail.Width, preview.Thumbnail.Width)
	s.Require().Equal(expected.Thumbnail.Height, preview.Thumbnail.Height)
	s.Require().Equal(expected.Thumbnail.URL, preview.Thumbnail.URL)
	s.Require().NotNil(preview.Favicon)
	s.assertContainsLongString(expected.Thumbnail.DataURI, preview.Thumbnail.DataURI, 100)
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_Reddit() {
	u := "https://www.reddit.com/r/Bitcoin/comments/13j0tzr/the_best_bitcoin_explanation_of_all_times/?utm_source=share"
	expected := common.LinkPreview{
		Type:        protobuf.UnfurledLink_LINK,
		URL:         u,
		Hostname:    "www.reddit.com",
		Title:       "The best bitcoin explanation of all times.",
		Description: "",
		Thumbnail:   common.LinkPreviewThumbnail{},
	}

	transport := StubTransport{}
	transport.AddURLMatcher(
		"https://www.reddit.com/oembed",
		[]byte(`
			{
				"provider_url": "https://www.reddit.com/",
				"version": "1.0",
				"title": "The best bitcoin explanation of all times.",
				"provider_name": "reddit",
				"type": "rich",
				"author_name": "DTheDev"
			}
		`),
		nil,
	)
	stubbedClient := http.Client{Transport: &transport}

	response, err := UnfurlURLs([]string{u}, &stubbedClient, nil, s.logger)
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Len(response.LinkPreviews, 1)
	preview := response.LinkPreviews[0]

	s.Require().Equal(expected.Type, preview.Type)
	s.Require().Equal(expected.URL, preview.URL)
	s.Require().Equal(expected.Hostname, preview.Hostname)
	s.Require().Equal(expected.Title, preview.Title)
	s.Require().Equal(expected.Description, preview.Description)
	s.Require().Equal(expected.Thumbnail, preview.Thumbnail)
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_Timeout() {
	httpClient := http.Client{Timeout: time.Nanosecond}
	response, err := UnfurlURLs([]string{"https://status.im"}, &httpClient, nil, s.logger)
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Empty(response.LinkPreviews)
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_CommonFailures() {
	httpClient := http.Client{}

	// Test URL that doesn't return any OpenGraph title.
	transport := StubTransport{}
	transport.AddURLMatcher(
		"https://wikipedia.org",
		[]byte("<html><head></head></html>"),
		nil,
	)
	stubbedClient := http.Client{Transport: &transport}
	response, err := UnfurlURLs([]string{"https://wikipedia.org"}, &stubbedClient, nil, s.logger)
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Empty(response.LinkPreviews)

	// Test 404.
	response, err = UnfurlURLs([]string{"https://github.com/status-im/i_do_not_exist"}, &httpClient, nil, s.logger) // FIXME: Internet access?
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Empty(response.LinkPreviews)

	// Test no response when trying to get OpenGraph metadata.
	response, err = UnfurlURLs([]string{"https://wikipedia.o"}, &httpClient, nil, s.logger) // FIXME: Internet access?// FIXME: Internet access?
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Empty(response.LinkPreviews)
}

func (s *LinkPreviewsTestSuite) Test_isSupportedImageURL() {
	examples := []struct {
		url      string
		expected bool
	}{
		{url: "https://placehold.co/600x400@2x.png", expected: true},
		{url: "https://placehold.co/600x400@2x.PNG", expected: true},
		{url: "https://placehold.co/600x400@2x.jpg", expected: true},
		{url: "https://placehold.co/600x400@2x.JPG", expected: true},
		{url: "https://placehold.co/600x400@2x.jpeg", expected: true},
		{url: "https://placehold.co/600x400@2x.Jpeg", expected: true},
		{url: "https://placehold.co/600x400@2x.webp", expected: true},
		{url: "https://placehold.co/600x400@2x.WebP", expected: true},
		{url: "https://placehold.co/600x400@2x.PnGs", expected: false},
		{url: "https://placehold.co/600x400@2x.tiff", expected: false},
	}

	for _, e := range examples {
		parsedURL, err := url.Parse(e.url)
		s.Require().NoError(err, e)
		s.Require().Equal(e.expected, unfurlers.IsSupportedImageURL(parsedURL), e.url)
	}
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_Image() {
	u := "https://placehold.co/600x400@3x.png"
	expected := common.LinkPreview{
		Type:        protobuf.UnfurledLink_IMAGE,
		URL:         u,
		Hostname:    "placehold.co",
		Title:       "600x400@3x.png",
		Description: "",
		Thumbnail: common.LinkPreviewThumbnail{
			Width:   1293,
			Height:  1900,
			DataURI: "data:image/jpeg;base64,/9j/2wCEABALDA4MChAODQ4SERATGCgaGBYWGDEjJR0oOjM9PDkzODdASFxOQERXRTc4UG1RV19iZ",
		},
	}

	imagePayload, err := os.ReadFile("testdata/IMG_1205.HEIC.jpg")
	s.Require().NoError(err)

	transport := StubTransport{}
	// Use a larger image to verify Thumbnail.DataURI is compressed.
	transport.AddURLMatcher(u, imagePayload, nil)
	stubbedClient := http.Client{Transport: &transport}

	response, err := UnfurlURLs([]string{u}, &stubbedClient, nil, s.logger)
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Len(response.LinkPreviews, 1)
	preview := response.LinkPreviews[0]

	s.Require().Equal(expected.Type, preview.Type)
	s.Require().Equal(expected.URL, preview.URL)
	s.Require().Equal(expected.Hostname, preview.Hostname)
	s.Require().Equal(expected.Title, preview.Title)
	s.Require().Equal(expected.Description, preview.Description)
	s.Require().Equal(expected.Thumbnail.Width, preview.Thumbnail.Width)
	s.Require().Equal(expected.Thumbnail.Height, preview.Thumbnail.Height)
	s.Require().Equal(expected.Thumbnail.URL, preview.Thumbnail.URL)
	s.assertContainsLongString(expected.Thumbnail.DataURI, preview.Thumbnail.DataURI, 100)
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_StatusContactAdded() {
	//publicKey := fake.PublicKey(s.T())
	//c := fake.Contact(s.T(), publicKey)

	var c contacts.Contact
	err := gofakeit.Struct(&c)
	s.Require().NoError(err)

	publicKey, err := c.PublicKey()
	s.Require().NoError(err)

	payload, err := images.GetPayloadFromURI(exampleIdenticonURI)
	s.Require().NoError(err)

	icon := images.IdentityImage{
		Width:   50,
		Height:  50,
		Payload: payload,
	}
	c.Images = map[string]images.IdentityImage{
		images.SmallDimName: icon,
	}

	// Generate a shared URL
	u, err := sharedurls.ShareUserURLWithData(&c)
	s.Require().NoError(err)

	// Provider a different contact with the same ID
	// This is required to test that URL-decoded data is not used in the preview.
	c2 := contacts.Contact{}
	err = gofakeit.Struct(&c2)
	s.Require().NoError(err)

	c2.ID = contacts.ContactIDFromPublicKey(publicKey)
	c2.Images = map[string]images.IdentityImage{
		images.SmallDimName: icon,
	}

	dataProvider := mock_unfurlers.NewMockStatusDataProvider(s.ctrl)
	dataProvider.EXPECT().GetContactByID(gomock.Eq(c2.ID)).Return(&c2, nil).Times(1)

	r, err := UnfurlURLs([]string{u}, nil, dataProvider, s.logger)
	s.Require().NoError(err)
	s.Require().Len(r.StatusLinkPreviews, 1)
	s.Require().Len(r.LinkPreviews, 0)

	preview := r.StatusLinkPreviews[0]
	s.Require().Equal(u, preview.URL)
	s.Require().Nil(preview.Community)
	s.Require().Nil(preview.Channel)
	s.Require().NotNil(preview.Contact)
	s.Require().Equal(c2.ID, preview.Contact.PublicKey)
	s.Require().Equal(c2.DisplayName, preview.Contact.DisplayName)
	s.Require().Equal(c2.Bio, preview.Contact.Description)
	s.Require().Equal(icon.Width, preview.Contact.Icon.Width)
	s.Require().Equal(icon.Height, preview.Contact.Icon.Height)
	s.Require().Equal("", preview.Contact.Icon.URL)

	expectedDataURI, err := images.GetPayloadDataURI(icon.Payload)
	s.Require().NoError(err)
	s.Require().Equal(expectedDataURI, preview.Contact.Icon.DataURI)
}

func (s *LinkPreviewsTestSuite) fakeCommunity() *communities.Community {
	faker := gofakeit.New(0)
	v, err := (&communities.Community{}).Fake(faker)
	s.Require().NoError(err)
	return v.(*communities.Community)
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_StatusCommunityJoined() {
	community := s.fakeCommunity()

	communityImages := community.Images()
	s.Require().Len(communityImages, 3)

	// Get icon data
	icon, ok := communityImages[images.SmallDimName]
	s.Require().True(ok)
	iconWidth, iconHeight, err := images.GetImageDimensions(icon.Payload)
	s.Require().NoError(err)
	iconDataURI, err := images.GetPayloadDataURI(icon.Payload)
	s.Require().NoError(err)

	// Get banner data
	banner, ok := communityImages[images.BannerIdentityName]
	s.Require().True(ok)
	bannerWidth, bannerHeight, err := images.GetImageDimensions(banner.Payload)
	s.Require().NoError(err)
	bannerDataURI, err := images.GetPayloadDataURI(banner.Payload)
	s.Require().NoError(err)

	// Create shared URL
	u, err := sharedurls.ShareCommunityURLWithData(community)
	s.Require().NoError(err)

	// Instantiate provider
	dataProvider := mock_unfurlers.NewMockStatusDataProvider(s.ctrl)
	dataProvider.EXPECT().FetchCommunity(gomock.Eq(community.IDString())).
		Return(community, nil).Times(1)

	// Unfurl community shared URL
	r, err := UnfurlURLs([]string{u}, nil, dataProvider, s.logger)
	s.Require().NoError(err)
	s.Require().Len(r.StatusLinkPreviews, 1)
	s.Require().Len(r.LinkPreviews, 0)

	preview := r.StatusLinkPreviews[0]
	s.Require().Equal(u, preview.URL)
	s.Require().NotNil(preview.Community)
	s.Require().Nil(preview.Channel)
	s.Require().Nil(preview.Contact)

	s.Require().Equal(community.IDString(), preview.Community.CommunityID)
	s.Require().Equal(community.Name(), preview.Community.DisplayName)
	s.Require().Equal(community.Identity().Description, preview.Community.Description)
	s.Require().Equal(iconWidth, preview.Community.Icon.Width)
	s.Require().Equal(iconHeight, preview.Community.Icon.Height)
	s.Require().Equal(iconDataURI, preview.Community.Icon.DataURI)
	s.Require().Equal(bannerWidth, preview.Community.Banner.Width)
	s.Require().Equal(bannerHeight, preview.Community.Banner.Height)
	s.Require().Equal(bannerDataURI, preview.Community.Banner.DataURI)
}

func (s *LinkPreviewsTestSuite) Test_UnfurlURLs_Settings() {
	// Create website stub
	const ogLink = "https://github.com"
	const statusUserLink = "https://status.app/c#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
	const gifLink = "https://media1.giphy.com/media/lcG3qwtTKSNI2i5vst/giphy.gif"

	linksToUnfurl := []string{ogLink, statusUserLink, gifLink}
	text := strings.Join(linksToUnfurl, " ")

	// Test `AlwaysAsk`
	plan := GetTextURLsToUnfurl(text, settings.URLUnfurlingAlwaysAsk)
	s.Require().Len(plan.URLs, len(linksToUnfurl))

	s.Require().Equal(plan.URLs[0].URL, ogLink)
	s.Require().Equal(plan.URLs[0].IsStatusSharedURL, false)
	s.Require().Equal(plan.URLs[0].Permission, URLUnfurlingAskUser)

	s.Require().Equal(plan.URLs[1].URL, statusUserLink)
	s.Require().Equal(plan.URLs[1].IsStatusSharedURL, true)
	s.Require().Equal(plan.URLs[1].Permission, URLUnfurlingAllowed)

	s.Require().Equal(plan.URLs[2].URL, gifLink)
	s.Require().Equal(plan.URLs[2].IsStatusSharedURL, false)
	s.Require().Equal(plan.URLs[2].Permission, URLUnfurlingNotSupported)

	// Test `EnableAll`
	plan = GetTextURLsToUnfurl(text, settings.URLUnfurlingEnableAll)
	s.Require().Len(plan.URLs, len(linksToUnfurl))

	s.Require().Equal(plan.URLs[0].URL, ogLink)
	s.Require().Equal(plan.URLs[0].IsStatusSharedURL, false)
	s.Require().Equal(plan.URLs[0].Permission, URLUnfurlingAllowed)

	s.Require().Equal(plan.URLs[1].URL, statusUserLink)
	s.Require().Equal(plan.URLs[1].IsStatusSharedURL, true)
	s.Require().Equal(plan.URLs[1].Permission, URLUnfurlingAllowed)

	s.Require().Equal(plan.URLs[2].URL, gifLink)
	s.Require().Equal(plan.URLs[2].IsStatusSharedURL, false)
	s.Require().Equal(plan.URLs[2].Permission, URLUnfurlingNotSupported)

	// Test `DisableAll`
	plan = GetTextURLsToUnfurl(text, settings.URLUnfurlingDisableAll)
	s.Require().Len(plan.URLs, len(linksToUnfurl))

	s.Require().Equal(plan.URLs[0].URL, ogLink)
	s.Require().Equal(plan.URLs[0].IsStatusSharedURL, false)
	s.Require().Equal(plan.URLs[0].Permission, URLUnfurlingForbiddenBySettings)

	s.Require().Equal(plan.URLs[1].URL, statusUserLink)
	s.Require().Equal(plan.URLs[1].IsStatusSharedURL, true)
	s.Require().Equal(plan.URLs[1].Permission, URLUnfurlingAllowed)

	s.Require().Equal(plan.URLs[2].URL, gifLink)
	s.Require().Equal(plan.URLs[2].IsStatusSharedURL, false)
	s.Require().Equal(plan.URLs[2].Permission, URLUnfurlingNotSupported)
}
