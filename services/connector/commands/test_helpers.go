package commands

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

var testDAppData = signal.ConnectorDApp{
	URL:     "http://testDAppURL",
	Name:    "testDAppName",
	IconURL: "http://testDAppIconUrl",
}

type RPCClientMock struct {
	response string
}

type NetworkManagerMock struct {
	networks []*params.Network
}

type EventType struct {
	Type  string          `json:"type"`
	Event json.RawMessage `json:"event"`
}

func (c *RPCClientMock) CallRaw(request string) string {
	return c.response
}

func (c *RPCClientMock) SetResponse(response string) {
	c.response = response
}

func (nm *NetworkManagerMock) GetActiveNetworks() ([]*params.Network, error) {
	return nm.networks, nil
}

func (nm *NetworkManagerMock) SetNetworks(networks []*params.Network) {
	nm.networks = networks
}

func SetupTestDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

func PersistDAppData(db *sql.DB, dApp signal.ConnectorDApp, sharedAccount types.Address, chainID uint64) error {
	dAppDb := persistence.DApp{
		URL:           dApp.URL,
		Name:          dApp.Name,
		IconURL:       dApp.IconURL,
		SharedAccount: sharedAccount,
		ChainID:       chainID,
	}

	return persistence.UpsertDApp(db, &dAppDb)
}

func ConstructRPCRequest(method string, params []interface{}, dApp *signal.ConnectorDApp) (RPCRequest, error) {
	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	if dApp != nil {
		request.URL = dApp.URL
		request.Name = dApp.Name
		request.IconURL = dApp.IconURL
	}

	return request, nil
}
