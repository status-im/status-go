package db_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc/network/db"
	"github.com/status-im/status-go/rpc/network/testutil"
	"github.com/status-im/status-go/t/helpers"
)

type RpcProviderPersistenceTestSuite struct {
	suite.Suite
	db             *sql.DB
	rpcPersistence *db.RpcProvidersPersistence
}

func (s *RpcProviderPersistenceTestSuite) SetupTest() {
	testDb := setupTestNetworkDB(s.T())
	s.db = testDb
	s.rpcPersistence = db.NewRpcProvidersPersistence(testDb)
}

func setupTestNetworkDB(t *testing.T) *sql.DB {
	testDb, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "rpc-providers-tests")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, cleanup()) })
	return testDb
}

func TestRpcProviderPersistenceTestSuite(t *testing.T) {
	suite.Run(t, new(RpcProviderPersistenceTestSuite))
}

// Test cases

func (s *RpcProviderPersistenceTestSuite) TestAddAndGetRpcProvider() {
	provider := testutil.CreateProvider(api.MainnetChainID, "Provider1", params.UserProviderType, true, "https://provider1.example.com")

	err := s.rpcPersistence.AddRpcProvider(provider)
	s.Require().NoError(err)

	// Verify the added provider
	providers, err := s.rpcPersistence.GetRpcProviders(api.MainnetChainID)
	s.Require().NoError(err)
	testutil.CompareProvidersList(s.T(), []params.RpcProvider{provider}, providers)
}

func (s *RpcProviderPersistenceTestSuite) TestGetRpcProvidersByType() {
	providers := []params.RpcProvider{
		testutil.CreateProvider(api.MainnetChainID, "UserProvider1", params.UserProviderType, true, "https://provider1.example.com"),
		testutil.CreateProvider(api.MainnetChainID, "EmbeddedDirect1", params.EmbeddedDirectProviderType, false, "https://provider2.example.com"),
		testutil.CreateProvider(api.MainnetChainID, "UserProvider2", params.UserProviderType, false, "https://provider3.example.com"),
		testutil.CreateProvider(api.MainnetChainID, "EmbeddedProxy1", params.EmbeddedProxyProviderType, true, "https://provider4.example.com"),
	}

	for _, provider := range providers {
		err := s.rpcPersistence.AddRpcProvider(provider)
		s.Require().NoError(err)
	}

	// Verify by type
	userProviders, err := s.rpcPersistence.GetRpcProvidersByType(api.MainnetChainID, params.UserProviderType)
	s.Require().NoError(err)
	testutil.CompareProvidersList(s.T(), []params.RpcProvider{providers[0], providers[2]}, userProviders)

	embeddedDirectProviders, err := s.rpcPersistence.GetRpcProvidersByType(api.MainnetChainID, params.EmbeddedDirectProviderType)
	s.Require().NoError(err)
	testutil.CompareProvidersList(s.T(), []params.RpcProvider{providers[1]}, embeddedDirectProviders)

	embeddedProxyProviders, err := s.rpcPersistence.GetRpcProvidersByType(api.MainnetChainID, params.EmbeddedProxyProviderType)
	s.Require().NoError(err)
	testutil.CompareProvidersList(s.T(), []params.RpcProvider{providers[3]}, embeddedProxyProviders)
}

func (s *RpcProviderPersistenceTestSuite) TestDeleteRpcProviders() {
	provider := testutil.CreateProvider(api.MainnetChainID, "Provider1", params.UserProviderType, true, "https://provider1.example.com")

	err := s.rpcPersistence.AddRpcProvider(provider)
	s.Require().NoError(err)

	err = s.rpcPersistence.DeleteRpcProviders(api.MainnetChainID)
	s.Require().NoError(err)

	// Verify deletion
	providers, err := s.rpcPersistence.GetRpcProviders(api.MainnetChainID)
	s.Require().NoError(err)
	s.Require().Empty(providers)
}

func (s *RpcProviderPersistenceTestSuite) TestUpdateRpcProvider() {
	provider := testutil.CreateProvider(api.MainnetChainID, "Provider1", params.UserProviderType, true, "https://provider1.example.com")

	err := s.rpcPersistence.AddRpcProvider(provider)
	s.Require().NoError(err)

	// Retrieve provider to get the ID
	providers, err := s.rpcPersistence.GetRpcProviders(api.MainnetChainID)
	s.Require().NoError(err)
	s.Require().Len(providers, 1)

	provider.ID = providers[0].ID
	provider.URL = "https://provider1-updated.example.com"
	provider.EnableRPSLimiter = false

	err = s.rpcPersistence.UpdateRpcProvider(provider)
	s.Require().NoError(err)

	// Verify update
	updatedProviders, err := s.rpcPersistence.GetRpcProviders(api.MainnetChainID)
	s.Require().NoError(err)
	testutil.CompareProvidersList(s.T(), []params.RpcProvider{provider}, updatedProviders)
}

func (s *RpcProviderPersistenceTestSuite) TestSetRpcProviders() {
	initialProviders := []params.RpcProvider{
		testutil.CreateProvider(api.MainnetChainID, "Provider1", params.UserProviderType, true, "https://provider1.example.com"),
		testutil.CreateProvider(api.MainnetChainID, "Provider2", params.EmbeddedDirectProviderType, false, "https://provider2.example.com"),
	}

	for _, provider := range initialProviders {
		err := s.rpcPersistence.AddRpcProvider(provider)
		s.Require().NoError(err)
	}

	newProviders := []params.RpcProvider{
		testutil.CreateProvider(api.MainnetChainID, "NewProvider1", params.UserProviderType, true, "https://newprovider1.example.com"),
		testutil.CreateProvider(api.MainnetChainID, "NewProvider2", params.EmbeddedProxyProviderType, true, "https://newprovider2.example.com"),
	}

	err := s.rpcPersistence.SetRpcProviders(api.MainnetChainID, newProviders)
	s.Require().NoError(err)

	// Verify replacement
	providers, err := s.rpcPersistence.GetRpcProviders(api.MainnetChainID)
	s.Require().NoError(err)
	testutil.CompareProvidersList(s.T(), newProviders, providers)
}

func (s *RpcProviderPersistenceTestSuite) TestAddRpcProviderValidation() {
	invalidProvider := params.RpcProvider{
		ChainID: 0,              // Invalid: must be greater than 0
		Name:    "",             // Invalid: cannot be empty
		URL:     "invalid-url",  // Invalid: not a valid URL
		Type:    "invalid-type", // Invalid: not in allowed values
	}

	err := s.rpcPersistence.AddRpcProvider(invalidProvider)
	s.Require().Error(err)
	s.Contains(err.Error(), "validation failed")
}
