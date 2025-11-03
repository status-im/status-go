package linkpreview

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
	"github.com/status-im/status-go/services/sharedurls"
)

func URLUnfurlingSupported(url string) bool {
	return !strings.HasSuffix(url, ".gif")
}

func normalizeHostname(hostname string) string {
	hostname = strings.ToLower(hostname)
	re := regexp.MustCompile(`^www\.(.*)$`)
	return re.ReplaceAllString(hostname, "$1")
}

func newURLUnfurler(httpClient *http.Client, url *neturl.URL, logger *zap.Logger) Unfurler {
	if IsSupportedImageURL(url) {
		return NewImageUnfurler(url, logger, httpClient)
	}

	switch normalizeHostname(url.Hostname()) {
	case "reddit.com":
		return NewOEmbedUnfurler("https://www.reddit.com/oembed", url, logger, httpClient)
	default:
		return NewOpenGraphUnfurler(url, logger, httpClient)
	}
}

func UnfurlURL(url string, httpClient *http.Client, logger *zap.Logger) (*common.LinkPreview, error) {
	preview := new(common.LinkPreview)

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return preview, err
	}

	unfurler := newURLUnfurler(httpClient, parsedURL, logger)
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

func GetTextURLsToUnfurl(text string, URLUnfurlingMode settings.URLUnfurlingModeType) *URLsUnfurlPlan {
	indexedUrls := map[string]struct{}{}
	result := &URLsUnfurlPlan{
		// The usage of `UnfurledLinksPerMessageLimit` is quite random here. I wanted to allocate
		// some not-zero place here, using the limit number is at least some binding.
		URLs: make([]URLUnfurlingMetadata, 0, UnfurledLinksPerMessageLimit),
	}
	parsedText := markdown.Parse([]byte(text), nil)
	visitor := common.RunLinksVisitor(parsedText)

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

		url := parsedURL.String()
		url = strings.TrimRight(url, "/") // Removes the spurious trailing forward slash.
		if _, exists := indexedUrls[url]; exists {
			continue
		}

		metadata := URLUnfurlingMetadata{
			URL:               url,
			IsStatusSharedURL: sharedurls.IsStatusSharedURL(url),
		}

		if !URLUnfurlingSupported(rawURL) {
			metadata.Permission = URLUnfurlingNotSupported
		} else if metadata.IsStatusSharedURL {
			metadata.Permission = URLUnfurlingAllowed
		} else {
			switch URLUnfurlingMode {
			case settings.URLUnfurlingAlwaysAsk:
				metadata.Permission = URLUnfurlingAskUser
			case settings.URLUnfurlingEnableAll:
				metadata.Permission = URLUnfurlingAllowed
			case settings.URLUnfurlingDisableAll:
				metadata.Permission = URLUnfurlingForbiddenBySettings
			default:
				metadata.Permission = URLUnfurlingForbiddenBySettings
			}
		}

		result.URLs = append(result.URLs, metadata)
	}

	return result
}

func NewDefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: DefaultRequestTimeout}
}

// UnfurlURLs assumes clients pass URLs verbatim that were validated and
// processed by GetTextURLsToUnfurl.
func UnfurlURLs(urls []string, httpClient *http.Client, statusDataProvider StatusDataProvider, logger *zap.Logger) (UnfurlURLsResponse, error) {
	response := UnfurlURLsResponse{}

	// Unfurl in a loop

	response.LinkPreviews = make([]*common.LinkPreview, 0, len(urls))
	response.StatusLinkPreviews = make([]*common.StatusLinkPreview, 0, len(urls))

	if httpClient == nil {
		httpClient = NewDefaultHTTPClient()
	}

	for _, url := range urls {
		logger.Debug("unfurling", zap.String("url", url))

		if sharedurls.IsStatusSharedURL(url) {
			unfurler := NewStatusUnfurler(url, statusDataProvider, logger)
			preview, err := unfurler.Unfurl()
			if err != nil {
				logger.Warn("failed to unfurl status link", zap.String("url", url), zap.Error(err))
				continue
			}
			response.StatusLinkPreviews = append(response.StatusLinkPreviews, preview)
			continue
		}

		p, err := UnfurlURL(url, httpClient, logger)
		if err != nil {
			logger.Warn("failed to unfurl", zap.String("url", url), zap.Error(err))
			continue
		}
		response.LinkPreviews = append(response.LinkPreviews, p)
	}

	return response, nil
}
