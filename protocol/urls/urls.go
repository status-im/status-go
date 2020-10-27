package urls

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
}

type Site struct {
	Title   string `json:"title"`
	Address string `json:"address"`
}

const YouTubeOembedLink = "https://www.youtube.com/oembed?format=json&url=%s"

func LinkPreviewWhitelist() []Site {
	return []Site{
		Site{
			Title:   "YouTube",
			Address: "youtube.com",
		},
		Site{
			Title:   "YouTube shortener",
			Address: "youtu.be",
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

	switch url.Hostname() {
	case "youtube.com", "www.youtube.com", "youtu.be":
		return GetYoutubePreviewData(link)
	}

	return previewData, fmt.Errorf("Link %s isn't whitelisted. Hostname - %s", link, url.Hostname())
}
