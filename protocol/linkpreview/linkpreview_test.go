package linkpreview

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/common"
)

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

func TestGetLinks(t *testing.T) {
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

func TestUnfurlURLs(t *testing.T) {
	examples := []struct {
		url      string
		expected common.LinkPreview
	}{
		{
			url: "https://github.com/",
			expected: common.LinkPreview{
				Description: "GitHub is where over 100 million developers shape the future of software, together. Contribute to the open source community, manage your Git repositories, review code like a pro, track bugs and fea...",
				Hostname:    "github.com",
				Title:       "GitHub: Let’s build from here",
				URL:         "https://github.com/",
				Thumbnail: common.LinkPreviewThumbnail{
					Width:   1200,
					Height:  630,
					URL:     "",
					DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAABLAAAAJ2CAMAAAB4",
				},
			},
		},
		{
			url: "https://github.com/status-im/status-mobile/issues/15469",
			expected: common.LinkPreview{
				Description: "Designs https://www.figma.com/file/wA8Epdki2OWa8Vr067PCNQ/Composer-for-Mobile?node-id=2102-232933&t=tTYKjMpICnzwF5Zv-0 Out of scope Enable link previews (we can assume for now that is always on) Mu...",
				Hostname:    "github.com",
				Title:       "Allow users to customize links · Issue #15469 · status-im/status-mobile",
				URL:         "https://github.com/status-im/status-mobile/issues/15469",
				Thumbnail: common.LinkPreviewThumbnail{
					Width:   1200,
					Height:  600,
					URL:     "",
					DataURI: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAABLAAAAJYCAYAAABy",
				},
			},
		},
		{
			url: "https://www.imdb.com/title/tt0117500/",
			expected: common.LinkPreview{
				Description: "The Rock: Directed by Michael Bay. With Sean Connery, Nicolas Cage, Ed Harris, John Spencer. A mild-mannered chemist and an ex-con must lead the counterstrike when a rogue group of military men, led by a renegade general, threaten a nerve gas attack from Alcatraz against San Francisco.",
				Hostname:    "www.imdb.com",
				Title:       "The Rock (1996) - IMDb",
				URL:         "https://www.imdb.com/title/tt0117500/",
				Thumbnail: common.LinkPreviewThumbnail{
					Width:   1000,
					Height:  1481,
					URL:     "",
					DataURI: "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wCEAAkJCgg",
				},
			},
		},
		{
			url: "https://www.youtube.com/watch?v=lE4UXdJSJM4",
			expected: common.LinkPreview{
				URL:         "https://www.youtube.com/watch?v=lE4UXdJSJM4",
				Hostname:    "www.youtube.com",
				Title:       "Interview with a GNU/Linux user - Partition 1",
				Description: "GNU/Linux Operating SystemInterview with a GNU/Linux user with Richie Guix - aired on © The GNU Linux.Programmer humorLinux humorProgramming jokesProgramming...",
				Thumbnail: common.LinkPreviewThumbnail{
					Width:   1280,
					Height:  720,
					DataURI: "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wCEAAUDBA8",
				},
			},
		},
	}

	var urls []string
	for _, e := range examples {
		urls = append(urls, e.url)
	}

	links, err := UnfurlURLs(nil, urls)
	require.NoError(t, err)
	require.Len(t, links, len(examples), "all URLs should have been unfurled successfully")

	for i, link := range links {
		e := examples[i]
		require.Equal(t, e.expected.URL, link.URL, e.url)
		require.Equal(t, e.expected.Hostname, link.Hostname, e.url)
		require.Equal(t, e.expected.Title, link.Title, e.url)
		require.Equal(t, e.expected.Description, link.Description, e.url)

		require.Equal(t, e.expected.Thumbnail.Width, link.Thumbnail.Width, e.url)
		require.Equal(t, e.expected.Thumbnail.Height, link.Thumbnail.Height, e.url)
		require.Equal(t, e.expected.Thumbnail.URL, link.Thumbnail.URL, e.url)
		assertContainsLongString(t, e.expected.Thumbnail.DataURI, link.Thumbnail.DataURI, 100)
	}

	// Test URL that doesn't return any OpenGraph title.
	previews, err := UnfurlURLs(nil, []string{"https://wikipedia.org"})
	require.NoError(t, err)
	require.Empty(t, previews)

	// Test 404.
	previews, err = UnfurlURLs(nil, []string{"https://github.com/status-im/i_do_not_exist"})
	require.NoError(t, err)
	require.Empty(t, previews)

	// Test no response when trying to get OpenGraph metadata.
	previews, err = UnfurlURLs(nil, []string{"https://wikipedia.o"})
	require.NoError(t, err)
	require.Empty(t, previews)
}
