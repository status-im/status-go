package commands

import (
	"database/sql"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
	statusRPC "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/transactions/fake"
	"github.com/status-im/status-go/walletdatabase"
)

func createDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "provider-tests-")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func TestChainIDCommandExecution(t *testing.T) {
	t.Skip("skip test using infura")
	db, _ := createDB(t)

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

	cmd := &ChainIDCommand{RpcClient: rpcClient}

	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "eth_chainId",
		Params:  []interface{}{},
	}
	expectedOutput := "0x1"

	result, err := cmd.Execute(request)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, result)
}
