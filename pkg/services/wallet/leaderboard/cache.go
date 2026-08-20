package leaderboard

import "sync"

type PageCache struct {
	dataMutex sync.RWMutex
	lastPage  LeaderboardPage
}

func NewPageCache() *PageCache {
	return &PageCache{
		lastPage: LeaderboardPage{},
	}
}

func (c *PageCache) Clear() {
	c.dataMutex.Lock()
	defer c.dataMutex.Unlock()

	c.lastPage = LeaderboardPage{}
}

func (c *PageCache) GetLastPage() LeaderboardPage {
	c.dataMutex.RLock()
	defer c.dataMutex.RUnlock()

	return c.lastPage
}

func (c *PageCache) UpdateLastPage(page *LeaderboardPage) {
	c.dataMutex.Lock()
	defer c.dataMutex.Unlock()

	c.lastPage.Page = page.Page
	c.lastPage.PageSize = page.PageSize
	c.lastPage.SortOrder = page.SortOrder
	c.lastPage.Currency = page.Currency

}
