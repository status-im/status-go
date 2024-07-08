package commands

import (
	"database/sql"
	"encoding/json"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	persistence "github.com/status-im/status-go/services/connector/database"
)

var testDAppData = DAppData{
	Origin:  "http://testDAppURL",
	Name:    "testDAppName",
	IconUrl: "http://testDAppIconUrl",
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

func persistDAppData(db *sql.DB, dAppData DAppData, sharedAccount types.Address, chainID uint64) error {
	dApp := persistence.DApp{
		URL:           dAppData.Origin,
		Name:          dAppData.Name,
		IconURL:       dAppData.IconUrl,
		SharedAccount: sharedAccount,
		ChainID:       chainID,
	}

	return persistence.UpsertDApp(db, &dApp)
}

func constructRPCRequest(method string, params []interface{}, dAppData *DAppData) RPCRequest {
	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	if dAppData != nil {
		request.Origin = dAppData.Origin
		request.DAppName = dAppData.Name
		request.DAppIconUrl = dAppData.IconUrl
	}

	return request
}
