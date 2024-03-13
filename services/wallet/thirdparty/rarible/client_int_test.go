package rarible

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestRaribleClientIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RaribleClientIntegrationSuite))
}

type RaribleClientIntegrationSuite struct {
	suite.Suite

	client *Client
}

func (s *RaribleClientIntegrationSuite) SetupTest() {
	mainnetKey := os.Getenv("STATUS_BUILD_RARIBLE_MAINNET_API_KEY")
	if mainnetKey == "" {
		mainnetKey = os.Getenv("STATUS_RUNTIME_RARIBLE_MAINNET_API_KEY")
	}
	testnetKey := os.Getenv("STATUS_BUILD_RARIBLE_TESTNET_API_KEY")
	if testnetKey == "" {
		testnetKey = os.Getenv("STATUS_RUNTIME_RARIBLE_TESTNET_API_KEY")
	}

	s.client = NewClient(mainnetKey, testnetKey)
}

func (s *RaribleClientIntegrationSuite) TestAPIKeysAvailable() {
	// TODO #13953: Enable for nightly runs
	s.T().Skip("integration test")

	assert.NotEmpty(s.T(), s.client.mainnetAPIKey)
	assert.NotEmpty(s.T(), s.client.testnetAPIKey)
}

func (s *RaribleClientIntegrationSuite) TestSearchCollections() {
	// TODO #13953: Enable for nightly runs
	s.T().Skip("integration test")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections, err := s.client.SearchCollections(
		ctx,
		walletCommon.ChainID(walletCommon.EthereumMainnet),
		"CryptoKitties",
		thirdparty.FetchFromStartCursor,
		10,
	)
	s.Require().NoError(err)
	s.Require().NotNil(collections)
	s.Require().NotEmpty(collections.Items)
}

func (s *RaribleClientIntegrationSuite) TestSearchCollectibles() {
	// TODO #13953: Enable for nightly runs
	s.T().Skip("integration test")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectibles, err := s.client.SearchCollectibles(
		ctx,
		walletCommon.ChainID(walletCommon.EthereumMainnet),
		[]common.Address{common.HexToAddress("0x06012c8cf97BEaD5deAe237070F9587f8E7A266d")},
		"Furbeard",
		thirdparty.FetchFromStartCursor,
		10,
	)
	s.Require().NoError(err)
	s.Require().NotNil(collectibles)
	s.Require().NotEmpty(collectibles.Items)
}
