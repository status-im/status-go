package commands

import "github.com/status-im/status-go/params"

type RPCClientMock struct {
	response string
}

type NetworkManagerMock struct {
	networks []*params.Network
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
