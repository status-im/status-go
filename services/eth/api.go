package eth

import (
	"context"
	"fmt"

	accounts "github.com/status-im/status-go/accounts-management"

	"github.com/status-im/status-go/rpc"
)

type (
	// EstimateGasArgs are the Gorilla RPC request args for eth_estimateGas.
	EstimateGasArgs struct {
		ChainID uint64      `json:"chainID"`
		Payload interface{} `json:"payload"`
	}
	// EstimateGasReply is the Gorilla RPC reply type for eth_estimateGas.
	EstimateGasReply string
)

type API struct {
	client          *rpc.Client
	accountsManager *accounts.AccountsManager
}

func NewAPI(client *rpc.Client, accountsManager *accounts.AccountsManager) *API {
	return &API{
		client:          client,
		accountsManager: accountsManager,
	}
}

// EstimateGas estimates gas for a given payload on a chain.
func (a *API) EstimateGas(ctx context.Context, chainID uint64, payload any) (result string, err error) {
	client, err := a.client.EthClient(chainID)
	if err != nil {
		return "", err
	}
	err = client.CallContext(ctx, &result, "eth_estimateGas", payload)
	return
}

// EstimateGasGorilla is a gorilla/rpc-compatible wrapper around EstimateGas.
// It matches RegisterTCPService expected signature: (args *Args, reply *Reply) error
// JSON-RPC method name: "eth_estimateGas"
func (a *API) EstimateGasGorilla(args *EstimateGasArgs, reply *EstimateGasReply) error {
	if args == nil || reply == nil {
		return nil
	}
	res, err := a.EstimateGas(context.Background(), args.ChainID, args.Payload)
	if err != nil {
		return err
	}
	*reply = EstimateGasReply(res)
	return nil
}

type GorillaArgs struct {
	A string `json:"a"`
}

type GorillaReply struct {
	B string `json:"b"`
}

func (a *API) TestGorilla(args *GorillaArgs, reply *GorillaReply) error {
	reply.B = fmt.Sprintf("a: %s", args.A)
	return nil
}

func (a *API) TestGorilla2(args *GorillaArgs, reply *GorillaReply) error {
	reply.B = fmt.Sprintf("a: %s", args.A)
	return fmt.Errorf("test error: %s", args.A)
}
