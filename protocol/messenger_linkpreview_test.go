package protocol

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/linkpreview/unfurlers"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestMessengerLinkPreviews(t *testing.T) {
	suite.Run(t, new(MessengerLinkPreviewsTestSuite))
}

type MessengerLinkPreviewsTestSuite struct {
	MessengerBaseTestSuite
}

//func (s *MessengerLinkPreviewsTestSuite) SetupTest() {
//	s.logger = tt.MustCreateTestLogger()
//
//	c := waku.DefaultConfig
//	c.MinimumAcceptedPoW = 0
//	shh := waku.New(&c, s.logger)
//	s.shh = gethbridge.NewGethWakuWrapper(shh)
//	s.Require().NoError(shh.Start())
//
//	s.m = s.newMessenger()
//	s.privateKey = s.m.identity
//	_, err := s.m.Start()
//	s.Require().NoError(err)
//}

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

// Add a matcher based on a URL regexp. If a given request URL matches the
// regexp, then responseBody will be returned with a hardcoded 200 status code.
// If headers is non-nil, use it as the value of http.Response.Header.
func (t *StubTransport) AddURLMatcher(urlRegexp string, responseBody []byte, headers map[string]string) {
	matcher := func(req *http.Request) *http.Response {
		rx, err := regexp.Compile(regexp.QuoteMeta(urlRegexp))
		if err != nil {
			return nil
		}
		if rx.MatchString(req.URL.String()) {
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
		return nil
	}

	t.matchers = append(t.matchers, matcher)
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
		links := GetURLs(ex.args)
		s.Require().Equal(ex.expected, links, "Failed for args: '%s'", ex.args)
	}
}

func (s *MessengerLinkPreviewsTestSuite) readAsset(filename string) []byte {
	b, err := ioutil.ReadFile("../_assets/tests/" + filename)
	s.Require().NoError(err)
	return b
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

	transport := StubTransport{}
	transport.AddURLMatcher(
		u,
		[]byte(fmt.Sprintf(`
			<html>
				<head>
					<meta property="og:title" content="%s">
					<meta property="og:description" content="%s">
					<meta property="og:image" content="%s">
				</head>
			</html>
		`, expected.Title, expected.Description, thumbnailURL)),
		nil,
	)
	transport.AddURLMatcher(thumbnailURL, s.readAsset("1.jpg"), nil)
	stubbedClient := http.Client{Transport: &transport}

	previews, err := s.m.UnfurlURLs(&stubbedClient, []string{u})
	s.Require().NoError(err)
	s.Require().Len(previews, 1)
	preview := previews[0]

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

	previews, err := s.m.UnfurlURLs(&stubbedClient, []string{u})
	s.Require().NoError(err)
	s.Require().Len(previews, 1)
	preview := previews[0]

	s.Require().Equal(expected.Type, preview.Type)
	s.Require().Equal(expected.URL, preview.URL)
	s.Require().Equal(expected.Hostname, preview.Hostname)
	s.Require().Equal(expected.Title, preview.Title)
	s.Require().Equal(expected.Description, preview.Description)
	s.Require().Equal(expected.Thumbnail, preview.Thumbnail)
}

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_Timeout() {
	httpClient := http.Client{Timeout: time.Nanosecond}
	previews, err := s.m.UnfurlURLs(&httpClient, []string{"https://status.im"})
	s.Require().NoError(err)
	s.Require().Empty(previews)
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
	previews, err := s.m.UnfurlURLs(&stubbedClient, []string{"https://wikipedia.org"})
	s.Require().NoError(err)
	s.Require().Empty(previews)

	// Test 404.
	previews, err = s.m.UnfurlURLs(&httpClient, []string{"https://github.com/status-im/i_do_not_exist"})
	s.Require().NoError(err)
	s.Require().Empty(previews)

	// Test no response when trying to get OpenGraph metadata.
	previews, err = s.m.UnfurlURLs(&httpClient, []string{"https://wikipedia.o"})
	s.Require().NoError(err)
	s.Require().Empty(previews)
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
		s.Require().Equal(e.expected, unfurlers.IsSupportedImageURL(parsedURL), e.url)
	}
}

func (s *MessengerLinkPreviewsTestSuite) Test_UnfurlURLs_Image() {
	u := "https://placehold.co/600x400@3x.png"
	expected := common.LinkPreview{
		Type:        protobuf.UnfurledLink_IMAGE,
		URL:         u,
		Hostname:    "placehold.co",
		Title:       "",
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

	previews, err := s.m.UnfurlURLs(&stubbedClient, []string{u})
	s.Require().NoError(err)
	s.Require().Len(previews, 1)
	preview := previews[0]

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