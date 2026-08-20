package thirdparty

import "strings"

// Image media types that can carry animation. Everything else under image/ is a
// still frame.
//
// The list errs on the side of keeping media: webp and avif are usually still
// but may animate, and treating them as animated costs at most one unnecessary
// download in the detail view, while treating an animated one as still would
// silently drop the animation.
var animatedImageSubtypes = []string{
	"gif",
	"apng",
	"webp",
	"avif",
	"svg+xml",
}

// IsAnimatedMediaType reports whether media of this type can be animated, and so
// belongs in a collectible's animation URL rather than its image URL. An empty
// or unrecognised type is not animated: a provider that tells us nothing is not
// evidence of animation.
func IsAnimatedMediaType(mediaType string) bool {
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if i := strings.IndexByte(mediaType, ';'); i >= 0 {
		mediaType = strings.TrimSpace(mediaType[:i])
	}

	if strings.HasPrefix(mediaType, "video/") || strings.HasPrefix(mediaType, "audio/") {
		return true
	}

	subtype, found := strings.CutPrefix(mediaType, "image/")
	if !found {
		return false
	}

	for _, animated := range animatedImageSubtypes {
		if subtype == animated {
			return true
		}
	}

	return false
}
