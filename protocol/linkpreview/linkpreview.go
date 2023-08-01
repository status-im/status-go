package linkpreview

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/keighl/metabolize"
	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"github.com/status-im/markdown"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
)

type LinkPreview struct {
	common.LinkPreview
}

type Unfurler interface {
	unfurl() (common.LinkPreview, error)
}

type Headers map[string]string

const (
	defaultRequestTimeout = 15000 * time.Millisecond

	headerAcceptJSON = "application/json; charset=utf-8"
	headerAcceptText = "text/html; charset=utf-8"

	// Without a particular user agent, many providers treat status-go as a
	// gluttony bot, and either respond more frequently with a 429 (Too Many
	// Requests), or simply refuse to return valid data. Note that using a known
	// browser UA doesn't work well with some providers, such as Spotify,
	// apparently they still flag status-go as a bad actor.
	headerUserAgent = "status-go/v0.151.15"

	// Currently set to English, but we could make this setting dynamic according
	// to the user's language of choice.
	headerAcceptLanguage = "en-US,en;q=0.5"
)

func fetchBody(logger *zap.Logger, httpClient http.Client, url string, headers Headers) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			logger.Error("failed to close response body", zap.Error(err))
		}
	}()

	if res.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("http request failed, statusCode='%d'", res.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body bytes: %w", err)
	}

	return bodyBytes, nil
}

func newDefaultLinkPreview(url *neturl.URL) common.LinkPreview {
	return common.LinkPreview{
		URL:      url.String(),
		Hostname: url.Hostname(),
	}
}

func fetchThumbnail(logger *zap.Logger, httpClient http.Client, url string) (common.LinkPreviewThumbnail, error) {
	var thumbnail common.LinkPreviewThumbnail

	imgBytes, err := fetchBody(logger, httpClient, url, nil)
	if err != nil {
		return thumbnail, fmt.Errorf("could not fetch thumbnail: %w", err)
	}

	width, height, err := images.GetImageDimensions(imgBytes)
	if err != nil {
		return thumbnail, fmt.Errorf("could not get image dimensions: %w", err)
	}
	thumbnail.Width = width
	thumbnail.Height = height

	dataURI, err := images.GetPayloadDataURI(imgBytes)
	if err != nil {
		return thumbnail, fmt.Errorf("could not build data URI: %w", err)
	}
	thumbnail.DataURI = dataURI

	return thumbnail, nil
}

type OEmbedUnfurler struct {
	logger     *zap.Logger
	httpClient http.Client
	// oembedEndpoint describes where the consumer may request representations for
	// the supported URL scheme. For example, for YouTube, it is
	// https://www.youtube.com/oembed.
	oembedEndpoint string
	// url is the actual URL to be unfurled.
	url *neturl.URL
}

type OEmbedBaseResponse struct {
	Type            string `json:"type"`
	Version         string `json:"version"`
	Title           string `json:"title,omitempty"`
	AuthorName      string `json:"author_name,omitempty"`
	AuthorURL       string `json:"author_url,omitempty"`
	ProviderName    string `json:"provider_name,omitempty"`
	ProviderURL     string `json:"provider_url,omitempty"`
	ThumbnailURL    string `json:"thumbnail_url,omitempty"`
	URL             string `json:"url,omitempty"`
	HTML            string `json:"html,omitempty"`
	CacheAge        int    `json:"cache_age,omitempty"`
	ThumbnailWidth  int    `json:"thumbnail_width,omitempty"`
	ThumbnailHeight int    `json:"thumbnail_height,omitempty"`
	Width           int    `json:"width,omitempty"`
	Height          int    `json:"height,omitempty"`
}

func (u OEmbedUnfurler) newOEmbedURL() (*neturl.URL, error) {
	oembedURL, err := neturl.Parse(u.oembedEndpoint)
	if err != nil {
		return nil, err
	}

	// When format is specified, the provider MUST return data in the requested
	// format, else return an error.
	oembedURL.RawQuery = neturl.Values{
		"url":    {u.url.String()},
		"format": {"json"},
	}.Encode()

	return oembedURL, nil
}

func handlePhotoOembedType(preview *common.LinkPreview, response OEmbedBaseResponse) {
	if response.URL != "" {
		preview.Thumbnail.URL = response.URL
	}
}

func handleVideoOembedType(preview *common.LinkPreview, response OEmbedBaseResponse) {
	if response.ThumbnailURL != "" {
		preview.Thumbnail.URL = response.ThumbnailURL
	}
	if response.Width != 0 {
		preview.Thumbnail.Width = response.ThumbnailWidth
	}
	if response.Height != 0 {
		preview.Thumbnail.Height = response.ThumbnailHeight
	}
}

func handleRichOembedType(preview *common.LinkPreview, response OEmbedBaseResponse) {
	if response.ThumbnailURL != "" {
		preview.Thumbnail.URL = response.ThumbnailURL
	}
	if response.Width != 0 {
		preview.Thumbnail.Width = response.Width
	}
	if response.Height != 0 {
		preview.Thumbnail.Height = response.Height
	}
}

func (u OEmbedUnfurler) unfurl() (common.LinkPreview, error) {
	preview := newDefaultLinkPreview(u.url)
	oembedURL, err := u.newOEmbedURL()
	if err != nil {
		return preview, err
	}

	headers := map[string]string{
		"accept":          headerAcceptJSON,
		"accept-language": headerAcceptLanguage,
		"user-agent":      headerUserAgent,
	}

	oembedBytes, err := fetchBody(u.logger, u.httpClient, oembedURL.String(), headers)
	if err != nil {
		return preview, err
	}

	var oembedResponse OEmbedBaseResponse

	err = json.Unmarshal(oembedBytes, &oembedResponse)
	if err != nil {
		return preview, err
	}

	if oembedResponse.Title != "" {
		preview.Title = oembedResponse.Title
	}

	switch oembedResponse.Type {
	case "photo":
		handlePhotoOembedType(&preview, oembedResponse)

	case "video":
		handleVideoOembedType(&preview, oembedResponse)

	case "rich":
		handleRichOembedType(&preview, oembedResponse)
	default:
		return preview, fmt.Errorf("unexpected oembed type: %v", oembedResponse.Type)
	}
	return preview, nil
}

type OpenGraphMetadata struct {
	Title        string `json:"title" meta:"og:title"`
	Description  string `json:"description" meta:"og:description"`
	ThumbnailURL string `json:"thumbnailUrl" meta:"og:image"`
}

// OpenGraphUnfurler should be preferred over OEmbedUnfurler because oEmbed
// gives back a JSON response with a "html" field that's supposed to be embedded
// in an iframe (hardly useful for existing Status' clients).
type OpenGraphUnfurler struct {
	url        *neturl.URL
	logger     *zap.Logger
	httpClient http.Client
}

func (u OpenGraphUnfurler) unfurl() (common.LinkPreview, error) {
	preview := newDefaultLinkPreview(u.url)

	headers := map[string]string{
		"accept":          headerAcceptText,
		"accept-language": headerAcceptLanguage,
		"user-agent":      headerUserAgent,
	}
	bodyBytes, err := fetchBody(u.logger, u.httpClient, u.url.String(), headers)
	if err != nil {
		return preview, err
	}

	var ogMetadata OpenGraphMetadata
	err = metabolize.Metabolize(ioutil.NopCloser(bytes.NewBuffer(bodyBytes)), &ogMetadata)
	if err != nil {
		return preview, fmt.Errorf("failed to parse OpenGraph data")
	}

	// There are URLs like https://wikipedia.org/ that don't have an OpenGraph
	// title tag, but article pages do. In the future, we can fallback to the
	// website's title by using the <title> tag.
	if ogMetadata.Title == "" {
		return preview, fmt.Errorf("missing required title in OpenGraph response")
	}

	if ogMetadata.ThumbnailURL != "" {
		t, err := fetchThumbnail(u.logger, u.httpClient, ogMetadata.ThumbnailURL)
		if err != nil {
			// Given we want to fetch thumbnails on a best-effort basis, if an error
			// happens we simply log it.
			u.logger.Info("failed to fetch thumbnail", zap.String("url", u.url.String()), zap.Error(err))
		} else {
			preview.Thumbnail = t
		}
	}

	preview.Title = ogMetadata.Title
	preview.Description = ogMetadata.Description
	return preview, nil
}

func normalizeHostname(hostname string) string {
	hostname = strings.ToLower(hostname)
	re := regexp.MustCompile(`^www\.(.*)$`)
	return re.ReplaceAllString(hostname, "$1")
}

func newUnfurler(logger *zap.Logger, httpClient http.Client, url *neturl.URL) Unfurler {
	switch normalizeHostname(url.Hostname()) {
	case "reddit.com":
		return OEmbedUnfurler{
			oembedEndpoint: "https://www.reddit.com/oembed",
			url:            url,
			logger:         logger,
			httpClient:     httpClient,
		}
	case "giphy.com", "media.giphy.com":
		return OEmbedUnfurler{
			oembedEndpoint: "https://giphy.com/services/oembed",
			url:            url,
			logger:         logger,
			httpClient:     httpClient,
		}
	case "tenor.com", "media.tenor.com":
		return OEmbedUnfurler{
			oembedEndpoint: "https://tenor.com/oembed",
			url:            url,
			logger:         logger,
			httpClient:     httpClient,
		}
	default:
		return OpenGraphUnfurler{
			url:        url,
			logger:     logger,
			httpClient: httpClient,
		}
	}
}

func unfurl(logger *zap.Logger, httpClient http.Client, url string) (common.LinkPreview, error) {
	var preview common.LinkPreview

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return preview, err
	}

	unfurler := newUnfurler(logger, httpClient, parsedURL)
	preview, err = unfurler.unfurl()
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

func NewDefaultHTTPClient() http.Client {
	return http.Client{Timeout: defaultRequestTimeout}
}

// UnfurlURLs assumes clients pass URLs verbatim that were validated and
// processed by GetURLs.
func UnfurlURLs(logger *zap.Logger, httpClient http.Client, urls []string) ([]common.LinkPreview, error) {
	var err error
	if logger == nil {
		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	previews := make([]common.LinkPreview, 0, len(urls))

	for _, url := range urls {
		logger.Debug("unfurling", zap.String("url", url))
		p, err := unfurl(logger, httpClient, url)
		if err != nil {
			logger.Info("failed to unfurl", zap.String("url", url), zap.Error(err))
			continue
		}
		previews = append(previews, p)
	}

	return previews, nil
}
