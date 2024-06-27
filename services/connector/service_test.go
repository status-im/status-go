package connector

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	statusRPC "github.com/status-im/status-go/rpc"
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
	rpcClient, err := statusRPC.NewClient(client, 1, upstreamConfig, nil, db)
	require.NoError(t, err)

	mockConnectorService := &Service{}
	service := NewService(db, rpcClient, mockConnectorService)

	assert.NotNil(t, service)
	assert.Equal(t, rpcClient, service.rpcClient)
	assert.Equal(t, mockConnectorService, service.connectorSrvc)
}

func TestService_Start(t *testing.T) {
	db, close := createDB(t)
	defer close()

	service := NewService(db, &rpc.Client{}, &Service{})
	err := service.Start()
	assert.NoError(t, err)
}

func TestService_Stop(t *testing.T) {
	db, close := createDB(t)
	defer close()

	service := NewService(db, &rpc.Client{}, &Service{})
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

	service := NewService(db, &rpc.Client{}, &Service{})
	protocols := service.Protocols()
	assert.Nil(t, protocols)
}
