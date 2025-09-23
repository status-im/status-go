package eth

import (
	"context"
	"fmt"

	accounts "github.com/status-im/status-go/accounts-management"

	"github.com/status-im/status-go/rpc"
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

func (a *API) TestGorilla2(args *GorillaArgs) (*GorillaReply, error) {
	reply := &GorillaReply{
		B: fmt.Sprintf("a: %s", args.A),
	}
	return reply, nil
}

func (a *API) TestPanic(args *GorillaArgs, reply *GorillaReply) error {
	panic("test panic")
}