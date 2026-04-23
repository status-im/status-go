package puzzleauth

import (
	"fmt"
	"net/url"
	"strings"
)

// OriginForURL returns scheme://host (host lowercased) for the puzzle auth server, given an RPC or HTTP URL.
func OriginForURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("incomplete url for puzzle origin: %q", raw)
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}
