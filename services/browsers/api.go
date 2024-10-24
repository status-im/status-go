package browsers

import (
	"context"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
)

func NewAPI(db *Database) *API {
	return &API{db: db}
}

// API is class with methods available over RPC.
type API struct {
	db *Database
}

func (api *API) GetBookmarks(ctx context.Context) ([]*Bookmark, error) {
	logutils.ZapLogger().Debug("call to get bookmarks")
	rst, err := api.db.GetBookmarks()
	logutils.ZapLogger().Debug("result from database for bookmarks", zap.Int("len", len(rst)))
	return rst, err
}

func (api *API) StoreBookmark(ctx context.Context, bookmark Bookmark) (Bookmark, error) {
	logutils.ZapLogger().Debug("call to create a bookmark")
	bookmarkResult, err := api.db.StoreBookmark(bookmark)
	logutils.ZapLogger().Debug("result from database for creating a bookmark", zap.Error(err))
	return bookmarkResult, err
}

func (api *API) UpdateBookmark(ctx context.Context, originalURL string, bookmark Bookmark) error {
	logutils.ZapLogger().Debug("call to update a bookmark")
	err := api.db.UpdateBookmark(originalURL, bookmark)
	logutils.ZapLogger().Debug("result from database for updating a bookmark", zap.Error(err))
	return err
}

func (api *API) DeleteBookmark(ctx context.Context, url string) error {
	logutils.ZapLogger().Debug("call to remove a bookmark")
	err := api.db.DeleteBookmark(url)
	logutils.ZapLogger().Debug("result from database for remove a bookmark", zap.Error(err))
	return err
}

func (api *API) RemoveBookmark(ctx context.Context, url string) error {
	logutils.ZapLogger().Debug("call to remove a bookmark logically")
	err := api.db.RemoveBookmark(url)
	logutils.ZapLogger().Debug("result from database for remove a bookmark logically", zap.Error(err))
	return err
}
