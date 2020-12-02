package urls

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type OembedData struct {
	ProviderName string `json:"provider_name"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type LinkPreviewData struct {
	Site         string `json:"site"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnailUrl"`
	ContentType  string `json:"contentType"`
}

type Site struct {
	Title        string `json:"title"`
	Address      string `json:"address"`
	ImageSite    bool   `json:imageSite`
}

const YouTubeOembedLink = "https://www.youtube.com/oembed?format=json&url=%s"

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
	}
}

func GetURLContent(url string) (data []byte, err error) {

	// nolint: gosec
	response, err := http.Get(url)
	if err != nil {
		return data, fmt.Errorf("Can't get content from link %s", url)
	}
	defer response.Body.Close()
	return ioutil.ReadAll(response.Body)
}

func GetYoutubeOembed(url string) (data OembedData, err error) {
	oembedLink := fmt.Sprintf(YouTubeOembedLink, url)

	jsonBytes, err := GetURLContent(oembedLink)
	if err != nil {
		return data, fmt.Errorf("Can't get bytes from youtube oembed response on %s link", oembedLink)
	}

	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		return data, fmt.Errorf("Can't unmarshall json")
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

func GetLinkPreviewData(link string) (previewData LinkPreviewData, err error) {

	url, err := url.Parse(link)
	if err != nil {
		return previewData, fmt.Errorf("Cant't parse link %s", link)
	}

	hostname := strings.ToLower(url.Hostname())
	youtubeHostnames := []string{"youtube.com", "www.youtube.com", "youtu.be"}
	for _, youtubeHostname := range youtubeHostnames {
		if youtubeHostname == hostname {
			return GetYoutubePreviewData(link)
		}
	}
	for _, site := range LinkPreviewWhitelist() {
		if strings.HasSuffix(hostname, site.Address) && site.ImageSite {
			content, contentErr := GetURLContent(link)
			if contentErr != nil {
				return previewData, contentErr
			}
			previewData.ThumbnailURL = link
			previewData.ContentType = http.DetectContentType(content)
			return previewData, nil
		}
	}

	return previewData, fmt.Errorf("Link %s isn't whitelisted. Hostname - %s", link, url.Hostname())
}
