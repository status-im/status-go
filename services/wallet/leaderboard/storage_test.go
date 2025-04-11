package leaderboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var mockCrypto = []Cryptocurrency{
	{
		ID:                       "bitcoin",
		Name:                     "Bitcoin",
		Symbol:                   "btc",
		Image:                    "https://coin-images.coingecko.com/coins/images/1/large/bitcoin.png?1696501400",
		CurrentPrice:             79451,
		MarketCap:                1577274527423,
		TotalVolume:              78498730801,
		PriceChangePercentage24h: 6.49692,
	},
	{
		ID:                       "ethereum",
		Name:                     "Ethereum",
		Symbol:                   "eth",
		Image:                    "https://coin-images.coingecko.com/coins/images/279/large/ethereum.png?1696501628",
		CurrentPrice:             1576.35,
		MarketCap:                190254450318,
		TotalVolume:              38689205530,
		PriceChangePercentage24h: 9.82681,
	},
	{
		ID:                       "tether",
		Name:                     "Tether",
		Symbol:                   "usdt",
		Image:                    "https://coin-images.coingecko.com/coins/images/325/large/Tether.png?1696501661",
		CurrentPrice:             0.999637,
		MarketCap:                144139703405,
		TotalVolume:              119147509139,
		PriceChangePercentage24h: 0.08216,
	},
	{
		ID:                       "ripple",
		Name:                     "XRP",
		Symbol:                   "xrp",
		Image:                    "https://coin-images.coingecko.com/coins/images/44/large/xrp-symbol-white-128.png?1696501442",
		CurrentPrice:             1.86,
		MarketCap:                108451149043,
		TotalVolume:              9387214286,
		PriceChangePercentage24h: 12.25473,
	},
	{
		ID:                       "cardano",
		Name:                     "Cardano",
		Symbol:                   "ada",
		Image:                    "https://coin-images.coingecko.com/coins/images/975/large/cardano.png?1696502090",
		CurrentPrice:             0.3742,
		MarketCap:                13200401234.55,
		TotalVolume:              238203495.91,
		PriceChangePercentage24h: 0.0041,
	},
}

var mockPriceData = map[string]PriceData{
	"btc": {
		Price:            79451,
		PercentChange24h: 6.49692,
	},
	"eth": {
		Price:            1576.35,
		PercentChange24h: 9.82681,
	},
	"ada": {
		Price:            0.3742,
		PercentChange24h: 0.0041,
	},
}

// Helper function to verify crypto price data
func verifyCryptoPriceData(t *testing.T, expected PriceData, actual Cryptocurrency) {
	t.Helper()
	require.Equal(t, expected.Price, actual.CurrentPrice)
	require.Equal(t, expected.PercentChange24h, actual.PriceChangePercentage24h)
}

func TestGetLeaderboardPageErrors(t *testing.T) {
	s := NewDataStorage()
	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")

	{
		_, err := s.GetLeaderboardPage(-1, 10, -1, "usd")
		require.Error(t, err)
	}

	{
		_, err := s.GetLeaderboardPage(1, 0, -1, "usd")
		require.Error(t, err)
	}

	{
		_, err := s.GetLeaderboardPage(100, 100, -1, "usd")
		require.Error(t, err)
	}
}

func TestGetLeaderboardPage(t *testing.T) {
	s := NewDataStorage()
	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")

	{
		rst, err := s.GetLeaderboardPage(0, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 0, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, "usd", rst.Currency)
		require.Equal(t, 3, len(rst.Data))
		require.Equal(t, mockCrypto[0], rst.Data[0])
		require.Equal(t, mockCrypto[1], rst.Data[1])
		require.Equal(t, mockCrypto[2], rst.Data[2])
	}

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1, "eur")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 1, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, "eur", rst.Currency)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 2, len(rst.Data))
		require.Equal(t, mockCrypto[3], rst.Data[0])
		require.Equal(t, mockCrypto[4], rst.Data[1])
	}
}

func TestGetLeaderboardPageEmpty(t *testing.T) {
	s := NewDataStorage()

	{
		rst, err := s.GetLeaderboardPage(0, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 0, rst.TotalCount)
		require.Equal(t, 0, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, "usd", rst.Currency)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 0, len(rst.Data))
	}
}

func TestGetLeaderboardPageWithUpdatedPrices(t *testing.T) {
	s := NewDataStorage()
	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")
	s.UpdatePriceDataWithEtag(mockPriceData, "test-etag")

	{
		rst, err := s.GetLeaderboardPage(0, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 0, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, "usd", rst.Currency)
		require.Equal(t, 3, len(rst.Data))
		require.Equal(t, mockCrypto[2], rst.Data[2])
		verifyCryptoPriceData(t, mockPriceData["btc"], rst.Data[0])
		verifyCryptoPriceData(t, mockPriceData["eth"], rst.Data[1])
	}
}

func TestGetLeaderboardPagePrices(t *testing.T) {
	s := NewDataStorage()
	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")
	s.UpdatePriceDataWithEtag(mockPriceData, "test-etag")

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 1, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, "usd", rst.Currency)
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
