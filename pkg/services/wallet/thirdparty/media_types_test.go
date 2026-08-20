package thirdparty

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAnimatedMediaType(t *testing.T) {
	testCases := []struct {
		mediaType string
		expected  bool
	}{
		{"video/mp4", true},
		{"video/webm", true},
		{"audio/mpeg", true},
		{"image/gif", true},
		{"image/apng", true},
		{"image/webp", true},
		{"image/avif", true},
		{"image/svg+xml", true},

		{"image/png", false},
		{"image/jpeg", false},
		{"image/jpg", false},
		{"image/bmp", false},
		{"image/tiff", false},

		// A provider that tells us nothing is not evidence of animation.
		{"", false},
		{"application/octet-stream", false},
		{"text/html", false},

		// Real responses carry parameters and inconsistent casing.
		{"IMAGE/GIF", true},
		{"image/gif; charset=binary", true},
		{"  image/png  ", false},
	}

	for _, tc := range testCases {
		t.Run(tc.mediaType, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsAnimatedMediaType(tc.mediaType))
		})
	}
}
