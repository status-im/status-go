package protocol

import (
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"

	"github.com/status-im/status-go/multiaccounts/settings"

	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"github.com/status-im/markdown"

	"github.com/status-im/status-go/protocol/common"
)

type UnfurlURLsResponse struct {
	LinkPreviews       []*common.LinkPreview       `json:"linkPreviews,omitempty"`
	StatusLinkPreviews []*common.StatusLinkPreview `json:"statusLinkPreviews,omitempty"`
}

func normalizeHostname(hostname string) string {
	hostname = strings.ToLower(hostname)
	re := regexp.MustCompile(`^www\.(.*)$`)
	return re.ReplaceAllString(hostname, "$1")
}

func (m *Messenger) newURLUnfurler(httpClient *http.Client, url *neturl.URL) Unfurler {

	if IsSupportedImageURL(url) {
		return NewImageUnfurler(
			url,
			m.logger,
			httpClient)
	}

	switch normalizeHostname(url.Hostname()) {
	case "reddit.com":
		return NewOEmbedUnfurler(
			"https://www.reddit.com/oembed",
			url,
			m.logger,
			httpClient)
	default:
		return NewOpenGraphUnfurler(
			url,
			m.logger,
			httpClient)
	}
}

func (m *Messenger) unfurlURL(httpClient *http.Client, url string) (*common.LinkPreview, error) {
	preview := new(common.LinkPreview)

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return preview, err
	}

	unfurler := m.newURLUnfurler(httpClient, parsedURL)
	preview, err = unfurler.Unfurl()
	if err != nil {
		return preview, err
	}
	preview.Hostname = strings.ToLower(parsedURL.Hostname())

	return preview, nil
}

// parseValidURL is a stricter version of url.Parse that performs additional
// checks to ensure the URL is valid for clients to request a link preview.
func parseValidURL(rawURL string) (*neturl.URL, error) {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL failed: %w", err)
	}

	if u.Scheme == "" {
		return nil, errors.New("missing URL scheme")
	}

	_, err = publicsuffix.EffectiveTLDPlusOne(u.Hostname())
	if err != nil {
		return nil, fmt.Errorf("missing known URL domain: %w", err)
	}

	return u, nil
}

// GetURLs returns only what we consider unfurleable URLs.
//
// If we wanted to be extra precise and help improve UX, we could ignore URLs
// that we know can't be unfurled. This is at least possible with the oEmbed
// protocol because providers must specify an endpoint scheme.
func GetURLs(text string) []string {
	parsedText := markdown.Parse([]byte(text), nil)
	visitor := common.RunLinksVisitor(parsedText)

	urls := make([]string, 0, len(visitor.Links))
	indexed := make(map[string]any, len(visitor.Links))

	for _, rawURL := range visitor.Links {
		parsedURL, err := parseValidURL(rawURL)
		if err != nil {
			continue
		}
		// Lowercase the host so the URL can be used as a cache key. Particularly on
		// mobile clients it is common that the first character in a text input is
		// automatically uppercased. In WhatsApp they incorrectly lowercase the
		// URL's path, but this is incorrect. For instance, some URL shorteners are
		// case-sensitive, some websites encode base64 in the path, etc.
		parsedURL.Host = strings.ToLower(parsedURL.Host)

		idx := parsedURL.String()
		// Removes the spurious trailing forward slash.
		idx = strings.TrimRight(idx, "/")
		if _, exists := indexed[idx]; exists {
			continue
		} else {
			indexed[idx] = nil
			urls = append(urls, idx)
		}
	}

	return urls
}

func NewDefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: DefaultRequestTimeout}
}

// UnfurlURLs assumes clients pass URLs verbatim that were validated and
// processed by GetURLs.
func (m *Messenger) UnfurlURLs(httpClient *http.Client, urls []string) (*UnfurlURLsResponse, error) {
	if httpClient == nil {
		httpClient = NewDefaultHTTPClient()
	}

	s, err := m.getSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settigs: %w", err)
	}

	// We use switch case, though there's most cases are empty for code clarity.
	switch s.UrlUnfurlingMode {
	case settings.UrlUnfurlingDisableAll:
		return nil, fmt.Errorf("url unfurling is disabled") // WARNING: Is this an error?
	case settings.UrlUnfurlingEnableAll:
		break
	case settings.UrlUnfurlingAlwaysAsk:
		// This mode should be handled on the app side
		// and is considered as equal to UrlUnfurlingEnableAll in status-go.
		break
	case settings.UrlUnfurlingSpecifyForSite:
		// TODO: implement filtering post-mvp
	default:
		return nil, fmt.Errorf("invalid url unfurling mode setting: %d", s.UrlUnfurlingMode)
	}

	// Unfurl in a loop

	r := UnfurlURLsResponse{
		LinkPreviews:       make([]*common.LinkPreview, 0, len(urls)),
		StatusLinkPreviews: make([]*common.StatusLinkPreview, 0, len(urls)),
	}

	for _, url := range urls {
		m.logger.Debug("unfurling", zap.String("url", url))

		if m.IsStatusSharedURL(url) {
			unfurler := NewStatusUnfurler(url, m, m.logger)
			preview, err := unfurler.Unfurl()
			if err != nil {
				m.logger.Warn("failed to unfurl status link", zap.String("url", url), zap.Error(err))
				continue
			}
			r.StatusLinkPreviews = append(r.StatusLinkPreviews, preview)
			continue
		}

		p, err := m.unfurlURL(httpClient, url)
		if err != nil {
			m.logger.Warn("failed to unfurl", zap.String("url", url), zap.Error(err))
			continue
		}
		r.LinkPreviews = append(r.LinkPreviews, p)
	}

	return &r, nil
}
