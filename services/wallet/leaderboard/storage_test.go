package leaderboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var mockCrypto = []Cryptocurrency{
	{
		ID:     1,
		Name:   "Bitcoin",
		Symbol: "BTC",
		Quote: Quote{
			USD: QuoteDetails{
				Price:            88260.1619455892,
				Volume24h:        23933002554.825462,
				MarketCap:        1751201028999.603,
				PercentChange24h: 3.85381097,
			},
		},
	},
	{
		ID:     2,
		Name:   "Ethereum",
		Symbol: "ETH",
		Quote: Quote{
			USD: QuoteDetails{
				Price:            2031.451,
				Volume24h:        13904210482.120,
				MarketCap:        245320045879.50,
				PercentChange24h: -1.4823,
			},
		},
	},
	{
		ID:     3,
		Name:   "Ripple",
		Symbol: "XRP",
		Quote: Quote{
			USD: QuoteDetails{
				Price:            0.5113,
				Volume24h:        1240345020.84,
				MarketCap:        27320420359.92,
				PercentChange24h: 0.0275,
			},
		},
	},
	{
		ID:     4,
		Name:   "Litecoin",
		Symbol: "LTC",
		Quote: Quote{
			USD: QuoteDetails{
				Price:            92.38,
				Volume24h:        637203095.76,
				MarketCap:        6789040512.34,
				PercentChange24h: -0.5721,
			},
		},
	},
	{
		ID:     5,
		Name:   "Cardano",
		Symbol: "ADA",
		Quote: Quote{
			USD: QuoteDetails{
				Price:            0.3742,
				Volume24h:        238203495.91,
				MarketCap:        13200401234.55,
				PercentChange24h: 0.0041,
			},
		},
	},
}

var mockPriceData = map[string]PriceData{
	"BTC": {
		Price:            88260.1619455892,
		Volume24h:        23933002554.825462,
		PercentChange24h: 3.85381097,
	},
	"ETH": {
		Price:            2031.451,
		Volume24h:        13904210482.120,
		PercentChange24h: -1.4823,
	},
	"ADA": {
		Price:            0.3742,
		Volume24h:        238203495.91,
		PercentChange24h: 0.0041,
	},
}

func TestGetLeaderboardPageErrors(t *testing.T) {
	s := NewDataStorage()
	s.UpdateCryptoData(mockCrypto)

	{
		_, err := s.GetLeaderboardPage(-1, 10, -1)
		require.Error(t, err)
	}

	{
		_, err := s.GetLeaderboardPage(1, 0, -1)
		require.Error(t, err)
	}

	{
		_, err := s.GetLeaderboardPage(100, 100, -1)
		require.Error(t, err)
	}
}

func TestGetLeaderboardPage(t *testing.T) {
	s := NewDataStorage()
	s.UpdateCryptoData(mockCrypto)

	{
		rst, err := s.GetLeaderboardPage(0, 3, -1)
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 0, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 3, len(rst.Data))
		require.Equal(t, mockCrypto[0], rst.Data[0])
		require.Equal(t, mockCrypto[1], rst.Data[1])
		require.Equal(t, mockCrypto[2], rst.Data[2])
	}

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1)
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 1, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 2, len(rst.Data))
		require.Equal(t, mockCrypto[3], rst.Data[0])
		require.Equal(t, mockCrypto[4], rst.Data[1])
	}
}

func TestGetLeaderboardPageEmpty(t *testing.T) {
	s := NewDataStorage()

	{
		rst, err := s.GetLeaderboardPage(0, 3, -1)
		require.NoError(t, err)
		require.Equal(t, 0, rst.TotalCount)
		require.Equal(t, 0, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 0, len(rst.Data))
	}
}

func TestGetLeaderboardPageWithUpdatedPrices(t *testing.T) {
	s := NewDataStorage()
	s.UpdateCryptoData(mockCrypto)
	s.UpdatePriceData(mockPriceData)

	{
		rst, err := s.GetLeaderboardPage(0, 3, -1)
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 0, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 3, len(rst.Data))
		require.Equal(t, mockCrypto[2], rst.Data[2])
		require.Equal(t, mockPriceData["BTC"].Price, rst.Data[0].Quote.USD.Price)
		require.Equal(t, mockPriceData["BTC"].Volume24h, rst.Data[0].Quote.USD.Volume24h)
		require.Equal(t, mockPriceData["BTC"].PercentChange24h, rst.Data[0].Quote.USD.PercentChange24h)
		require.Equal(t, mockPriceData["ETH"].Price, rst.Data[1].Quote.USD.Price)
		require.Equal(t, mockPriceData["ETH"].Volume24h, rst.Data[1].Quote.USD.Volume24h)
		require.Equal(t, mockPriceData["ETH"].PercentChange24h, rst.Data[1].Quote.USD.PercentChange24h)
	}
}

func TestGetLeaderboardPagePrices(t *testing.T) {
	s := NewDataStorage()
	s.UpdateCryptoData(mockCrypto)
	s.UpdatePriceData(mockPriceData)

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1)
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 1, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 2, len(rst.Data))
	}

	{
		page := LeaderboardPage{}
		rst := s.GetLeaderboardPagePrices(page)
		require.Nil(t, rst)
	}

	{
		page := LeaderboardPage{
			Page:      1,
			PageSize:  3,
			SortOrder: -1,
		}
		rst := s.GetLeaderboardPagePrices(page)
		require.NotNil(t, rst)
		require.Equal(t, 1, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 1, len(rst.Data)) // Only one crypto price (out of 2) was updated on this page
	}
}
