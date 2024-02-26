package protocol

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

const (
	exampleIdenticonURI = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAYAAAAeP4ixA" +
		"AAAiklEQVR4nOzWwQmFQAwG4ffEXmzLIizDImzLarQBhSwSGH7mO+9hh0DI9AthCI0hNIbQGEJjCI0hNIbQxITM1YfHfl69X3m2bsu/8i5mI" +
		"obQGEJjCI0hNIbQlG+tUW83UtfNFjMRQ2gMofm8tUa3U9c2i5mIITSGqEnMRAyhMYTGEBpDaO4AAAD//5POEGncqtj1AAAAAElFTkSuQmCC"
)

func TestMessengerLinkPreviews(t *testing.T) {
	suite.Run(t, new(MessengerLinkPreviewsTestSuite))
}

type MessengerLinkPreviewsTestSuite struct {
	MessengerBaseTestSuite
}

// StubMatcher should either return an http.Response or nil in case the request
// doesn't match.
type StubMatcher func(req *http.Request) *http.Response

type StubTransport struct {
	// fallbackToDefaultTransport when true will make the transport use
	// http.DefaultTransport in case no matcher is found.
	fallbackToDefaultTransport bool
	// disabledStubs when true, will skip all matchers and use
	// http.DefaultTransport.
	//
	// Useful while testing to toggle between the original and stubbed responses.
	disabledStubs bool
	// matchers are http.RoundTripper functions.
	matchers []StubMatcher
}

// RoundTrip returns a stubbed response if any matcher returns a non-nil
// http.Response. If no matcher is found and fallbackToDefaultTransport is true,
// then it executes the HTTP request using the default http transport.
//
// If StubTransport#disabledStubs is true, the default http transport is used.
func (t *StubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.disabledStubs {
		return http.DefaultTransport.RoundTrip(req)
	}

	for _, matcher := range t.matchers {
		res := matcher(req)
		if res != nil {
			return res, nil
		}
	}

	if t.fallbackToDefaultTransport {
		return http.DefaultTransport.RoundTrip(req)
	}

	return nil, fmt.Errorf("no HTTP matcher found")
}

func (t *StubTransport) AddURLMatcherRoundTrip(urlRegexp string, roundTrip func(r *http.Request) *http.Response) {
	matcher := func(req *http.Request) *http.Response {
		rx, err := regexp.Compile(regexp.QuoteMeta(urlRegexp))
		if err != nil {
			return nil
		}
		if !rx.MatchString(req.URL.String()) {
			return nil
		}
		return roundTrip(req)
	}
	t.matchers = append(t.matchers, matcher)
}

// Add a matcher based on a URL regexp. If a given request URL matches the
// regexp, then responseBody will be returned with a hardcoded 200 status code.
// If headers is non-nil, use it as the value of http.Response.Header.
func (t *StubTransport) AddURLMatcher(urlRegexp string, responseBody []byte, headers map[string]string) {
	matcher := func(req *http.Request) *http.Response {
		res := &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
		}

		if headers != nil {
			res.Header = http.Header{}
			for k, v := range headers {
				res.Header.Set(k, v)
			}
		}

		return res
	}
	t.AddURLMatcherRoundTrip(urlRegexp, matcher)
}

// assertContainsLongString verifies if actual contains a slice of expected and
// correctly prints the cause of the failure. The default behavior of
// require.Contains with long strings is to not print the formatted message
// (varargs to require.Contains).
func (s *MessengerLinkPreviewsTestSuite) assertContainsLongString(expected string, actual string, maxLength int) {
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

func (s *MessengerLinkPreviewsTestSuite) Test_GetLinks() {
	examples := []struct {
		args     string
		expected []string
	}{
		// Invalid URLs are not taken in consideration.
		{args: "", expected: []string{}},
		{args: "  ", expected: []string{}},
		{args: "https", expected: []string{}},
		{args: "https://", expected: []string{}},
		{args: "https://status", expected: []string{}},
		{args: "https://status.", expected: []string{}},
		// URLs must include the sheme.
		{args: "status.com", expected: []string{}},

		{args: "https://status.im", expected: []string{"https://status.im"}},

		// Only the host should be lowercased.
		{args: "HTTPS://STATUS.IM/path/to?Q=AbCdE", expected: []string{"https://status.im/path/to?Q=AbCdE"}},

		// Remove trailing forward slash.
		{args: "https://github.com/", expected: []string{"https://github.com"}},
		{args: "https://www.youtube.com/watch?v=mzOyYtfXkb0/", expected: []string{"https://www.youtube.com/watch?v=mzOyYtfXkb0"}},

		// Valid URL.
		{args: "https://status.c", expected: []string{"https://status.c"}},
		{args: "https://status.im/test", expected: []string{"https://status.im/test"}},
		{args: "https://192.168.0.100:9999/xyz", expected: []string{"https://192.168.0.100:9999/xyz"}},

		// There is a bug in the code that builds the AST from markdown text,
		// because it removes the closing parenthesis, which means it won't be
		// possible to unfurl this URL.
		{args: "https://en.wikipedia.org/wiki/Status_message_(instant_messaging)", expected: []string{"https://en.wikipedia.org/wiki/Status_message_(instant_messaging"}},

		// Multiple URLs.
		{
			args:     "https://status.im/test https://www.youtube.com/watch?v=mzOyYtfXkb0",
			expected: []string{"https://status.im/test", "https://www.youtube.com/watch?v=mzOyYtfXkb0"},
		},
		{
			args:     "status.im https://www.youtube.com/watch?v=mzOyYtfXkb0",
			expected: []string{"https://www.youtube.com/watch?v=mzOyYtfXkb0"},
		},
	}

	for _, ex := range examples {
		links := s.m.GetURLs(ex.args)
		s.Require().Equal(ex.expected, links, "Failed for args: '%s'", ex.args)
	}
}

func (s *MessengerLinkPreviewsTestSuite) readAsset(filename string) []byte {
	b, err := ioutil.ReadFile("../_assets/tests/" + filename)
	s.Require().NoError(err)
	return b
}

func (s *MessengerLinkPreviewsTestSuite) Test_GetFavicon() {
	html := []byte(
		`
	<html>
		<head>
			<link rel="shortcut icon" href="https://www.somehost.com/favicon.png">
		</head>
	</html>`)

	faviconPath := GetFavicon(html)
	s.Require().Equal("https://www.somehost.com/favicon.png", faviconPath)
}

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_YouTube() {
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
	transport.AddURLMatcher(thumbnailURL, s.readAsset("1.jpg"), nil)
	stubbedClient := http.Client{Transport: &transport}

	response, err := s.m.UnfurlURLs(&stubbedClient, []string{u})
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

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_Reddit() {
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

	response, err := s.m.UnfurlURLs(&stubbedClient, []string{u})
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

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_Timeout() {
	httpClient := http.Client{Timeout: time.Nanosecond}
	response, err := s.m.UnfurlURLs(&httpClient, []string{"https://status.im"})
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Empty(response.LinkPreviews)
}

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_CommonFailures() {
	httpClient := http.Client{}

	// Test URL that doesn't return any OpenGraph title.
	transport := StubTransport{}
	transport.AddURLMatcher(
		"https://wikipedia.org",
		[]byte("<html><head></head></html>"),
		nil,
	)
	stubbedClient := http.Client{Transport: &transport}
	response, err := s.m.UnfurlURLs(&stubbedClient, []string{"https://wikipedia.org"})
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Empty(response.LinkPreviews)

	// Test 404.
	response, err = s.m.UnfurlURLs(&httpClient, []string{"https://github.com/status-im/i_do_not_exist"})
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Empty(response.LinkPreviews)

	// Test no response when trying to get OpenGraph metadata.
	response, err = s.m.UnfurlURLs(&httpClient, []string{"https://wikipedia.o"})
	s.Require().NoError(err)
	s.Require().Len(response.StatusLinkPreviews, 0)
	s.Require().Empty(response.LinkPreviews)
}

func (s *MessengerLinkPreviewsTestSuite) Test_isSupportedImageURL() {
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
		s.Require().Equal(e.expected, IsSupportedImageURL(parsedURL), e.url)
	}
}

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_Image() {
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

	transport := StubTransport{}
	// Use a larger image to verify Thumbnail.DataURI is compressed.
	transport.AddURLMatcher(u, s.readAsset("IMG_1205.HEIC.jpg"), nil)
	stubbedClient := http.Client{Transport: &transport}

	response, err := s.m.UnfurlURLs(&stubbedClient, []string{u})
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

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_StatusContactAdded() {
	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)

	c, err := BuildContactFromPublicKey(&identity.PublicKey)
	s.Require().NoError(err)
	s.Require().NotNil(c)

	payload, err := images.GetPayloadFromURI(exampleIdenticonURI)
	s.Require().NoError(err)

	icon := images.IdentityImage{
		Width:   50,
		Height:  50,
		Payload: payload,
	}

	c.Bio = "TestBio_1"
	c.DisplayName = "TestDisplayName_1"
	c.Images = map[string]images.IdentityImage{}
	c.Images[images.SmallDimName] = icon
	s.m.allContacts.Store(c.ID, c)

	// Generate a shared URL
	u, err := s.m.ShareUserURLWithData(c.ID)
	s.Require().NoError(err)

	// Update contact info locally after creating the shared URL
	// This is required to test that URL-decoded data is not used in the preview.
	c.Bio = "TestBio_2"
	c.DisplayName = "TestDisplayName_2"
	s.m.allContacts.Store(c.ID, c)

	r, err := s.m.UnfurlURLs(nil, []string{u})
	s.Require().NoError(err)
	s.Require().Len(r.StatusLinkPreviews, 1)
	s.Require().Len(r.LinkPreviews, 0)

	preview := r.StatusLinkPreviews[0]
	s.Require().Equal(u, preview.URL)
	s.Require().Nil(preview.Community)
	s.Require().Nil(preview.Channel)
	s.Require().NotNil(preview.Contact)
	s.Require().Equal(c.ID, preview.Contact.PublicKey)
	s.Require().Equal(c.DisplayName, preview.Contact.DisplayName)
	s.Require().Equal(c.Bio, preview.Contact.Description)
	s.Require().Equal(icon.Width, preview.Contact.Icon.Width)
	s.Require().Equal(icon.Height, preview.Contact.Icon.Height)
	s.Require().Equal("", preview.Contact.Icon.URL)

	expectedDataURI, err := images.GetPayloadDataURI(icon.Payload)
	s.Require().NoError(err)
	s.Require().Equal(expectedDataURI, preview.Contact.Icon.DataURI)
}

func (s *MessengerLinkPreviewsTestSuite) setProfileParameters(messenger *Messenger, displayName string, bio string, identityImages []images.IdentityImage) {
	const timeout = 1 * time.Second

	changes := SelfContactChangeEvent{}

	SetSettingsAndWaitForChange(&s.Suite, messenger, timeout, func() {
		err := messenger.SetDisplayName(displayName)
		s.Require().NoError(err)
		err = messenger.SetBio(bio)
		s.Require().NoError(err)
	}, func(event *SelfContactChangeEvent) bool {
		if event.DisplayNameChanged {
			changes.DisplayNameChanged = true
		}
		if event.BioChanged {
			changes.BioChanged = true
		}
		return changes.DisplayNameChanged && changes.BioChanged
	})

	SetIdentityImagesAndWaitForChange(&s.Suite, messenger, timeout, func() {
		err := messenger.multiAccounts.StoreIdentityImages(messenger.account.KeyUID, identityImages, false)
		s.Require().NoError(err)
	})

	selfContact := messenger.GetSelfContact()
	s.Require().Equal(selfContact.DisplayName, displayName)
	s.Require().Equal(selfContact.Bio, bio)

	for _, image := range identityImages {
		saved, ok := selfContact.Images[image.Name]
		s.Require().True(ok)
		s.Require().Equal(saved, image)
	}
	s.Require().Equal(selfContact.DisplayName, displayName)
}

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_SelfLink() {
	profileKp := accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = s.m.account.KeyUID
	profileKp.Accounts[0].KeyUID = s.m.account.KeyUID

	err := s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	// Set initial profile parameters
	identityImages := images.SampleIdentityImages()
	s.setProfileParameters(s.m, "TestDisplayName_3", "TestBio_3", identityImages)

	// Generate a shared URL
	u, err := s.m.ShareUserURLWithData(s.m.IdentityPublicKeyString())
	s.Require().NoError(err)

	// Update contact info locally after creating the shared URL
	// This is required to test that URL-decoded data is not used in the preview.
	iconPayload, err := images.GetPayloadFromURI(exampleIdenticonURI)
	s.Require().NoError(err)
	icon := images.IdentityImage{
		Name:    images.SmallDimName,
		Width:   50,
		Height:  50,
		Payload: iconPayload,
	}
	s.setProfileParameters(s.m, "TestDisplayName_4", "TestBio_4", []images.IdentityImage{icon})

	r, err := s.m.UnfurlURLs(nil, []string{u})
	s.Require().NoError(err)
	s.Require().Len(r.StatusLinkPreviews, 1)
	s.Require().Len(r.LinkPreviews, 0)

	userSettings, err := s.m.getSettings()
	s.Require().NoError(err)

	preview := r.StatusLinkPreviews[0]
	s.Require().Equal(u, preview.URL)
	s.Require().Nil(preview.Community)
	s.Require().Nil(preview.Channel)
	s.Require().NotNil(preview.Contact)
	s.Require().Equal(s.m.IdentityPublicKeyString(), preview.Contact.PublicKey)
	s.Require().Equal(userSettings.DisplayName, preview.Contact.DisplayName)
	s.Require().Equal(userSettings.Bio, preview.Contact.Description)

	s.Require().Equal(icon.Width, preview.Contact.Icon.Width)
	s.Require().Equal(icon.Height, preview.Contact.Icon.Height)
	s.Require().Equal("", preview.Contact.Icon.URL)

	expectedDataURI, err := images.GetPayloadDataURI(icon.Payload)
	s.Require().NoError(err)
	s.Require().Equal(expectedDataURI, preview.Contact.Icon.DataURI)
}

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_StatusCommunityJoined() {

	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "status",
		Description: "status community description",
		Color:       "#123456",
		Image:       "../_assets/tests/status.png", // 256*256 px
		ImageAx:     0,
		ImageAy:     0,
		ImageBx:     256,
		ImageBy:     256,
		Banner: images.CroppedImage{
			ImagePath: "../_assets/tests/IMG_1205.HEIC.jpg", // 2282*3352 px
			X:         0,
			Y:         0,
			Width:     160,
			Height:    90,
		},
	}

	response, err := s.m.CreateCommunity(description, false)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	community := response.Communities()[0]
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
	u, err := s.m.ShareCommunityURLWithData(community.ID())
	s.Require().NoError(err)

	// Unfurl community shared URL
	r, err := s.m.UnfurlURLs(nil, []string{u})
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

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_Settings() {
	// Create website stub
	const ogLink = "https://github.com"
	const statusUserLink = "https://status.app/c#zQ3shYSHp7GoiXaauJMnDcjwU2yNjdzpXLosAWapPS4CFxc11"
	const gifLink = "https://media1.giphy.com/media/lcG3qwtTKSNI2i5vst/giphy.gif"

	linksToUnfurl := []string{ogLink, statusUserLink, gifLink}
	text := strings.Join(linksToUnfurl, " ")

	// Test `AlwaysAsk`

	err := s.m.settings.SaveSettingField(settings.URLUnfurlingMode, settings.URLUnfurlingAlwaysAsk)
	s.Require().NoError(err)

	plan := s.m.GetTextURLsToUnfurl(text)
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
	err = s.m.settings.SaveSettingField(settings.URLUnfurlingMode, settings.URLUnfurlingEnableAll)
	s.Require().NoError(err)

	plan = s.m.GetTextURLsToUnfurl(text)
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
	err = s.m.settings.SaveSettingField(settings.URLUnfurlingMode, settings.URLUnfurlingDisableAll)
	s.Require().NoError(err)

	plan = s.m.GetTextURLsToUnfurl(text)
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

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_Limit() {
	text := "https://www.youtube.com/watch?v=6dkDepLX0rk " +
		"https://www.youtube.com/watch?v=ferZnZ0_rSM " +
		"https://www.youtube.com/watch?v=bdneye4pzMw " +
		"https://www.youtube.com/watch?v=pRERgcQe-fQ " +
		"https://www.youtube.com/watch?v=j82L3pLjb_0 " +
		"https://www.youtube.com/watch?v=hxsJvKYyVyg " +
		"https://www.youtube.com/watch?v=jIIuzB11dsA "

	urls := s.m.GetURLs(text)
	s.Require().Equal(UnfurledLinksPerMessageLimit, len(urls))
}
