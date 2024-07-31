package connector

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
	statusRPC "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/connector/commands"
	"github.com/status-im/status-go/transactions/fake"
)

func TestNewService(t *testing.T) {
	db, close := createDB(t)
	defer close()

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

	service := NewService(db, rpcClient, rpcClient.NetworkManager)

	assert.NotNil(t, service)
	assert.Equal(t, rpcClient.NetworkManager, service.nm)
}

func TestService_Start(t *testing.T) {
	db, close := createDB(t)
	defer close()

	service := NewService(db, &commands.RPCClientMock{}, &commands.NetworkManagerMock{})
	err := service.Start()
	assert.NoError(t, err)
}

func TestService_Stop(t *testing.T) {
	db, close := createDB(t)
	defer close()

	service := NewService(db, &commands.RPCClientMock{}, &commands.NetworkManagerMock{})
	err := service.Stop()
	assert.NoError(t, err)
}

func TestService_APIs(t *testing.T) {
	api, cancel := setupTestAPI(t)
	defer cancel()

	apis := api.s.APIs()

	assert.Len(t, apis, 1)
	assert.Equal(t, "connector", apis[0].Namespace)
	assert.Equal(t, "0.1.0", apis[0].Version)
	assert.NotNil(t, apis[0].Service)
}

func TestService_Protocols(t *testing.T) {
	db, close := createDB(t)
	defer close()

	service := NewService(db, &commands.RPCClientMock{}, &commands.NetworkManagerMock{})
	protocols := service.Protocols()
	assert.Nil(t, protocols)
}
