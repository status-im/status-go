package urls

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/keighl/metabolize"
)

type YoutubeOembedData struct {
	ProviderName string `json:"provider_name"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type TwitterOembedData struct {
	ProviderName string `json:"provider_name"`
	AuthorName   string `json:"author_name"`
	HTML         string `json:"html"`
}

type GiphyOembedData struct {
	ProviderName string `json:"provider_name"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Height       int    `json:"height"`
	Width        int    `json:"width"`
}

type LinkPreviewData struct {
	Site         string `json:"site" meta:"og:site_name"`
	Title        string `json:"title" meta:"og:title"`
	ThumbnailURL string `json:"thumbnailUrl" meta:"og:image"`
	ContentType  string `json:"contentType"`
	Height       int    `json:"height"`
	Width        int    `json:"width"`
}

type Site struct {
	Title     string `json:"title"`
	Address   string `json:"address"`
	ImageSite bool   `json:"imageSite"`
}

const (
	YoutubeOembedLink = "https://www.youtube.com/oembed?format=json&url=%s"
	TwitterOembedLink = "https://publish.twitter.com/oembed?url=%s"
	GiphyOembedLink   = "https://giphy.com/services/oembed?url=%s"
)

func LinkPreviewWhitelist() []Site {
	return []Site{
		{
			Title:     "Status",
			Address:   "our.status.im",
			ImageSite: false,
		},
		{
			Title:     "YouTube",
			Address:   "youtube.com",
			ImageSite: false,
		},
		{
			Title:     "YouTube with subdomain",
			Address:   "www.youtube.com",
			ImageSite: false,
		},
		{
			Title:     "YouTube shortener",
			Address:   "youtu.be",
			ImageSite: false,
		},
		{
			Title:     "YouTube Mobile",
			Address:   "m.youtube.com",
			ImageSite: false,
		},
		{
			Title:     "Twitter",
			Address:   "twitter.com",
			ImageSite: false,
		},
		{
			Title:     "Twitter Mobile",
			Address:   "mobile.twitter.com",
			ImageSite: false,
		},
		{
			Title:     "GIPHY GIFs shortener",
			Address:   "gph.is",
			ImageSite: true,
		},
		{
			Title:     "GIPHY GIFs",
			Address:   "giphy.com",
			ImageSite: true,
		},
		{
			Title:     "GIPHY GIFs subdomain",
			Address:   "media.giphy.com",
			ImageSite: true,
		},
		{
			Title:     "GitHub",
			Address:   "github.com",
			ImageSite: false,
		},
		{
			Title:     "Tenor GIFs subdomain",
			Address:   "media.tenor.com",
			ImageSite: false,
		},
		// Medium unfurling is failing - https://github.com/status-im/status-go/issues/2192
		//
		// {
		// 	Title:     "Medium",
		// 	Address:   "medium.com",
		// 	ImageSite: false,
		// },
	}
}

type oembedGetter interface {
	GetOembed(name, endpoint, url string, data interface{}) error
	GetResponse(url string) (resp *http.Response, err error)
}

// baseOembedGetter is the production oembedGetter
type baseOembedGetter struct {
	client http.Client
}

func newOembedGetter() *baseOembedGetter {
	og := new(baseOembedGetter)
	og.client = http.Client{Timeout: 30 * time.Second}
	return og
}

func (o *baseOembedGetter) getURLContent(url string) (data []byte, err error) {
	response, err := o.client.Get(url)
	o.client.Head()
	if err != nil {
		return data, fmt.Errorf("can't get content from link %s", url)
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

func (o *baseOembedGetter) GetOembed(name, endpoint, url string, data interface{}) error {
	oembedLink := fmt.Sprintf(endpoint, url)

	jsonBytes, err := o.getURLContent(oembedLink)
	if err != nil {
		return fmt.Errorf("can't get bytes from %s oembed response on %s link", name, oembedLink)
	}

	return json.Unmarshal(jsonBytes, &data)
}

func (o *baseOembedGetter) GetResponse(url string) (resp *http.Response, err error) {
	return o.client.Get(url)
}

// oembedParser
type oembedParser struct {
	getter oembedGetter
}

func newOembedParser(getter oembedGetter) *oembedParser {
	return &oembedParser{
		getter: getter,
	}
}

func (o *oembedParser) getYoutubePreviewData(link string) (previewData LinkPreviewData, err error) {
	oembedData := new(YoutubeOembedData)
	err = o.getter.GetOembed("Youtube", YoutubeOembedLink, link, &oembedData)
	if err != nil {
		return
	}

	previewData.Title = oembedData.Title
	previewData.Site = oembedData.ProviderName
	previewData.ThumbnailURL = oembedData.ThumbnailURL
	return
}

func (o *oembedParser) getTwitterPreviewData(link string) (previewData LinkPreviewData, err error) {
	oembedData := new(TwitterOembedData)
	err = o.getter.GetOembed("Twitter", TwitterOembedLink, link, oembedData)
	if err != nil {
		return previewData, err
	}

	previewData.Title = o.getReadableTextFromTweetHTML(oembedData.HTML)
	previewData.Site = oembedData.ProviderName

	return previewData, nil
}

func (o *oembedParser) getReadableTextFromTweetHTML(s string) string {
	s = strings.ReplaceAll(s, "\u003Cbr\u003E", "\n")   // Adds line break for all <br>
	s = strings.ReplaceAll(s, "https://", "\nhttps://") // Displays links in next line
	s = html.UnescapeString(s)                          // Parses html special characters like &#225;
	s = stripHTMLTags(s)
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "\n")
	s = strings.TrimLeft(s, "\n")

	return s
}

func (o *oembedParser) getGenericLinkPreviewData(link string) (previewData LinkPreviewData, err error) {
	res, err := o.getter.GetResponse(link)
	if err != nil {
		return previewData, fmt.Errorf("can't get content from link %s", link)
	}

	err = metabolize.Metabolize(res.Body, &previewData)
	if err != nil {
		return previewData, fmt.Errorf("can't get meta info from link %s", link)
	}

	return previewData, nil
}

func (o *oembedParser) FakeGenericImageLinkPreviewData(title string, link string) (previewData LinkPreviewData, err error) {
	u, err := url.Parse(link)
	if err != nil {
		return previewData, fmt.Errorf("Failed to parse link %s", link)
	}

	res, err := httpClient.Head(link)
	if err != nil {
		return previewData, fmt.Errorf("Failed to get HEAD from link %s", link)
	}

	if res.StatusCode != 200 {
		return previewData, fmt.Errorf("Image link %s is not available", link)
	}

	previewData.Title = title
	previewData.Site = strings.ToLower(u.Hostname())
	previewData.ContentType = res.Header.Get("Content-type")
	previewData.ThumbnailURL = link
	previewData.Height = 0
	previewData.Width = 0
	return previewData, nil
}

func (o *oembedParser) getGiphyPreviewData(link string) (previewData LinkPreviewData, err error) {
	oembedData := new(GiphyOembedData)
	err = o.getter.GetOembed("Giphy", GiphyOembedLink, link, oembedData)
	if err != nil {
		return previewData, err
	}

	previewData.Title = oembedData.Title
	previewData.Site = oembedData.ProviderName
	previewData.ThumbnailURL = oembedData.URL
	previewData.Height = oembedData.Height
	previewData.Width = oembedData.Width

	return previewData, nil
}

// getGiphyLongURL Giphy has a shortener service called gph.is, the oembed service doesn't work with shortened urls,
// so we need to fetch the long url first
func (o *oembedParser) getGiphyLongURL(shortURL string) (longURL string, err error) {
	res, err := o.getter.GetResponse(shortURL)
	if err != nil {
		return longURL, fmt.Errorf("can't get bytes from Giphy's short url at %s", shortURL)
	}

	canonicalURL := res.Request.URL.String()
	if canonicalURL == shortURL {
		// no redirect, ie. not a valid url
		return longURL, fmt.Errorf("unable to process Giphy's short url at %s", shortURL)
	}

	return canonicalURL, err
}

func (o *oembedParser) getGiphyShortURLPreviewData(shortURL string) (data LinkPreviewData, err error) {
	longURL, err := o.getGiphyLongURL(shortURL)
	if err != nil {
		return data, err
	}

	return o.getGiphyPreviewData(longURL)
}

func (o *oembedParser) GetLinkPreviewData(link string) (previewData LinkPreviewData, err error) {
	u, err := url.Parse(link)
	if err != nil {
		return previewData, fmt.Errorf("cant't parse link %s", link)
	}

	hostname := strings.ToLower(u.Hostname())

	switch hostname {
	case "youtube.com", "youtu.be", "www.youtube.com", "m.youtube.com":
		return o.getYoutubePreviewData(link)
	case "github.com", "our.status.im":
		return o.getGenericLinkPreviewData(link)
	case "giphy.com", "media.giphy.com":
		return o.getGiphyPreviewData(link)
	case "gph.is":
		return o.getGiphyShortURLPreviewData(link)
	case "twitter.com", "mobile.twitter.com":
		return o.getTwitterPreviewData(link)
	case "media.tenor.com":
		//TODO rename to getTenor...?
		return o.FakeGenericImageLinkPreviewData("Tenor", link)
	default:
		return previewData, fmt.Errorf("link %s isn't whitelisted. Hostname - %s", link, u.Hostname())
	}
}
