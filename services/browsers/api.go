package browsers

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
)

func NewAPI(db *Database) *API {
	return &API{db: db}
}

// API is class with methods available over RPC.
type API struct {
	db *Database
}

func (api *API) AddBrowser(ctx context.Context, browser Browser) error {
	return api.db.InsertBrowser(browser)
}

func (api *API) GetBrowsers(ctx context.Context) ([]*Browser, error) {
	return api.db.GetBrowsers()
}

func (api *API) DeleteBrowser(ctx context.Context, id string) error {
	return api.db.DeleteBrowser(id)
}

func (api *API) GetBookmarks(ctx context.Context) ([]*Bookmark, error) {
	log.Debug("call to get bookmarks")
	rst, err := api.db.GetBookmarks()
	log.Debug("result from database for bookmarks", "len", len(rst))
	return rst, err
}

func (api *API) StoreBookmark(ctx context.Context, bookmark Bookmark) error {
	log.Debug("call to create a bookmark")
	err := api.db.StoreBookmark(bookmark)
	log.Debug("result from database for creating a bookmark", "err", err)
	return err
}

func (api *API) DeleteBookmark(ctx context.Context, url string) error {
	log.Debug("call to remove a bookmark")
	err := api.db.DeleteBookmark(url)
	log.Debug("result from database for remove a bookmark", "err", err)
	return err
}
