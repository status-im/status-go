package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/transactions"
)

// errors
var (
	ErrRequestMissingDAppData   = errors.New("request missing dApp data")
	ErrDAppIsNotPermittedByUser = errors.New("dApp is not permitted by user")
	ErrEmptyRPCParams           = errors.New("empty rpc params")
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
	Origin  string `json:"origin"`
	Name    string `json:"name"`
	IconUrl string `json:"iconUrl"`
}

type RequestAccountsFinishedArgs struct {
	Account types.Address `json:"account"`
	ChainID uint64        `json:"chainId"`
	Error   *error        `json:"error"`
}

type SendTransactionFinishedArgs struct {
	Hash  types.Hash `json:"hash"`
	Error *error     `json:"error"`
}

type ClientSideHandlerInterface interface {
	RequestShareAccountForDApp(dApp DAppData, chainIDs []uint64) (types.Address, uint64, error)
	RequestSendTransaction(dApp DAppData, chainID uint64, txArgs *transactions.SendTxArgs) (types.Hash, error)

	RequestAccountsFinished(args RequestAccountsFinishedArgs) error
	SendTransactionFinished(args SendTransactionFinishedArgs) error
}

type NetworkManagerInterface interface {
	GetActiveNetworks() ([]*params.Network, error)
}

type RPCClientInterface interface {
	CallRaw(body string) string
}

func RPCRequestFromJSON(inputJSON string) (RPCRequest, error) {
	var request RPCRequest

	err := json.Unmarshal([]byte(inputJSON), &request)
	if err != nil {
		return RPCRequest{}, fmt.Errorf("error unmarshalling JSON: %v", err)
	}
	return request, nil
}

func (r *RPCRequest) Validate() error {
	if r.Origin == "" || r.DAppName == "" {
		return ErrRequestMissingDAppData
	}
	return nil
}

func (r *RPCRequest) GetDAppData() DAppData {
	return DAppData{
		Origin:  r.Origin,
		Name:    r.DAppName,
		IconUrl: r.DAppIconUrl,
	}
}
