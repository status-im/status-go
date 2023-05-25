package linkpreview

import (
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

// UnfurlError means a non-critical error, and that processing of the preview
// should be interrupted and the preview probably ignored.
type UnfurlError struct {
	msg string
	url string
	err error
}

func (ue UnfurlError) Error() string {
	return fmt.Sprintf("%s, url='%s'", ue.msg, ue.url)
}

func (ue UnfurlError) Unwrap() error {
	return ue.err
}

type LinkPreview struct {
	common.LinkPreview
}

type Unfurler interface {
	unfurl() (common.LinkPreview, error)
}

const (
	defaultRequestTimeout = 15000 * time.Millisecond

	// Without an user agent, many providers treat status-go as a gluttony bot,
	// and either respond more frequently with a 429 (Too Many Requests), or
	// simply refuse to return valid data.
	defaultUserAgent = "status-go/v0.151.15"

	// Currently set to English, but we could make this setting dynamic according
	// to the user's language of choice.
	defaultAcceptLanguage = "en-US,en;q=0.5"
)

func fetchResponseBody(logger *zap.Logger, httpClient http.Client, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = res.Body.Close(); err != nil {
			logger.Error("Failed to close response body", zap.Error(err))
		}
	}()

	if res.StatusCode >= http.StatusBadRequest {
		return nil, errors.New(http.StatusText(res.StatusCode))
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
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

	imgBytes, err := fetchResponseBody(logger, httpClient, url)
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

type OEmbedResponse struct {
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
}

func (u OEmbedUnfurler) unfurl() (common.LinkPreview, error) {
	preview := newDefaultLinkPreview(u.url)

	requestURL, err := neturl.Parse(u.oembedEndpoint)
	if err != nil {
		return preview, err
	}

	// When format is specified, the provider MUST return data in the requested
	// format, else return an error.
	requestURL.RawQuery = neturl.Values{
		"url":    {u.url.String()},
		"format": {"json"},
	}.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL.String(), nil)
	if err != nil {
		return preview, err
	}
	req.Header.Set("accept", "application/json; charset=utf-8")
	req.Header.Set("accept-language", defaultAcceptLanguage)
	req.Header.Set("user-agent", defaultUserAgent)

	res, err := u.httpClient.Do(req)
	defer func() {
		if res != nil {
			if err = res.Body.Close(); err != nil {
				u.logger.Error("failed to close response body", zap.Error(err))
			}
		}
	}()
	if err != nil {
		return preview, UnfurlError{
			msg: "failed to get oEmbed",
			url: u.url.String(),
			err: err,
		}
	}

	if res.StatusCode >= http.StatusBadRequest {
		return preview, UnfurlError{
			msg: fmt.Sprintf("failed to fetch oEmbed metadata, statusCode='%d'", res.StatusCode),
			url: u.url.String(),
			err: nil,
		}
	}

	var oembedResponse OEmbedResponse
	oembedBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return preview, err
	}
	err = json.Unmarshal(oembedBytes, &oembedResponse)
	if err != nil {
		return preview, err
	}

	if oembedResponse.Title == "" {
		return preview, UnfurlError{
			msg: "missing title",
			url: u.url.String(),
			err: errors.New(""),
		}
	}

	preview.Title = oembedResponse.Title
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

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", u.url.String(), nil)
	if err != nil {
		return preview, err
	}
	req.Header.Set("accept", "text/html; charset=utf-8")
	req.Header.Set("accept-language", defaultAcceptLanguage)
	req.Header.Set("user-agent", defaultUserAgent)

	res, err := u.httpClient.Do(req)
	defer func() {
		if res != nil {
			if err = res.Body.Close(); err != nil {
				u.logger.Error("failed to close response body", zap.Error(err))
			}
		}
	}()
	if err != nil {
		return preview, UnfurlError{
			msg: "failed to get HTML page",
			url: u.url.String(),
			err: err,
		}
	}

	if res.StatusCode >= http.StatusBadRequest {
		return preview, UnfurlError{
			msg: fmt.Sprintf("failed to fetch OpenGraph metadata, statusCode='%d'", res.StatusCode),
			url: u.url.String(),
			err: nil,
		}
	}

	var ogMetadata OpenGraphMetadata
	err = metabolize.Metabolize(res.Body, &ogMetadata)
	if err != nil {
		return preview, UnfurlError{
			msg: "failed to parse OpenGraph data",
			url: u.url.String(),
			err: err,
		}
	}

	// There are URLs like https://wikipedia.org/ that don't have an OpenGraph
	// title tag, but article pages do. In the future, we can fallback to the
	// website's title by using the <title> tag.
	if ogMetadata.Title == "" {
		return preview, UnfurlError{
			msg: "missing title",
			url: u.url.String(),
			err: errors.New(""),
		}
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
		p, err := unfurl(logger, httpClient, url)
		if err != nil {
			if unfurlErr, ok := err.(UnfurlError); ok {
				logger.Info("failed to unfurl", zap.Error(unfurlErr))
				continue
			}

			return nil, err
		}
		previews = append(previews, p)
	}

	return previews, nil
}
