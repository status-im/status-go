package leaderboard

import (
	"database/sql"
	"testing"

	sq "github.com/Masterminds/squirrel"
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
	"bitcoin": {
		Price:            79451,
		MarketCap:        1577274527423,
		Volume24h:        78498730801,
		PercentChange24h: 6.49692,
	},
	"ethereum": {
		Price:            1576.35,
		MarketCap:        190254450318,
		Volume24h:        38689205530,
		PercentChange24h: 9.82681,
	},
	"cardano": {
		Price:            0.3742,
		MarketCap:        13200401234.55,
		Volume24h:        238203495.91,
		PercentChange24h: 0.0041,
	},
}

func insertCryptoDataToDatabase(t *testing.T, db *sql.DB, cryptoData []Cryptocurrency) {
	t.Helper()

	persistence := NewPersistance(db)
	err := persistence.UpsertCryptocurrencies(cryptoData)
	require.NoError(t, err)
}

// Helper function to verify crypto price data
func verifyCryptoPriceData(t *testing.T, expected PriceData, actual Cryptocurrency) {
	t.Helper()
	require.Equal(t, expected.Price, actual.CurrentPrice)
	require.Equal(t, expected.MarketCap, actual.MarketCap)
	require.Equal(t, expected.Volume24h, actual.TotalVolume)
	require.Equal(t, expected.PercentChange24h, actual.PriceChangePercentage24h)
}

func TestGetLeaderboardPageErrors(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	s := NewDataStorage(db)
	s.Start()
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
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	s := NewDataStorage(db)
	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")

	{
		_, err := s.GetLeaderboardPage(0, 3, -1, "usd")
		require.Error(t, err) // Page 0 is invalid
	}
	{
		rst, err := s.GetLeaderboardPage(1, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 1, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, "usd", rst.Currency)
		require.Equal(t, 3, len(rst.Data))
		require.Equal(t, mockCrypto[0], rst.Data[0])
		require.Equal(t, mockCrypto[1], rst.Data[1])
		require.Equal(t, mockCrypto[2], rst.Data[2])
	}

	{
		rst, err := s.GetLeaderboardPage(2, 3, -1, "eur")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 2, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, "eur", rst.Currency)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 2, len(rst.Data))
		require.Equal(t, mockCrypto[3], rst.Data[0])
		require.Equal(t, mockCrypto[4], rst.Data[1])
	}
}

func TestGetLeaderboardPageEmpty(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	s := NewDataStorage(db)

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 0, rst.TotalCount)
		require.Equal(t, 1, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, "usd", rst.Currency)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 0, len(rst.Data))
	}
}

func TestGetLeaderboardPageWithUpdatedPrices(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	s := NewDataStorage(db)
	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")
	s.UpdatePriceDataWithEtag(mockPriceData, "test-etag")

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 1, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, "usd", rst.Currency)
		require.Equal(t, 3, len(rst.Data))
		require.Equal(t, mockCrypto[2], rst.Data[2])
		verifyCryptoPriceData(t, mockPriceData["bitcoin"], rst.Data[0])
		verifyCryptoPriceData(t, mockPriceData["ethereum"], rst.Data[1])
	}
}

func TestGetLeaderboardPagePrices(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	s := NewDataStorage(db)
	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")
	s.UpdatePriceDataWithEtag(mockPriceData, "test-etag")

	{
		rst, err := s.GetLeaderboardPage(2, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount)
		require.Equal(t, 2, rst.Page)
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
		rst := s.GetLeaderboardPagePrices(LeaderboardPage{
			Page:      2,
			PageSize:  3,
			SortOrder: -1,
			Currency:  "usd",
		})
		require.NotNil(t, rst)
		require.Equal(t, 2, rst.Page)
		require.Equal(t, 3, rst.PageSize)
		require.Equal(t, "usd", rst.Currency)
		require.Equal(t, -1, rst.SortOrder)
		require.Equal(t, 1, len(rst.Data)) // Only one crypto price (out of 2) was updated on this page
	}
}

func TestGetLeaderboardPageDatabase(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	s := NewDataStorage(db)

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 0, rst.TotalCount)
	}

	insertCryptoDataToDatabase(t, db, []Cryptocurrency{
		{
			ID:                       "bitcoin",
			Symbol:                   "btc",
			Name:                     "Bitcoin",
			CurrentPrice:             79451,
			MarketCap:                1577274527423,
			TotalVolume:              78498730801,
			PriceChangePercentage24h: 6.49692,
		},
		{
			ID:                       "ethereum",
			Symbol:                   "eth",
			Name:                     "Ethereum",
			CurrentPrice:             1576.35,
			MarketCap:                190254450318,
			TotalVolume:              38689205530,
			PriceChangePercentage24h: 9.82681,
		},
		{
			ID:                       "samba",
			Symbol:                   "smb",
			Name:                     "Samba",
			CurrentPrice:             2376.35,
			MarketCap:                194454450318,
			TotalVolume:              386669205530,
			PriceChangePercentage24h: 5.5124,
		},
	})

	s.Start()

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 3, rst.TotalCount) // Only from database
	}

	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")

	{
		rst, err := s.GetLeaderboardPage(1, 3, -1, "usd")
		require.NoError(t, err)
		require.Equal(t, 5, rst.TotalCount) // From mocked data
	}

	{
		rows, err := sq.Select("id").From("market_data").RunWith(db).Query()
		require.NoError(t, err)

		var ids []string
		for rows.Next() {
			id := ""
			err := rows.Scan(
				&id,
			)
			require.NoError(t, err)
			ids = append(ids, id)
		}
		rows.Close()

		// Samba should be deleted because it doesn't exist in the mocked data
		require.Equal(t, 5, len(ids))

		actualIDs := make(map[string]bool)
		for _, id := range ids {
			actualIDs[id] = true
		}

		for _, crypto := range mockCrypto {
			require.True(t, actualIDs[crypto.ID], "Expected ID %s was missing from results", crypto.ID)
		}
	}
}

func TestGetCombinedDataWithPriceUpdates(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	s := NewDataStorage(db)

	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")
	combined := s.GetCombinedData()
	require.Equal(t, len(mockCrypto), len(combined))
	require.Equal(t, mockCrypto[0].CurrentPrice, combined[0].CurrentPrice)

	priceDataWithUpdates := map[string]PriceData{
		"bitcoin": {
			Price:            80000.0,
			MarketCap:        1600000000000,
			Volume24h:        80000000000,
			PercentChange24h: 7.5,
		},
		"ethereum": {
			Price:            1600.0,
			MarketCap:        195000000000,
			Volume24h:        40000000000,
			PercentChange24h: 10.0,
		},
	}
	s.UpdatePriceDataWithEtag(priceDataWithUpdates, "price-etag")

	combined = s.GetCombinedData()
	require.Equal(t, len(mockCrypto), len(combined))

	require.Equal(t, priceDataWithUpdates["bitcoin"].Price, combined[0].CurrentPrice)
	require.Equal(t, priceDataWithUpdates["bitcoin"].MarketCap, combined[0].MarketCap)
	require.Equal(t, priceDataWithUpdates["bitcoin"].Volume24h, combined[0].TotalVolume)
	require.Equal(t, priceDataWithUpdates["bitcoin"].PercentChange24h, combined[0].PriceChangePercentage24h)

	require.Equal(t, priceDataWithUpdates["ethereum"].Price, combined[1].CurrentPrice)
	require.Equal(t, priceDataWithUpdates["ethereum"].MarketCap, combined[1].MarketCap)
	require.Equal(t, priceDataWithUpdates["ethereum"].Volume24h, combined[1].TotalVolume)
	require.Equal(t, priceDataWithUpdates["ethereum"].PercentChange24h, combined[1].PriceChangePercentage24h)

	require.Equal(t, mockCrypto[2].CurrentPrice, combined[2].CurrentPrice)
	require.Equal(t, mockCrypto[2].MarketCap, combined[2].MarketCap)

	priceDataWithZeros := map[string]PriceData{
		"ripple": {Price: 2.0},
	}
	s.UpdatePriceDataWithEtag(priceDataWithZeros, "price-etag-2")

	combined = s.GetCombinedData()
	require.Equal(t, priceDataWithZeros["ripple"].Price, combined[3].CurrentPrice)
	require.Equal(t, mockCrypto[3].MarketCap, combined[3].MarketCap)
	require.Equal(t, mockCrypto[3].TotalVolume, combined[3].TotalVolume)
	require.Equal(t, mockCrypto[3].PriceChangePercentage24h, combined[3].PriceChangePercentage24h)
}

func TestGetLeaderboardPagePricesWithIDLookup(t *testing.T) {
	db, cleanup := setupTestWalletDB(t)
	defer cleanup()
	s := NewDataStorage(db)
	s.UpdateCryptoDataWithEtag(mockCrypto, "test-etag")

	priceDataWithIDs := map[string]PriceData{
		"bitcoin": {
			Price:            81000.0,
			MarketCap:        1620000000000,
			Volume24h:        82000000000,
			PercentChange24h: 8.0,
		},
		"ripple": {
			Price:            1.95,
			MarketCap:        110000000000,
			Volume24h:        9500000000,
			PercentChange24h: 13.0,
		},
	}
	s.UpdatePriceDataWithEtag(priceDataWithIDs, "price-etag")

	rst := s.GetLeaderboardPagePrices(LeaderboardPage{
		Page:      1,
		PageSize:  3,
		SortOrder: -1,
		Currency:  "usd",
	})

	require.NotNil(t, rst)
	require.Equal(t, 1, rst.Page)
	require.Equal(t, 3, rst.PageSize)
	require.Equal(t, "usd", rst.Currency)
	require.Equal(t, -1, rst.SortOrder)
	require.Equal(t, 1, len(rst.Data))

	require.Equal(t, "bitcoin", rst.Data[0].ID)
	require.Equal(t, priceDataWithIDs["bitcoin"].Price, rst.Data[0].Price)
	require.Equal(t, priceDataWithIDs["bitcoin"].MarketCap, rst.Data[0].MarketCap)
	require.Equal(t, priceDataWithIDs["bitcoin"].Volume24h, rst.Data[0].Volume24h)

	rst2 := s.GetLeaderboardPagePrices(LeaderboardPage{
		Page:      2,
		PageSize:  3,
		SortOrder: -1,
		Currency:  "usd",
	})

	require.NotNil(t, rst2)
	require.Equal(t, 1, len(rst2.Data))
	require.Equal(t, "ripple", rst2.Data[0].ID)
	require.Equal(t, priceDataWithIDs["ripple"].Price, rst2.Data[0].Price)
}
