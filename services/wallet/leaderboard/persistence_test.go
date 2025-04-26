package leaderboard

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

type MarketDataPersistenceTestSuite struct {
	suite.Suite
	db                    *sql.DB
	cleanup               func() error
	marketDataPersistence MarketDataPersistenceInterface
}

func (s *MarketDataPersistenceTestSuite) SetupTest() {
	memDb, cleanup, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "market-data-tests")
	s.Require().NoError(err)
	s.db = memDb
	s.cleanup = cleanup
	s.marketDataPersistence = NewPersistance(memDb)
}

func (s *MarketDataPersistenceTestSuite) TearDownTest() {
	if s.cleanup != nil {
		err := s.cleanup()
		require.NoError(s.T(), err)
	}
}

func TestMarketDataPersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(MarketDataPersistenceTestSuite))
}

func (s *MarketDataPersistenceTestSuite) TestUpsertAndGetCryptocurrencies() {
	cryptos, err := s.marketDataPersistence.GetCryptocurrencies()
	s.Require().NoError(err)
	s.Require().Equal(0, len(cryptos))

	err = s.marketDataPersistence.UpsertCryptocurrencies(mockCrypto)
	s.Require().NoError(err)
	cryptos, err = s.marketDataPersistence.GetCryptocurrencies()
	s.Require().NoError(err)
	s.Require().Equal(len(mockCrypto), len(cryptos))
	for i, crypto := range cryptos {
		s.Require().Equal(mockCrypto[i].ID, crypto.ID)
	}
}

func (s *MarketDataPersistenceTestSuite) TestDeleteCryptocurrencies() {
	err := s.marketDataPersistence.UpsertCryptocurrencies(mockCrypto)
	s.Require().NoError(err)
	cryptos, err := s.marketDataPersistence.GetCryptocurrencies()
	s.Require().NoError(err)
	s.Require().Equal(len(mockCrypto), len(cryptos))

	err = s.marketDataPersistence.DeleteCryptocurrencies([]string{mockCrypto[0].ID})
	s.Require().NoError(err)

	cryptos, err = s.marketDataPersistence.GetCryptocurrencies()
	s.Require().NoError(err)
	s.Require().Equal(len(mockCrypto)-1, len(cryptos))
	for i, crypto := range cryptos {
		s.Require().Equal(mockCrypto[i+1].ID, crypto.ID)
	}
}
