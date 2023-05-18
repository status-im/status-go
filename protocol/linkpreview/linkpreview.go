package linkpreview

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	neturl "net/url"
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
	unfurl(*neturl.URL) (common.LinkPreview, error)
}

const (
	requestTimeout = 15000 * time.Millisecond

	// Certain websites return an HTML error page if the user agent is unknown to
	// them, e.g. IMDb.
	defaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/109.0"

	// Currently set to English, but we could make this setting dynamic according
	// to the user's language of choice.
	defaultAcceptLanguage = "en-US,en;q=0.5"
)

var (
	httpClient = http.Client{Timeout: requestTimeout}
)

func fetchResponseBody(logger *zap.Logger, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
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

func httpGETForOpenGraph(url string) (*http.Response, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, cancel, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept-Language", defaultAcceptLanguage)

	res, err := httpClient.Do(req)
	return res, cancel, err
}

func fetchThumbnail(logger *zap.Logger, url string) (common.LinkPreviewThumbnail, error) {
	var thumbnail common.LinkPreviewThumbnail

	imgBytes, err := fetchResponseBody(logger, url)
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

type OpenGraphMetadata struct {
	Title        string `json:"title" meta:"og:title"`
	Description  string `json:"description" meta:"og:description"`
	ThumbnailURL string `json:"thumbnailUrl" meta:"og:image"`
}

// OpenGraphUnfurler can be used either as the default unfurler for some websites
// (e.g. GitHub), or as a fallback strategy. It parses HTML and extract
// OpenGraph meta tags. If an oEmbed endpoint is available, it should be
// preferred.
type OpenGraphUnfurler struct {
	logger *zap.Logger
}

func (u OpenGraphUnfurler) unfurl(url *neturl.URL) (common.LinkPreview, error) {
	preview := newDefaultLinkPreview(url)

	res, cancel, err := httpGETForOpenGraph(url.String())
	defer cancel()
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
			url: url.String(),
			err: err,
		}
	}

	// Behave like WhatsApp, i.e. if the response is a 404, consider the URL
	// unfurleable. We can try to unfurl from the 404 HTML, which works well for
	// certain websites, like GitHub, but it also potentially confuses users
	// because they'll be sharing previews that don't match the actual URLs.
	if res.StatusCode == http.StatusNotFound {
		return preview, UnfurlError{
			msg: "could not find page",
			url: url.String(),
			err: errors.New(""),
		}
	}

	var ogMetadata OpenGraphMetadata
	err = metabolize.Metabolize(res.Body, &ogMetadata)
	if err != nil {
		return preview, UnfurlError{
			msg: "failed to parse OpenGraph data",
			url: url.String(),
			err: err,
		}
	}

	// There are URLs like https://wikipedia.org/ that don't have an OpenGraph
	// title tag, but article pages do. In the future, we can fallback to the
	// website's title by using the <title> tag.
	if ogMetadata.Title == "" {
		return preview, UnfurlError{
			msg: "missing title",
			url: url.String(),
			err: errors.New(""),
		}
	}

	if ogMetadata.ThumbnailURL != "" {
		t, err := fetchThumbnail(u.logger, ogMetadata.ThumbnailURL)
		if err != nil {
			// Given we want to fetch thumbnails on a best-effort basis, if an error
			// happens we simply log it.
			u.logger.Info("failed to fetch thumbnail", zap.String("url", url.String()), zap.Error(err))
		} else {
			preview.Thumbnail = t
		}
	}

	preview.Title = ogMetadata.Title
	preview.Description = ogMetadata.Description
	return preview, nil
}

func newUnfurler(logger *zap.Logger, url *neturl.URL) Unfurler {
	u := new(OpenGraphUnfurler)
	u.logger = logger
	return u
}

func unfurl(logger *zap.Logger, url string) (common.LinkPreview, error) {
	var preview common.LinkPreview

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return preview, err
	}

	unfurler := newUnfurler(logger, parsedURL)
	preview, err = unfurler.unfurl(parsedURL)
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

// UnfurlURLs assumes clients pass URLs verbatim that were validated and
// processed by GetURLs.
func UnfurlURLs(logger *zap.Logger, urls []string) ([]common.LinkPreview, error) {
	var err error
	if logger == nil {
		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, fmt.Errorf("failed to create logger: %w", err)
		}
	}

	previews := make([]common.LinkPreview, 0, len(urls))

	for _, url := range urls {
		p, err := unfurl(logger, url)
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
