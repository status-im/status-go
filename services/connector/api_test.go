package connector

import (
	"database/sql"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/params"
	statusRPC "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/connector/commands"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions/fake"
	"github.com/status-im/status-go/walletdatabase"
)

func createDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "provider-tests-")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func setupTestAPI(t *testing.T) (*API, func()) {
	db, cancel := createDB(t)

	txServiceMockCtrl := gomock.NewController(t)
	server, _ := fake.NewTestServer(txServiceMockCtrl)

	// Creating a dummy status node to simulate what it's done in get_status_node.go
	upstreamConfig := params.UpstreamRPCConfig{
		URL:     "https://mainnet.infura.io/v3/fake",
		Enabled: true,
	}

	client := gethrpc.DialInProc(server)
	rpcClient, err := statusRPC.NewClient(client, 1, upstreamConfig, nil, db, nil)
	require.NoError(t, err)

	service := NewService(db, rpcClient, nil)

	return NewAPI(service), cancel
}

func TestCallRPC(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	tests := []struct {
		request     string
		expectError error
	}{
		{
			request:     "{\"method\": \"eth_chainId\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
		{
			request:     "{\"method\": \"eth_accounts\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
		{
			request:     "{\"method\": \"eth_requestAccounts\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
		{
			request:     "{\"method\": \"eth_sendTransaction\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
		{
			request:     "{\"method\": \"wallet_switchEthereumChain\", \"params\": []}",
			expectError: commands.ErrRequestMissingDAppData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.request, func(t *testing.T) {
			_, err := api.CallRPC(tt.request)
			require.Error(t, err)
			require.Equal(t, tt.expectError, err)
		})
	}
}
