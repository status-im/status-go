package main

import (
	"github.com/ethereum/go-ethereum/rpc"
)

type client struct {
	rpcClient *rpc.Client
}

func newClient(ipcPath string) (*client, error) {
	rpcClient, err := rpc.Dial(ipcPath)
	if err != nil {
		return nil, err
	}

	return &client{rpcClient}, nil
}

func (c *client) metrics() (metrics, error) {
	var res metrics
	return res, c.rpcClient.Call(&res, "debug_metrics", true)
}
