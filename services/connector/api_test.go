package connector

import (
	"database/sql"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	statusRPC "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions/fake"
)

func createDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(appdatabase.DbInitializer{}, "provider-tests-")
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
	rpcClient, err := statusRPC.NewClient(client, 1, upstreamConfig, nil, db)
	require.NoError(t, err)

	service := NewService(rpcClient, nil)

	return NewAPI(service), cancel
}

func TestCallRPC(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	tests := []struct {
		request          string
		expectError      bool
		expectedContains string
		notContains      bool
	}{
		{
			request:          "{\"method\": \"eth_blockNumber\", \"params\": []}",
			expectError:      false,
			expectedContains: "does not exist/is not available",
			notContains:      true,
		},
		{
			request:          "{\"method\": \"eth_blockNumbers\", \"params\": []}",
			expectError:      false,
			expectedContains: "does not exist/is not available",
			notContains:      false,
		},
		{
			request:          "",
			expectError:      true,
			expectedContains: "does not exist/is not available",
			notContains:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.request, func(t *testing.T) {
			response, err := api.CallRPC(tt.request)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, response)
				if tt.notContains {
					require.NotContains(t, response, tt.expectedContains)
				} else {
					require.Contains(t, response, tt.expectedContains)
				}
			}
		})
	}
}
