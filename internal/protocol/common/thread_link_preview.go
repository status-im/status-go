package common

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseStatusThreadParentURL normalizes the existing parent routes without
// resolving identities or fetching any data.
func ParseStatusThreadParentURL(rawURL string) (*url.URL, error) {
	const baseURL = "https://status.app"
	rawURL = strings.TrimSpace(rawURL)
	for _, prefix := range []string{baseURL + "/", "http://status.app/", "status-app://"} {
		if !strings.HasPrefix(rawURL, prefix) {
			continue
		}
		parent, err := url.Parse(baseURL + "/" + strings.TrimPrefix(rawURL, prefix))
		if err != nil {
			return nil, err
		}
		for _, path := range []string{"/cc/", "/p/", "/g/"} {
			if strings.HasPrefix(parent.Path, path) {
				return parent, nil
			}
		}
		break
	}
	return nil, fmt.Errorf("not a status thread url")
}

// IsStatusThreadURL also recognizes malformed destinations so their metadata
// cannot bypass privacy handling by falling through to an ordinary preview.
func IsStatusThreadURL(rawURL string) bool {
	parent, err := ParseStatusThreadParentURL(rawURL)
	if err != nil {
		return false
	}
	for _, pair := range strings.Split(parent.RawQuery, "&") {
		key, _, _ := strings.Cut(pair, "=")
		if key, err := url.QueryUnescape(key); err == nil && key == "thread-id" {
			return true
		}
	}
	return false
}

// NewUnknownThreadPreview is safe to send to recipients with unknown access.
// Authorized clients replace it using locally resolved metadata.
func NewUnknownThreadPreview(rawURL string) *LinkPreview {
	return &LinkPreview{
		URL: rawURL, Hostname: "status.app", Title: "Unknown thread name",
		Description: "Unknown", ThreadParentName: "Unknown",
	}
}
