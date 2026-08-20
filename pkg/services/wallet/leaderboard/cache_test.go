package leaderboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitialValue(t *testing.T) {
	cache := NewPageCache()

	cachedPage := cache.GetLastPage()

	require.Equal(t, 0, cachedPage.Page)
	require.Equal(t, 0, cachedPage.PageSize)
	require.Equal(t, 0, cachedPage.SortOrder)
}

func TestUpdateCache(t *testing.T) {
	cache := NewPageCache()

	page := &LeaderboardPage{
		Page:      1,
		PageSize:  10,
		SortOrder: 2,
	}

	cache.UpdateLastPage(page)

	cachedPage := cache.GetLastPage()

	require.Equal(t, page.Page, cachedPage.Page)
	require.Equal(t, page.PageSize, cachedPage.PageSize)
	require.Equal(t, page.SortOrder, cachedPage.SortOrder)
}

func TestClearCache(t *testing.T) {
	cache := NewPageCache()

	page := &LeaderboardPage{
		Page:      1,
		PageSize:  10,
		SortOrder: 2,
	}

	cache.UpdateLastPage(page)
	cache.Clear()

	cachedPage := cache.GetLastPage()

	require.Equal(t, 0, cachedPage.Page)
	require.Equal(t, 0, cachedPage.PageSize)
	require.Equal(t, 0, cachedPage.SortOrder)
}
