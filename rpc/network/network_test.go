package network_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/suite"

	api_common "github.com/status-im/status-go/api/common"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/internal/security"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/params/networkhelper"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/rpc/network/db"
	"github.com/status-im/status-go/rpc/network/testutil"
	"github.com/status-im/status-go/t/helpers"
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
	s.manager = network.NewManager(testDb, nil, nil, nil)
	persistence := db.NewNetworksPersistence(testDb)

	// Use testutil to initialize networks
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api_common.MainnetChainID, "Infura Mainnet", params.UserProviderType, true, security.NewSensitiveString("https://mainnet.infura.io")),
		}),
		*testutil.CreateNetwork(api_common.SepoliaChainID, "Sepolia Testnet", []params.RpcProvider{
			testutil.CreateProvider(api_common.SepoliaChainID, "Infura Sepolia", params.UserProviderType, true, security.NewSensitiveString("https://sepolia.infura.io")),
		}),
		*testutil.CreateNetwork(api_common.OptimismChainID, "Optimistic Ethereum", []params.RpcProvider{
			testutil.CreateProvider(api_common.OptimismChainID, "Infura Optimism", params.UserProviderType, true, security.NewSensitiveString("https://optimism.infura.io")),
		}),
		*testutil.CreateNetwork(api_common.OptimismSepoliaChainID, "Optimistic Sepolia", []params.RpcProvider{
			testutil.CreateProvider(api_common.OptimismSepoliaChainID, "Infura Optimism Sepolia", params.UserProviderType, true, security.NewSensitiveString("https://optimism-sepolia.infura.io")),
		}),
		*testutil.CreateNetwork(api_common.BaseChainID, "Base", []params.RpcProvider{
			testutil.CreateProvider(api_common.BaseChainID, "Infura Base", params.UserProviderType, true, security.NewSensitiveString("https://base.infura.io")),
		}),
		*testutil.CreateNetwork(api_common.BaseSepoliaChainID, "Base Sepolia", []params.RpcProvider{
			testutil.CreateProvider(api_common.BaseSepoliaChainID, "Infura Base Sepolia", params.UserProviderType, true, security.NewSensitiveString("https://base-sepolia.infura.io")),
		}),
	}
	// Make "Ethereum Mainnet" network not deactivatable
	initNetworks[0].IsDeactivatable = false

	// Make "Optimistic Ethereum" network inactive by default
	initNetworks[2].IsActive = false

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

func (s *NetworkManagerTestSuite) TestUserAddsCustomProviders() {
	// Adding custom providers
	customProviders := []params.RpcProvider{
		testutil.CreateProvider(api_common.MainnetChainID, "CustomProvider1", params.UserProviderType, true, security.NewSensitiveString("https://custom1.example.com")),
		testutil.CreateProvider(api_common.MainnetChainID, "CustomProvider2", params.UserProviderType, false, security.NewSensitiveString("https://custom2.example.com")),
	}
	err := s.manager.SetUserRpcProviders(api_common.MainnetChainID, customProviders)
	s.Require().NoError(err)

	// Assert providers
	foundNetwork := s.manager.Find(api_common.MainnetChainID)
	s.Require().NotNil(foundNetwork)
	expectedProviders := append(customProviders, networkhelper.GetEmbeddedProviders(foundNetwork.RpcProviders)...)
	testutil.CompareProvidersList(s.T(), expectedProviders, foundNetwork.RpcProviders)
}

func (s *NetworkManagerTestSuite) TestInitNetworksKeepsUserProviders() {
	// Add custom providers
	customProviders := []params.RpcProvider{
		testutil.CreateProvider(api_common.MainnetChainID, "CustomProvider1", params.UserProviderType, true, security.NewSensitiveString("https://custom1.example.com")),
		testutil.CreateProvider(api_common.MainnetChainID, "CustomProvider2", params.UserProviderType, false, security.NewSensitiveString("https://custom2.example.com")),
	}
	err := s.manager.SetUserRpcProviders(api_common.MainnetChainID, customProviders)
	s.Require().NoError(err)

	// Re-initialize networks
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api_common.MainnetChainID, "Infura Mainnet", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://mainnet.infura.io")),
		}),
	}
	err = s.manager.InitEmbeddedNetworks(initNetworks)
	s.Require().NoError(err)

	// Check that custom providers are retained
	foundNetwork := s.manager.Find(api_common.MainnetChainID)
	s.Require().NotNil(foundNetwork)
	expectedProviders := append(customProviders, networkhelper.GetEmbeddedProviders(initNetworks[0].RpcProviders)...)
	testutil.CompareProvidersList(s.T(), expectedProviders, foundNetwork.RpcProviders)
}

func (s *NetworkManagerTestSuite) TestInitNetworksDoesNotSaveEmbeddedProviders() {
	persistence := db.NewNetworksPersistence(s.db)
	s.Require().NoError(persistence.DeleteAllNetworks())

	// Re-initialize networks
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api_common.MainnetChainID, "Infura Mainnet", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://mainnet.infura.io")),
		}),
	}
	err := s.manager.InitEmbeddedNetworks(initNetworks)
	s.Require().NoError(err)

	// Check that embedded providers are not saved using persistence
	networks, err := persistence.GetNetworks(false, nil)
	s.Require().NoError(err)
	s.Require().Len(networks, 1)
	s.Require().Len(networks[0].RpcProviders, 0)
}

func (s *NetworkManagerTestSuite) TestInitEmbeddedNetworks() {
	// Re-initialize networks
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api_common.MainnetChainID, "Infura Mainnet", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://mainnet.infura.io")),
		}),
	}
	expectedProviders := networkhelper.GetEmbeddedProviders(initNetworks[0].RpcProviders)
	err := s.manager.InitEmbeddedNetworks(initNetworks)
	s.Require().NoError(err)

	// functor tests if embedded providers are present in the networks
	expectEmbeddedProviders := func(networks []*params.Network) {
		for _, network := range networks {
			if network.ChainID == api_common.MainnetChainID {
				storedEmbeddedProviders := networkhelper.GetEmbeddedProviders(network.RpcProviders)
				testutil.CompareProvidersList(s.T(), expectedProviders, storedEmbeddedProviders)
			}
		}
	}

	// GetAll
	networks, err := s.manager.GetAll()
	s.Require().NoError(err)
	expectEmbeddedProviders(networks)

	// Get
	networks, err = s.manager.Get(false)
	s.Require().NoError(err)
	expectEmbeddedProviders(networks)

	// GetActiveNetworks
	networks, err = s.manager.GetActiveNetworks()
	s.Require().NoError(err)
	expectEmbeddedProviders(networks)

	// GetCombinedNetworks
	combinedNetworks, err := s.manager.GetCombinedNetworks()
	s.Require().NoError(err)
	for _, combinedNetwork := range combinedNetworks {
		if combinedNetwork.Test != nil && combinedNetwork.Test.ChainID == api_common.MainnetChainID {
			storedEmbeddedProviders := networkhelper.GetEmbeddedProviders(combinedNetwork.Test.RpcProviders)
			testutil.CompareProvidersList(s.T(), expectedProviders, storedEmbeddedProviders)
		}
		if combinedNetwork.Prod != nil && combinedNetwork.Prod.ChainID == api_common.MainnetChainID {
			storedEmbeddedProviders := networkhelper.GetEmbeddedProviders(combinedNetwork.Prod.RpcProviders)
			testutil.CompareProvidersList(s.T(), expectedProviders, storedEmbeddedProviders)
		}
	}

	// GetEmbeddedNetworks
	embeddedNetworks := s.manager.GetEmbeddedNetworks()
	expectEmbeddedProviders(testutil.ConvertNetworksToPointers(embeddedNetworks))
}

func (s *NetworkManagerTestSuite) TestLegacyFieldPopulation() {
	// Create initial test networks with various providers
	initNetworks := []params.Network{
		*testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", []params.RpcProvider{
			testutil.CreateProvider(api_common.MainnetChainID, "DirectProvider1", params.EmbeddedDirectProviderType, true, security.NewSensitiveString("https://direct1.ethereum.io")),
			testutil.CreateProvider(api_common.MainnetChainID, "DirectProvider2", params.EmbeddedDirectProviderType, true, security.NewSensitiveString("https://direct2.ethereum.io")),
			testutil.CreateProvider(api_common.MainnetChainID, "ProxyProvider1", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://proxy1.ethereum.io")),
			testutil.CreateProvider(api_common.MainnetChainID, "ProxyProvider2", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://proxy2.ethereum.io")),
			testutil.CreateProvider(api_common.MainnetChainID, "ProxyProvider3", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://proxy3.ethereum.io")),
			testutil.CreateProvider(api_common.MainnetChainID, "UserProvider1", params.UserProviderType, true, security.NewSensitiveString("https://user1.ethereum.io")),
			testutil.CreateProvider(api_common.MainnetChainID, "UserProvider2", params.UserProviderType, true, security.NewSensitiveString("https://user2.ethereum.io")),
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
		*testutil.CreateNetwork(api_common.SepoliaChainID, "Sepolia Testnet", []params.RpcProvider{
			testutil.CreateProvider(api_common.SepoliaChainID, "DirectProvider1", params.EmbeddedDirectProviderType, true, security.NewSensitiveString("https://direct1.sepolia.io")),
			testutil.CreateProvider(api_common.SepoliaChainID, "ProxyProvider1", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://proxy1.sepolia.io")),
			testutil.CreateProvider(api_common.SepoliaChainID, "ProxyProvider2", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://proxy2.sepolia.io")),
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

func (s *NetworkManagerTestSuite) TestUpsertNetwork() {
	chainID := uint64(999)

	// Create a new network
	newNetwork := testutil.CreateNetwork(chainID, "Ethereum Mainnet", []params.RpcProvider{
		testutil.CreateProvider(chainID, "Infura Mainnet", params.EmbeddedProxyProviderType, true, security.NewSensitiveString("https://mainnet.infura.io")),
	})
	// Check that these values are overiden for upserted networks
	newNetwork.IsActive = true
	newNetwork.IsDeactivatable = false

	// Upsert the network
	err := s.manager.Upsert(newNetwork)
	s.Require().NoError(err)

	// Verify the network was upserted without embedded providers
	persistence := db.NewNetworksPersistence(s.db)
	networks, err := persistence.GetNetworks(false, &chainID)
	s.Require().NoError(err)
	s.Require().Len(networks, 1)
	s.Require().Len(networkhelper.GetEmbeddedProviders(networks[0].RpcProviders), 0)
	s.Require().False(networks[0].IsActive)
	s.Require().True(networks[0].IsDeactivatable)
}

func (s *NetworkManagerTestSuite) TestSetActive() {
	var err error
	var n *params.Network

	// Check that the  "Base" network is active by default
	n = s.manager.Find(api_common.BaseChainID)
	s.Require().NotNil(n)
	s.True(n.IsActive)

	// Set the "Base" network to inactive
	err = s.manager.SetActive(api_common.BaseChainID, false)
	s.Require().NoError(err)

	// Verify the network was set to inactive
	n = s.manager.Find(api_common.BaseChainID)
	s.Require().NotNil(n)
	s.False(n.IsActive)

	// Set the "Base" network to active
	err = s.manager.SetActive(api_common.BaseChainID, true)
	s.Require().NoError(err)

	// Verify the network was set to active
	n = s.manager.Find(api_common.BaseChainID)
	s.Require().NotNil(n)
	s.True(n.IsActive)
}

func (s *NetworkManagerTestSuite) TestSetActiveNotDeactivatable() {
	var err error
	var n *params.Network

	// Try to set Ethereum Mainnet to inactive (should fail)
	err = s.manager.SetActive(api_common.MainnetChainID, false)
	s.Require().Error(err)

	// Verify the network was not set to inactive
	n = s.manager.Find(api_common.MainnetChainID)
	s.Require().NotNil(n)
	s.True(n.IsActive)
}

func (s *NetworkManagerTestSuite) TestSetActiveMaxNumberOfActiveNetworks() {
	var err error
	var n *params.Network

	// Check that we're at the limit of active networks
	activeNetworks, err := s.manager.GetActiveNetworks()
	s.Require().NoError(err)
	s.Require().Len(activeNetworks, network.MaxActiveNetworks)

	// Check that the "Optimistic Ethereum" network is inactive by default
	n = s.manager.Find(api_common.OptimismChainID)
	s.Require().NotNil(n)
	s.False(n.IsActive)

	// Try to set the "Optimistic Ethereum" network to active (should fail due to number networks active)
	err = s.manager.SetActive(api_common.OptimismChainID, true)
	s.Require().Error(err)

	// Verify the network was not set to active
	n = s.manager.Find(api_common.OptimismChainID)
	s.Require().NotNil(n)
	s.False(n.IsActive)
}
