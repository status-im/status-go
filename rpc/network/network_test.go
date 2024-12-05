package network_test

import (
	"database/sql"
	"github.com/status-im/status-go/api"
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/rpc/network/db"
	"github.com/status-im/status-go/rpc/network/testutil"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/suite"
)

type NetworkManagerTestSuite struct {
	suite.Suite
	db      *sql.DB
	cleanup func()
	manager *network.Manager
}

func (s *NetworkManagerTestSuite) SetupTest() {

	testDb, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "wallet-network-tests")
	s.Require().NoError(err)
	s.db = testDb
	s.cleanup = func() { s.Require().NoError(cleanup()) }
	s.manager = network.NewManager(testDb)
	persistence := db.NewNetworksPersistence(testDb)

	// Use testutil to initialize networks
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api.MainnetChainID, "Infura Mainnet", params.UserProviderType, true, "https://mainnet.infura.io"),
			testutil.CreateProvider(api.MainnetChainID, "Backup Mainnet", params.EmbeddedProxyProviderType, false, "https://backup.mainnet.provider.com"),
		}),
		*testutil.CreateNetwork(api.SepoliaChainID, "Sepolia Testnet", []params.RpcProvider{
			testutil.CreateProvider(api.SepoliaChainID, "Infura Sepolia", params.UserProviderType, true, "https://sepolia.infura.io"),
		}),
		*testutil.CreateNetwork(api.OptimismChainID, "Optimistic Ethereum", []params.RpcProvider{
			testutil.CreateProvider(api.OptimismChainID, "Infura Optimism", params.UserProviderType, true, "https://optimism.infura.io"),
			testutil.CreateProvider(api.OptimismChainID, "Backup Optimism", params.EmbeddedDirectProviderType, false, "https://backup.optimism.provider.com"),
		}),
	}
	err = persistence.SetNetworks(initNetworks)
	s.Require().NoError(err)
	s.assertDbNetworks(initNetworks)
}

func (s *NetworkManagerTestSuite) TearDownTest() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

func TestNetworkManagerTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkManagerTestSuite))
}

// Helper functions

func (s *NetworkManagerTestSuite) assertDbNetworks(expectedNetworks []params.Network) {
	// Convert []model.Network to []*model.Network
	expectedNetworksPtr := testutil.ConvertNetworksToPointers(expectedNetworks)

	// Fetch saved networks
	savedNetworks, err := s.manager.GetAll()
	s.Require().NoError(err)

	// Compare the lists
	testutil.CompareNetworksList(s.T(), expectedNetworksPtr, savedNetworks)
}

func (s *NetworkManagerTestSuite) TestInitNetworksWithChangedAuth() {
	// Change auth token for a provider
	updatedNetworks := []params.Network{
		*testutil.CreateNetwork(api.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api.MainnetChainID, "Infura Mainnet", params.UserProviderType, true, "https://mainnet.infura.io"),
			{
				Name:      "Backup Mainnet",
				ChainID:   api.MainnetChainID,
				Type:      params.EmbeddedProxyProviderType,
				Enabled:   false,
				URL:       "https://backup.mainnet.provider.com",
				AuthType:  params.TokenAuth,
				AuthToken: "new-token",
			},
		}),
	}

	// Re-initialize and assert
	err := s.manager.InitEmbeddedNetworks(updatedNetworks)
	s.Require().NoError(err)
	s.assertDbNetworks(updatedNetworks)
}

func (s *NetworkManagerTestSuite) TestUserAddsCustomProviders() {
	// Adding custom providers
	customProviders := []params.RpcProvider{
		testutil.CreateProvider(api.MainnetChainID, "CustomProvider1", params.UserProviderType, true, "https://custom1.example.com"),
		testutil.CreateProvider(api.MainnetChainID, "CustomProvider2", params.UserProviderType, false, "https://custom2.example.com"),
	}
	err := s.manager.SetUserRpcProviders(api.MainnetChainID, customProviders)
	s.Require().NoError(err)

	// Assert providers
	foundNetwork := s.manager.Find(api.MainnetChainID)
	s.Require().NotNil(foundNetwork)
	expectedProviders := append(customProviders, networkhelper.GetEmbeddedProviders(foundNetwork.RpcProviders)...)
	testutil.CompareProvidersList(s.T(), expectedProviders, foundNetwork.RpcProviders)
}

func (s *NetworkManagerTestSuite) TestInitNetworksKeepsUserProviders() {
	// Add custom providers
	customProviders := []params.RpcProvider{
		testutil.CreateProvider(api.MainnetChainID, "CustomProvider1", params.UserProviderType, true, "https://custom1.example.com"),
		testutil.CreateProvider(api.MainnetChainID, "CustomProvider2", params.UserProviderType, false, "https://custom2.example.com"),
	}
	err := s.manager.SetUserRpcProviders(api.MainnetChainID, customProviders)
	s.Require().NoError(err)

	// Re-initialize networks
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api.MainnetChainID, "Infura Mainnet", params.UserProviderType, true, "https://mainnet.infura.io"),
		}),
	}
	err = s.manager.InitEmbeddedNetworks(initNetworks)
	s.Require().NoError(err)

	// Check that custom providers are retained
	foundNetwork := s.manager.Find(api.MainnetChainID)
	s.Require().NotNil(foundNetwork)
	expectedProviders := append(customProviders, networkhelper.GetEmbeddedProviders(initNetworks[0].RpcProviders)...)
	testutil.CompareProvidersList(s.T(), expectedProviders, foundNetwork.RpcProviders)
}

func (s *NetworkManagerTestSuite) TestLegacyFieldPopulation() {
	// Create initial test networks with various providers
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api.MainnetChainID, "DirectProvider1", params.EmbeddedDirectProviderType, true, "https://direct1.ethereum.io"),
			testutil.CreateProvider(api.MainnetChainID, "DirectProvider2", params.EmbeddedDirectProviderType, true, "https://direct2.ethereum.io"),
			testutil.CreateProvider(api.MainnetChainID, "ProxyProvider1", params.EmbeddedProxyProviderType, true, "https://proxy1.ethereum.io"),
			testutil.CreateProvider(api.MainnetChainID, "ProxyProvider2", params.EmbeddedProxyProviderType, true, "https://proxy2.ethereum.io"),
			testutil.CreateProvider(api.MainnetChainID, "ProxyProvider3", params.EmbeddedProxyProviderType, true, "https://proxy3.ethereum.io"),
			testutil.CreateProvider(api.MainnetChainID, "UserProvider1", params.UserProviderType, true, "https://user1.ethereum.io"),
			testutil.CreateProvider(api.MainnetChainID, "UserProvider2", params.UserProviderType, true, "https://user2.ethereum.io"),
		}),
	}

	// Add the network to the database
	persistence := db.NewNetworksPersistence(s.db)
	err := persistence.SetNetworks(initNetworks)
	s.Require().NoError(err)

	// Fetch networks and verify legacy field population
	networks, err := persistence.GetNetworks(false, nil)
	s.Require().NoError(err)
	s.Require().Len(networks, 1)

	network := networks[0]

	// Check legacy fields
	s.Equal("https://direct1.ethereum.io", network.OriginalRPCURL)
	s.Equal("https://direct2.ethereum.io", network.OriginalFallbackURL)
	s.Equal("https://user1.ethereum.io", network.RPCURL)
	s.Equal("https://user2.ethereum.io", network.FallbackURL)
	s.Equal("https://proxy1.ethereum.io", network.DefaultRPCURL)
	s.Equal("https://proxy2.ethereum.io", network.DefaultFallbackURL)
	s.Equal("https://proxy3.ethereum.io", network.DefaultFallbackURL2)
}

func (s *NetworkManagerTestSuite) TestLegacyFieldPopulationWithoutUserProviders() {
	// Create a test network with only EmbeddedDirect and EmbeddedProxy providers
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api.SepoliaChainID, "Sepolia Testnet", []params.RpcProvider{
			testutil.CreateProvider(api.SepoliaChainID, "DirectProvider1", params.EmbeddedDirectProviderType, true, "https://direct1.sepolia.io"),
			testutil.CreateProvider(api.SepoliaChainID, "ProxyProvider1", params.EmbeddedProxyProviderType, true, "https://proxy1.sepolia.io"),
			testutil.CreateProvider(api.SepoliaChainID, "ProxyProvider2", params.EmbeddedProxyProviderType, true, "https://proxy2.sepolia.io"),
		}),
	}

	// Add the network to the database
	persistence := db.NewNetworksPersistence(s.db)
	err := persistence.SetNetworks(initNetworks)
	s.Require().NoError(err)

	// Fetch networks and verify legacy field population
	networks, err := persistence.GetNetworks(false, nil)
	s.Require().NoError(err)
	s.Require().Len(networks, 1)

	network := networks[0]

	// Check legacy fields
	s.Equal("https://direct1.sepolia.io", network.OriginalRPCURL)
	s.Empty(network.OriginalFallbackURL)                  // No second EmbeddedDirect provider
	s.Equal("https://direct1.sepolia.io", network.RPCURL) // Defaults to OriginalRPCURL
	s.Empty(network.FallbackURL)                          // Defaults to OriginalFallbackURL, which is empty
	s.Equal("https://proxy1.sepolia.io", network.DefaultRPCURL)
	s.Equal("https://proxy2.sepolia.io", network.DefaultFallbackURL)
	s.Empty(network.DefaultFallbackURL2) // No third Proxy provider
}
