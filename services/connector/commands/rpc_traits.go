package commands

import (
	"errors"

	"github.com/status-im/status-go/params"
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

type NetworkManagerInterface interface {
	GetActiveNetworks() ([]*params.Network, error)
}

type RPCClientInterface interface {
	CallRaw(body string) string
}

func (r *RPCRequest) checkDAppData() error {
	if r.Origin == "" || r.DAppName == "" {
		return ErrRequestMissingDAppData
	}

	return nil
}

func (r *RPCRequest) getChainID() (uint64, error) {
	if r.Params == nil || len(r.Params) == 0 {
		return 0, ErrEmptyRPCParams
	}

	// First, try to assert it as float64 (which is the default for numbers in JSON)
	chainIDFloat, ok := r.Params[0].(float64)
	if !ok {
		return 0, ErrNoChainIDInParams
	}
	return uint64(chainIDFloat), nil
}
