package gif

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/multiaccounts/accounts"
)

type Gif struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	TinyURL    string `json:"tinyUrl"`
	Height     int    `json:"height"`
	IsFavorite bool   `json:"isFavorite"`
}

type Container struct {
	Items []Gif `json:"items"`
}

var tenorAPIKey = ""
var defaultParams = "&media_filter=minimal&limit=50&key="

const maxRetry = 3
const baseURL = "https://g.tenor.com/v1/"

func NewGifAPI(db *accounts.Database) *API {
	return &API{db, &http.Client{Timeout: time.Minute}}
}

// API is class with methods available over RPC.
type API struct {
	db         *accounts.Database
	httpClient *http.Client
}

func (api *API) SetTenorAPIKey(key string) (err error) {
	log.Info("[GifAPI::SetTenorAPIKey]")
	err = api.db.SaveSetting("gifs/api-key", key)
	if err != nil {
		return err
	}
	tenorAPIKey = key
	return nil
}

func (api *API) GetContentWithRetry(path string) (value string, err error) {
	var currentRetry = 0
	var response *http.Response

	for currentRetry < maxRetry {
		response, err = api.httpClient.Get(baseURL + path + defaultParams + tenorAPIKey)
		if err != nil {
			log.Error("can't get content from path %s", path)
			currentRetry++
			time.Sleep(100 * time.Millisecond)
		} else {
			break
		}
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Status error: %v", response.StatusCode)
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("Read body: %v", err)
	}

	return string(data), nil
}

func (api *API) FetchGifs(path string) (value string, err error) {
	log.Info("[GifAPI::fetchGifs]")
	return api.GetContentWithRetry(path)
}

func (api *API) UpdateRecentGifs(updatedGifs json.RawMessage) (err error) {
	log.Info("[GifAPI::updateRecentGifs]")
	recentGifsContainer := Container{}
	err = json.Unmarshal(updatedGifs, &recentGifsContainer)
	if err != nil {
		return err
	}
	err = api.db.SaveSetting("gifs/recent-gifs", recentGifsContainer.Items)
	if err != nil {
		return err
	}
	return nil
}

func (api *API) UpdateFavoriteGifs(updatedGifs json.RawMessage) (err error) {
	log.Info("[GifAPI::updateFavoriteGifs]", updatedGifs)
	favsGifsContainer := Container{}
	err = json.Unmarshal(updatedGifs, &favsGifsContainer)
	if err != nil {
		return err
	}
	err = api.db.SaveSetting("gifs/favorite-gifs", favsGifsContainer.Items)
	if err != nil {
		return err
	}
	return nil
}

func (api *API) GetRecentGifs() (recentGifs []Gif, err error) {
	log.Info("[GifAPI::getRecentGifs]")
	gifs, err := api.db.GifRecents()
	if err != nil {
		return nil, err
	}
	savedRecentGifs := []Gif{}
	err = json.Unmarshal(gifs, &savedRecentGifs)
	if err != nil {
		return nil, err
	}
	recentGifs = savedRecentGifs
	return recentGifs, nil
}

func (api *API) GetFavoriteGifs() (favoriteGifs []Gif, err error) {
	log.Info("[GifAPI::getFavoriteGifs]")
	gifs, err := api.db.GifFavorites()
	if err != nil {
		return nil, err
	}
	savedFavGifs := []Gif{}
	err = json.Unmarshal(gifs, &savedFavGifs)
	if err != nil {
		return nil, err
	}
	favoriteGifs = savedFavGifs
	return favoriteGifs, nil
}
