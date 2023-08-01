package linkpreview

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/common"
)

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
func (t *StubTransport) AddURLMatcher(urlRegexp string, responseBody []byte) {
	matcher := func(req *http.Request) *http.Response {
		rx, err := regexp.Compile(regexp.QuoteMeta(urlRegexp))
		if err != nil {
			return nil
		}
		if rx.MatchString(req.URL.String()) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBuffer(responseBody)),
			}
		}
		return nil
	}

	t.matchers = append(t.matchers, matcher)
}

// assertContainsLongString verifies if actual contains a slice of expected and
// correctly prints the cause of the failure. The default behavior of
// require.Contains with long strings is to not print the formatted message
// (varargs to require.Contains).
func assertContainsLongString(t *testing.T, expected string, actual string, maxLength int) {
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

	require.Contains(
		t,
		actual, expected,
		"'%s' should contain '%s'",
		actualShort,
		expectedShort,
	)
}

func Test_GetLinks(t *testing.T) {
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
		require.Equal(t, ex.expected, links, "Failed for args: '%s'", ex.args)
	}
}

func readAsset(t *testing.T, filename string) []byte {
	b, err := ioutil.ReadFile("../../_assets/tests/" + filename)
	require.NoError(t, err)
	return b
}

func Test_UnfurlURLs_YouTube(t *testing.T) {
	url := "https://www.youtube.com/watch?v=lE4UXdJSJM4"
	thumbnailURL := "https://i.ytimg.com/vi/lE4UXdJSJM4/maxresdefault.jpg"
	expected := common.LinkPreview{
		URL:         url,
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
		url,
		[]byte(fmt.Sprintf(`
			<html>
				<head>
					<meta property="og:title" content="%s">
					<meta property="og:description" content="%s">
					<meta property="og:image" content="%s">
				</head>
			</html>
		`, expected.Title, expected.Description, thumbnailURL)),
	)
	transport.AddURLMatcher(thumbnailURL, readAsset(t, "1.jpg"))
	stubbedClient := http.Client{Transport: &transport}

	previews, err := UnfurlURLs(nil, stubbedClient, []string{url})
	require.NoError(t, err)
	require.Len(t, previews, 1)
	preview := previews[0]

	require.Equal(t, expected.URL, preview.URL)
	require.Equal(t, expected.Hostname, preview.Hostname)
	require.Equal(t, expected.Title, preview.Title)
	require.Equal(t, expected.Description, preview.Description)
	require.Equal(t, expected.Thumbnail.Width, preview.Thumbnail.Width)
	require.Equal(t, expected.Thumbnail.Height, preview.Thumbnail.Height)
	require.Equal(t, expected.Thumbnail.URL, preview.Thumbnail.URL)
	assertContainsLongString(t, expected.Thumbnail.DataURI, preview.Thumbnail.DataURI, 100)
}

func Test_UnfurlURLs_giphy(t *testing.T) {
	url := "https://www.giphy.com/stickers/happyplaceshow-transparent-sof2kXOSK5beJdb7xH"

	expected := common.LinkPreview{
		URL:      url,
		Hostname: "www.giphy.com",
		Title:    "Floating Tv Show Sticker by Happy Place for iOS & Android | GIPHY",
		Thumbnail: common.LinkPreviewThumbnail{
			URL: "https://media4.giphy.com/media/sof2kXOSK5beJdb7xH/giphy.gif",
			Height: 480,
			Width: 400,
		},
	}

	transport := StubTransport{}
	transport.AddURLMatcher(
		"https://giphy.com/services/oembed",
		[]byte(`
		{
			"title": "Floating Tv Show Sticker by Happy Place for iOS & Android | GIPHY",
			"url": "https://media4.giphy.com/media/sof2kXOSK5beJdb7xH/giphy.gif",
			"height": 480,
			"width": 400,
			"author_name": "Happy Place",
			"author_url": "https://giphy.com/happyplaceshow",
			"provider_name": "GIPHY",
			"provider_url": "https://giphy.com/",
			"type": "photo"
		}
		`),
	)

	stubbedClient := http.Client{Transport: &transport}

	previews, err := UnfurlURLs(nil, stubbedClient, []string{url})
	require.NoError(t, err)
	require.Len(t, previews, 1)
	preview := previews[0]

	require.Equal(t, expected.URL, preview.URL)
	require.Equal(t, expected.Hostname, preview.Hostname)
	require.Equal(t, expected.Title, preview.Title)
	require.Equal(t, expected.Thumbnail.Width, preview.Thumbnail.Width)
	require.Equal(t, expected.Thumbnail.Height, preview.Thumbnail.Height)
	require.Equal(t, expected.Thumbnail.URL, preview.Thumbnail.URL)
}

func Test_UnfurlURLs_Reddit(t *testing.T) {
	url := "https://www.reddit.com/r/Bitcoin/comments/13j0tzr/the_best_bitcoin_explanation_of_all_times/?utm_source=share"
	expected := common.LinkPreview{
		URL:         url,
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
	)
	stubbedClient := http.Client{Transport: &transport}

	previews, err := UnfurlURLs(nil, stubbedClient, []string{url})
	require.NoError(t, err)
	require.Len(t, previews, 1)
	preview := previews[0]

	require.Equal(t, expected.URL, preview.URL)
	require.Equal(t, expected.Hostname, preview.Hostname)
	require.Equal(t, expected.Title, preview.Title)
	require.Equal(t, expected.Description, preview.Description)
	require.Equal(t, expected.Thumbnail, preview.Thumbnail)
}

func Test_UnfurlURLs_Timeout(t *testing.T) {
	httpClient := http.Client{Timeout: time.Nanosecond}
	previews, err := UnfurlURLs(nil, httpClient, []string{"https://status.im"})
	require.NoError(t, err)
	require.Empty(t, previews)
}

func Test_UnfurlURLs_CommonFailures(t *testing.T) {
	httpClient := http.Client{}

	// Test URL that doesn't return any OpenGraph title.
	transport := StubTransport{}
	transport.AddURLMatcher(
		"https://wikipedia.org",
		[]byte("<html><head></head></html>"),
	)
	stubbedClient := http.Client{Transport: &transport}
	previews, err := UnfurlURLs(nil, stubbedClient, []string{"https://wikipedia.org"})
	require.NoError(t, err)
	require.Empty(t, previews)

	// Test 404.
	previews, err = UnfurlURLs(nil, httpClient, []string{"https://github.com/status-im/i_do_not_exist"})
	require.NoError(t, err)
	require.Empty(t, previews)

	// Test no response when trying to get OpenGraph metadata.
	previews, err = UnfurlURLs(nil, httpClient, []string{"https://wikipedia.o"})
	require.NoError(t, err)
	require.Empty(t, previews)
}
