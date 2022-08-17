package browsers

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "browsers-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "browsers-tests")
	require.NoError(t, err)
	return NewDB(db), func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func setupTestAPI(t *testing.T) (*API, func()) {
	db, cancel := setupTestDB(t)
	return &API{db: db}, cancel
}

func TestBookmarks(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	bookmark := &Bookmark{
		Name:     "MyBookmark",
		URL:      "https://status.im",
		ImageURL: "",
	}

	_, err := api.StoreBookmark(context.TODO(), *bookmark)
	require.NoError(t, err)

	rst, err := api.GetBookmarks(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 1)

	err = api.RemoveBookmark(context.TODO(), bookmark.URL)
	require.NoError(t, err)
	rst, err = api.GetBookmarks(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, rst[0].Removed, true)

	require.NoError(t, api.DeleteBookmark(context.TODO(), bookmark.URL))
	rst, err = api.GetBookmarks(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 0)

}

func TestShouldSyncBookmark(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	bookmark := &Bookmark{
		Name:     "MyBookmark",
		URL:      "https://status.im",
		ImageURL: "",
		Clock:    1,
	}

	_, err := api.StoreBookmark(context.TODO(), *bookmark)
	require.NoError(t, err)

	bookmark.Clock = 2
	shouldSync, err := api.db.shouldSyncBookmark(bookmark, nil)
	require.NoError(t, err)
	require.True(t, shouldSync)

	bookmark.Clock = 0
	shouldSync, err = api.db.shouldSyncBookmark(bookmark, nil)
	require.NoError(t, err)
	require.False(t, shouldSync)
}
