package protocol

import (
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"github.com/status-im/markdown"

	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
)

const UnfurledLinksPerMessageLimit = 5

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

type URLUnfurlPermit int

const (
	URLUnfurlAllowed URLUnfurlPermit = iota
	URLUnfurlAskUser
	URLUnfurlForbiddenBySettings
	URLUnfurlForbiddenByLimit
	URLUnfurlNotSupported
)

type URLUnfurlingMetadata struct {
	permit            URLUnfurlPermit `json:"permit"`
	isStatusSharedURL bool
}

type URLsUnfurlPlan struct {
	urls map[string]URLUnfurlingMetadata
}

func GetURLsToUnfurl(text string) *URLsUnfurlPlan {
	result := &URLsUnfurlPlan{}

	parsedText := markdown.Parse([]byte(text), nil)
	visitor := common.RunLinksVisitor(parsedText)

	//urls := make([]string, 0, len(visitor.Links))
	//indexed := make(map[string]any, len(visitor.Links))

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
		idx = strings.TrimRight(idx, "/") // Removes the spurious trailing forward slash.
		if _, exists := result.urls[idx]; exists {
			continue
		}

		result.urls[idx] = URLUnfurlingMetadata{
			isStatusSharedURL: IsStatusSharedURL(),
		}
		urls = append(urls, idx)

		// This is a temporary limitation solution,
		// should be changed with https://github.com/status-im/status-go/issues/4235
		if len(urls) == UnfurledLinksPerMessageLimit {
			break
		}
	}

	return urls

	return result
}

// Deprecated: GetURLs is deprecated in favor of more generic GetURLsToUnfurl.
//
// This is a wrapper around GetURLsToUnfurl that returns the list of URLs found in the text
// without any additional information.
func GetURLs(text string) []string {
	plan := GetURLsToUnfurl(text)
	urls := make([]string, 0, len(plan.urls))
	for _, url := range plan.urls {
		urls = append(urls, url)
	}
	return urls
}

func NewDefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: DefaultRequestTimeout}
}

// UnfurlURLs assumes clients pass URLs verbatim that were validated and
// processed by GetURLs.
func (m *Messenger) UnfurlURLs(httpClient *http.Client, urls []string) (UnfurlURLsResponse, error) {
	response := UnfurlURLsResponse{}

	s, err := m.getSettings()
	if err != nil {
		return response, fmt.Errorf("failed to get settigs: %w", err)
	}

	// Unfurl in a loop

	response.LinkPreviews = make([]*common.LinkPreview, 0, len(urls))
	response.StatusLinkPreviews = make([]*common.StatusLinkPreview, 0, len(urls))

	if httpClient == nil {
		httpClient = NewDefaultHTTPClient()
	}

	for _, url := range urls {
		m.logger.Debug("unfurling", zap.String("url", url))

		if IsStatusSharedURL(url) {
			unfurler := NewStatusUnfurler(url, m, m.logger)
			preview, err := unfurler.Unfurl()
			if err != nil {
				m.logger.Warn("failed to unfurl status link", zap.String("url", url), zap.Error(err))
				continue
			}
			response.StatusLinkPreviews = append(response.StatusLinkPreviews, preview)
			continue
		}

		// `AlwaysAsk` mode should be handled on the app side
		// and is considered as equal to `EnableAll` in status-go.
		if s.URLUnfurlingMode == settings.URLUnfurlingDisableAll {
			continue
		}

		p, err := m.unfurlURL(httpClient, url)
		if err != nil {
			m.logger.Warn("failed to unfurl", zap.String("url", url), zap.Error(err))
			continue
		}
		response.LinkPreviews = append(response.LinkPreviews, p)
	}

	return response, nil
}
