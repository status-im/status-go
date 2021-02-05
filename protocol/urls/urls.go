package urls

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type GiphyOembedData struct {
	ProviderName string `json:"provider_name"`
	Title        string `json:"title"`
	URL          string `json:"url"`
}

type TenorOembedData struct {
	ProviderName string `json:"provider_name"`
	ThumbnailURL string `json:"thumbnail_url"`
	AuthorName   string `json:"author_name"`
}

type LinkPreviewData struct {
	Site         string `json:"site" meta:"og:site_name"`
	Title        string `json:"title" meta:"og:title"`
	ThumbnailURL string `json:"thumbnailUrl" meta:"og:image"`
	ContentType  string `json:"contentType"`
}

type Site struct {
	Title     string `json:"title"`
	Address   string `json:"address"`
	ImageSite bool   `json:"imageSite"`
}

const YoutubeOembedLink = "https://www.youtube.com/oembed?format=json&url=%s"
const GiphyOembedLink = "https://giphy.com/services/oembed?url=%s"
const TenorOembedLink = "https://tenor.com/oembed?url=%s"

var httpClient = http.Client{
	Timeout: 30 * time.Second,
}

func LinkPreviewWhitelist() []Site {
	return []Site{
		Site{
			Title:     "YouTube",
			Address:   "youtube.com",
			ImageSite: false,
		},
		Site{
			Title:     "YouTube shortener",
			Address:   "youtu.be",
			ImageSite: false,
		},
		Site{
			Title:     "Tenor GIFs",
			Address:   "tenor.com",
			ImageSite: true,
		},
		Site{
			Title:     "GIPHY GIFs",
			Address:   "giphy.com",
			ImageSite: true,
		},
		Site{
			Title:     "GIPHY GIFs subdomain",
			Address:   "media.giphy.com",
			ImageSite: true,
		},
		Site{
			Title:     "GitHub",
			Address:   "github.com",
			ImageSite: false,
		},
	}
}

func GetURLContent(url string) (data []byte, err error) {
	// nolint: gosec
	response, err := httpClient.Get(url)
	if err != nil {
		return data, fmt.Errorf("can't get content from link %s", url)
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

func GetYoutubeOembed(url string) (data YoutubeOembedData, err error) {
	oembedLink := fmt.Sprintf(YoutubeOembedLink, url)

	jsonBytes, err := GetURLContent(oembedLink)
	if err != nil {
		return data, fmt.Errorf("can't get bytes from youtube oembed response on %s link", oembedLink)
	}

	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		return data, fmt.Errorf("can't unmarshall json")
	}

	return data, nil
}

func GetYoutubePreviewData(link string) (previewData LinkPreviewData, err error) {
	oembedData, err := GetYoutubeOembed(link)
	if err != nil {
		return previewData, err
	}

	previewData.Title = oembedData.Title
	previewData.Site = oembedData.ProviderName
	previewData.ThumbnailURL = oembedData.ThumbnailURL

	return previewData, nil
}

func GetGithubPreviewData(link string) (previewData LinkPreviewData, err error) {
	// nolint: gosec
	res, err := httpClient.Get(link)

	if err != nil {
		return previewData, fmt.Errorf("can't get content from link %s", link)
	}

	err = metabolize.Metabolize(res.Body, &previewData)
	if err != nil {
		return previewData, fmt.Errorf("can't get meta info from link %s", link)
	}

	return previewData, nil
}

func GetGiphyOembed(url string) (data GiphyOembedData, err error) {
	oembedLink := fmt.Sprintf(GiphyOembedLink, url)

	jsonBytes, err := GetURLContent(oembedLink)

	if err != nil {
		return data, fmt.Errorf("can't get bytes from Giphy oembed response at %s", oembedLink)
	}

	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		return data, fmt.Errorf("can't unmarshall json")
	}

	return data, nil
}

func GetGiphyPreviewData(link string) (previewData LinkPreviewData, err error) {
	oembedData, err := GetGiphyOembed(link)
	if err != nil {
		return previewData, err
	}

	previewData.Title = oembedData.Title
	previewData.Site = oembedData.ProviderName
	previewData.ThumbnailURL = oembedData.URL

	return previewData, nil
}

func GetTenorOembed(url string) (data TenorOembedData, err error) {
	oembedLink := fmt.Sprintf(TenorOembedLink, url)

	jsonBytes, err := GetURLContent(oembedLink)

	if err != nil {
		return data, fmt.Errorf("can't get bytes from Tenor oembed response at %s", oembedLink)
	}

	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		return data, fmt.Errorf("can't unmarshall json")
	}

	return data, nil
}

func GetTenorPreviewData(link string) (previewData LinkPreviewData, err error) {
	oembedData, err := GetTenorOembed(link)
	if err != nil {
		return previewData, err
	}

	previewData.Title = oembedData.AuthorName // Tenor Oembed service doesn't return title of the Gif
	previewData.Site = oembedData.ProviderName
	previewData.ThumbnailURL = oembedData.ThumbnailURL

	return previewData, nil
}

func GetLinkPreviewData(link string) (previewData LinkPreviewData, err error) {
	url, err := url.Parse(link)
	if err != nil {
		return previewData, fmt.Errorf("cant't parse link %s", link)
	}

	hostname := strings.ToLower(url.Hostname())

	switch hostname {
	case "youtube.com", "youtu.be", "www.youtube.com":
		return GetYoutubePreviewData(link)
	case "github.com":
		return GetGithubPreviewData(link)
	case "giphy.com":
		return GetGiphyPreviewData(link)
	case "tenor.com":
		return GetTenorPreviewData(link)
	default:
		return previewData, fmt.Errorf("link %s isn't whitelisted. Hostname - %s", link, url.Hostname())
	}
}
