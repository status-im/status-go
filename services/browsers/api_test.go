package browsers

import (
	"context"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "browsers-tests-")
	require.NoError(t, err)
	db, err := InitializeDB(tmpfile.Name(), "browsers-tests")
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func setupTestAPI(t *testing.T) (*API, func()) {
	db, cancel := setupTestDB(t)
	return &API{s: &Service{db: db}}, cancel
}

func TestBrowsersOrderedNewestFirst(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	browsers := []*Browser{
		{
			ID:        "1",
			Name:      "first",
			Dapp:      true,
			Timestamp: hexutil.Uint64(10),
		},
		{
			ID:        "2",
			Name:      "second",
			Dapp:      true,
			Timestamp: hexutil.Uint64(50),
		},
		{
			ID:        "3",
			Name:      "third",
			Dapp:      true,
			Timestamp: hexutil.Uint64(100),
		},
	}
	// insert in reverse order for a clean test
	for i := 0; i < len(browsers); i++ {
		require.NoError(t, api.AddBrowser(context.TODO(), *browsers[i]))
	}

	sort.Slice(browsers, func(i, j int) bool {
		return browsers[i].Timestamp > browsers[j].Timestamp
	})

	rst, err := api.GetBrowsers(context.TODO())
	require.NoError(t, err)
	require.Equal(t, browsers, rst)
}

func TestBrowsersHistoryIncluded(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	browser := &Browser{
		ID:           "1",
		Name:         "first",
		Dapp:         true,
		Timestamp:    hexutil.Uint64(10),
		HistoryIndex: hexutil.Uint(1),
		History:      []string{"one", "two"},
	}
	require.NoError(t, api.AddBrowser(context.TODO(), *browser))
	rst, err := api.GetBrowsers(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, browser, rst[0])
}

func TestBrowsersReplaceOnUpdate(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	browser := &Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: hexutil.Uint64(10),
		History:   []string{"one", "two"},
	}
	require.NoError(t, api.AddBrowser(context.TODO(), *browser))
	browser.Dapp = false
	browser.History = []string{"one", "three"}
	browser.Timestamp = hexutil.Uint64(107)
	require.NoError(t, api.AddBrowser(context.TODO(), *browser))
	rst, err := api.GetBrowsers(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, browser, rst[0])
}

func TestDeleteBrowser(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	browser := &Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: hexutil.Uint64(10),
		History:   []string{"one", "two"},
	}

	require.NoError(t, api.AddBrowser(context.TODO(), *browser))
	rst, err := api.GetBrowsers(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 1)

	require.NoError(t, api.DeleteBrowser(context.TODO(), browser.ID))
	rst, err = api.GetBrowsers(context.TODO())
	require.NoError(t, err)
	require.Len(t, rst, 0)
}
