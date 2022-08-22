package protocol

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/services/browsers"
)

func TestBrowsersOrderedNewestFirst(t *testing.T) {
	db, _ := openTestDB()
	p := newSQLitePersistence(db)
	testBrowsers := []*browsers.Browser{
		{
			ID:        "1",
			Name:      "first",
			Dapp:      true,
			Timestamp: 10,
		},
		{
			ID:        "2",
			Name:      "second",
			Dapp:      true,
			Timestamp: 50,
		},
		{
			ID:           "3",
			Name:         "third",
			Dapp:         true,
			Timestamp:    100,
			HistoryIndex: 0,
			History:      []string{"zero"},
		},
	}
	for i := 0; i < len(testBrowsers); i++ {
		require.NoError(t, p.AddBrowser(*testBrowsers[i]))
	}

	sort.Slice(testBrowsers, func(i, j int) bool {
		return testBrowsers[i].Timestamp > testBrowsers[j].Timestamp
	})

	rst, err := p.GetBrowsers()
	require.NoError(t, err)
	require.Equal(t, testBrowsers, rst)
}

func TestBrowsersHistoryIncluded(t *testing.T) {
	db, _ := openTestDB()
	p := newSQLitePersistence(db)
	browser := &browsers.Browser{
		ID:           "1",
		Name:         "first",
		Dapp:         true,
		Timestamp:    10,
		HistoryIndex: 1,
		History:      []string{"one", "two"},
	}
	require.NoError(t, p.AddBrowser(*browser))
	rst, err := p.GetBrowsers()
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, browser, rst[0])
}

func TestBrowsersReplaceOnUpdate(t *testing.T) {
	db, _ := openTestDB()
	p := newSQLitePersistence(db)
	browser := &browsers.Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: 10,
		History:   []string{"one", "two"},
	}
	require.NoError(t, p.AddBrowser(*browser))
	browser.Dapp = false
	browser.History = []string{"one", "three"}
	browser.Timestamp = 107
	require.NoError(t, p.AddBrowser(*browser))
	rst, err := p.GetBrowsers()
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, browser, rst[0])
}

func TestDeleteBrowser(t *testing.T) {
	db, _ := openTestDB()
	p := newSQLitePersistence(db)
	browser := &browsers.Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: 10,
		History:   []string{"one", "two"},
	}

	require.NoError(t, p.AddBrowser(*browser))
	rst, err := p.GetBrowsers()
	require.NoError(t, err)
	require.Len(t, rst, 1)

	require.NoError(t, p.DeleteBrowser(browser.ID))
	rst, err = p.GetBrowsers()
	require.NoError(t, err)
	require.Len(t, rst, 0)
}
