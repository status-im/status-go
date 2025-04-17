package db_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	api_common "github.com/status-im/status-go/api/common"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/internal/security"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc/network/db"
	"github.com/status-im/status-go/rpc/network/testutil"
	"github.com/status-im/status-go/t/helpers"
)

type NetworksPersistenceTestSuite struct {
	suite.Suite
	db                  *sql.DB
	cleanup             func() error
	networksPersistence db.NetworksPersistenceInterface
}

func (s *NetworksPersistenceTestSuite) SetupTest() {
	memDb, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "networks-tests")
	s.Require().NoError(err)
	s.db = memDb
	s.cleanup = cleanup
	s.networksPersistence = db.NewNetworksPersistence(memDb)
}

func (s *NetworksPersistenceTestSuite) TearDownTest() {
	if s.cleanup != nil {
		err := s.cleanup()
		require.NoError(s.T(), err)
	}
}

func TestNetworksPersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(NetworksPersistenceTestSuite))
}

// Helper function to create default providers for a given chainID
func DefaultProviders(chainID uint64) []params.RpcProvider {
	return []params.RpcProvider{
		{
			Name:     "Provider1",
			ChainID:  chainID,
			URL:      security.NewSensitiveString("https://rpc.provider1.io"),
			Type:     params.UserProviderType,
			Enabled:  true,
			AuthType: params.NoAuth,
		},
		{
			Name:         "Provider2",
			ChainID:      chainID,
			URL:          security.NewSensitiveString("https://rpc.provider2.io"),
			Type:         params.EmbeddedProxyProviderType,
			Enabled:      true,
			AuthType:     params.BasicAuth,
			AuthLogin:    security.NewSensitiveString("user1"),
			AuthPassword: security.NewSensitiveString("password1"),
		},
	}
}

// Helper function to add and verify networks
func (s *NetworksPersistenceTestSuite) addAndVerifyNetworks(networks []*params.Network) {
	networkValues := make([]params.Network, 0, len(networks))
	for _, network := range networks {
		networkValues = append(networkValues, *network)
	}
	err := s.networksPersistence.SetNetworks(networkValues)
	s.Require().NoError(err)

	s.verifyNetworks(networks)
}

// Helper function to verify networks against the database
func (s *NetworksPersistenceTestSuite) verifyNetworks(networks []*params.Network) {
	allNetworks, err := s.networksPersistence.GetAllNetworks()
	s.Require().NoError(err)
	testutil.CompareNetworksList(s.T(), networks, allNetworks)
}

// Helper function to verify network deletion
func (s *NetworksPersistenceTestSuite) verifyNetworkDeletion(chainID uint64) {
	nets, err := s.networksPersistence.GetNetworkByChainID(chainID)
	s.Require().NoError(err)
	s.Require().Len(nets, 0)

	providers, err := s.networksPersistence.GetRpcPersistence().GetRpcProviders(chainID)
	s.Require().NoError(err)
	s.Require().Len(providers, 0)
}

// Tests

func (s *NetworksPersistenceTestSuite) TestAddAndGetNetworkWithProviders() {
	network := testutil.CreateNetwork(api_common.OptimismChainID, "Optimism Mainnet", []params.RpcProvider{
		testutil.CreateProvider(api_common.OptimismChainID, "Provider1", params.UserProviderType, true, security.NewSensitiveString("https://rpc.optimism.io")),
		testutil.CreateProvider(api_common.OptimismChainID, "Provider2", params.EmbeddedProxyProviderType, false, security.NewSensitiveString("https://backup.optimism.io")),
	})
	s.addAndVerifyNetworks([]*params.Network{network})
}

func (s *NetworksPersistenceTestSuite) TestDeleteNetworkWithProviders() {
	network := testutil.CreateNetwork(api_common.OptimismChainID, "Optimism Mainnet", DefaultProviders(api_common.OptimismChainID))
	s.addAndVerifyNetworks([]*params.Network{network})

	err := s.networksPersistence.DeleteNetwork(network.ChainID)
	s.Require().NoError(err)

	s.verifyNetworkDeletion(network.ChainID)
}

func (s *NetworksPersistenceTestSuite) TestUpdateNetworkAndProviders() {
	network := testutil.CreateNetwork(api_common.OptimismChainID, "Optimism Mainnet", DefaultProviders(api_common.OptimismChainID))
	s.addAndVerifyNetworks([]*params.Network{network})

	// Update fields
	network.ChainName = "Updated Optimism Mainnet"
	network.RpcProviders = []params.RpcProvider{
		testutil.CreateProvider(api_common.OptimismChainID, "UpdatedProvider", params.UserProviderType, true, security.NewSensitiveString("https://rpc.optimism.updated.io")),
	}

	s.addAndVerifyNetworks([]*params.Network{network})
}

func (s *NetworksPersistenceTestSuite) TestDeleteAllNetworks() {
	networks := []*params.Network{
		testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", DefaultProviders(api_common.MainnetChainID)),
		testutil.CreateNetwork(api_common.SepoliaChainID, "Sepolia Testnet", DefaultProviders(api_common.SepoliaChainID)),
	}
	s.addAndVerifyNetworks(networks)

	err := s.networksPersistence.DeleteAllNetworks()
	s.Require().NoError(err)

	allNetworks, err := s.networksPersistence.GetAllNetworks()
	s.Require().NoError(err)
	s.Require().Len(allNetworks, 0)
}

func (s *NetworksPersistenceTestSuite) TestSetNetworks() {
	initialNetworks := []*params.Network{
		testutil.CreateNetwork(api_common.MainnetChainID, "Ethereum Mainnet", DefaultProviders(api_common.MainnetChainID)),
		testutil.CreateNetwork(api_common.SepoliaChainID, "Sepolia Testnet", DefaultProviders(api_common.SepoliaChainID)),
	}
	newNetworks := []*params.Network{
		testutil.CreateNetwork(api_common.OptimismChainID, "Optimism Mainnet", DefaultProviders(api_common.OptimismChainID)),
	}

	// Add initial networks
	s.addAndVerifyNetworks(initialNetworks)

	// Replace with new networks
	s.addAndVerifyNetworks(newNetworks)

	// Verify old networks are removed
	s.verifyNetworkDeletion(api_common.MainnetChainID)
	s.verifyNetworkDeletion(api_common.SepoliaChainID)
}

func (s *NetworksPersistenceTestSuite) TestValidationForNetworksAndProviders() {
	// Invalid Network: Missing required ChainName
	invalidNetwork := testutil.CreateNetwork(api_common.MainnetChainID, "", DefaultProviders(api_common.MainnetChainID))

	// Invalid Provider: Missing URL
	invalidProvider := params.RpcProvider{
		Name:    "InvalidProvider",
		ChainID: api_common.MainnetChainID,
		URL:     security.NewSensitiveString(""), // Invalid
		Type:    params.UserProviderType,
		Enabled: true,
	}

	// Add invalid provider to a valid network
	validNetworkWithInvalidProvider := testutil.CreateNetwork(api_common.OptimismChainID, "Optimism Mainnet", []params.RpcProvider{invalidProvider})

	// Invalid networks and providers should fail validation
	networksToValidate := []*params.Network{
		invalidNetwork,
		validNetworkWithInvalidProvider,
	}

	for _, network := range networksToValidate {
		err := s.networksPersistence.UpsertNetwork(network)
		s.Require().Error(err, "Expected validation to fail for invalid network or provider")
	}

	// Ensure no invalid data is saved in the database
	allNetworks, err := s.networksPersistence.GetAllNetworks()
	s.Require().NoError(err)
	s.Require().Len(allNetworks, 0, "No invalid networks should be saved")
}

func (s *NetworksPersistenceTestSuite) TestSetActive() {
	network := testutil.CreateNetwork(api_common.OptimismChainID, "Optimism Mainnet", DefaultProviders(api_common.OptimismChainID))
	s.addAndVerifyNetworks([]*params.Network{network})

	// Deactivate the network
	err := s.networksPersistence.SetActive(network.ChainID, false)
	s.Require().NoError(err)

	// Verify the network is deactivated
	updatedNetwork, err := s.networksPersistence.GetNetworkByChainID(network.ChainID)
	s.Require().NoError(err)
	s.Require().Len(updatedNetwork, 1)
	s.Require().False(updatedNetwork[0].IsActive)

	// Activate the network
	err = s.networksPersistence.SetActive(network.ChainID, true)
	s.Require().NoError(err)

	// Verify the network is enabled
	updatedNetwork, err = s.networksPersistence.GetNetworkByChainID(network.ChainID)
	s.Require().NoError(err)
	s.Require().Len(updatedNetwork, 1)
	s.Require().True(updatedNetwork[0].IsActive)
}

func (s *NetworksPersistenceTestSuite) TestSetEnabled() {
	network := testutil.CreateNetwork(api_common.OptimismChainID, "Optimism Mainnet", DefaultProviders(api_common.OptimismChainID))
	s.addAndVerifyNetworks([]*params.Network{network})

	// Disable the network
	err := s.networksPersistence.SetEnabled(network.ChainID, false)
	s.Require().NoError(err)

	// Verify the network is disabled
	updatedNetwork, err := s.networksPersistence.GetNetworkByChainID(network.ChainID)
	s.Require().NoError(err)
	s.Require().Len(updatedNetwork, 1)
	s.Require().False(updatedNetwork[0].Enabled)

	// Enable the network
	err = s.networksPersistence.SetEnabled(network.ChainID, true)
	s.Require().NoError(err)

	// Verify the network is enabled
	updatedNetwork, err = s.networksPersistence.GetNetworkByChainID(network.ChainID)
	s.Require().NoError(err)
	s.Require().Len(updatedNetwork, 1)
	s.Require().True(updatedNetwork[0].Enabled)
}
