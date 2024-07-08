package commands

import (
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/transactions"
)

// errors
var (
	ErrRequestMissingDAppData   = errors.New("request missing dApp data")
	ErrDAppIsNotPermittedByUser = errors.New("dApp is not permitted by user")
	ErrEmptyRPCParams           = errors.New("empty rpc params")
	ErrNoChainIDInParams        = errors.New("no chain id in params")
)

type RPCRequest struct {
	JSONRPC     string        `json:"jsonrpc"`
	ID          int           `json:"id"`
	Method      string        `json:"method"`
	Params      []interface{} `json:"params"`
	Origin      string        `json:"origin"`
	DAppName    string        `json:"dAppName"`
	DAppIconUrl string        `json:"dAppIconUrl"`
}

type RPCCommand interface {
	Execute(request RPCRequest) (string, error)
}

type DAppData struct {
	Origin  string
	Name    string
	IconUrl string
}

type ConnectorSendTransactionFinishedArgs struct {
	Hash  types.Hash
	Error *error
}

type ClientSideHandlerInterface interface {
	RequestShareAccountForDApp(dApp *DAppData) (types.Address, error)
	RequestSendTransaction(dApp *DAppData, chainID uint64, txArgs *transactions.SendTxArgs) (types.Hash, error)
	ConnectorSendTransactionFinished(args ConnectorSendTransactionFinishedArgs) error
}

type NetworkManagerInterface interface {
	GetActiveNetworks() ([]*params.Network, error)
}

type RPCClientInterface interface {
	CallRaw(body string) string
}

func (r *RPCRequest) getDAppData() (*DAppData, error) {
	if r.Origin == "" || r.DAppName == "" {
		return nil, ErrRequestMissingDAppData
	}

	return &DAppData{
		Origin:  r.Origin,
		Name:    r.DAppName,
		IconUrl: r.DAppIconUrl,
	}, nil
}
