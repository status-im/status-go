package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

// errors
var (
	ErrRequestMissingDAppData   = errors.New("request missing dApp data")
	ErrDAppIsNotPermittedByUser = errors.New("dApp is not permitted by user")
	ErrEmptyRPCParams           = errors.New("empty rpc params")
)

type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	URL     string        `json:"url"`
	Name    string        `json:"name"`
	IconURL string        `json:"iconUrl"`
}

type RPCCommand interface {
	Execute(request RPCRequest) (string, error)
}

type RequestAccountsAcceptedArgs struct {
	RequestID string        `json:"requestId"`
	Account   types.Address `json:"account"`
	ChainID   uint64        `json:"chainId"`
}

type SendTransactionAcceptedArgs struct {
	RequestID string     `json:"requestId"`
	Hash      types.Hash `json:"hash"`
}

type RejectedArgs struct {
	RequestID string `json:"requestId"`
}

type ClientSideHandlerInterface interface {
	RequestShareAccountForDApp(dApp signal.ConnectorDApp) (types.Address, uint64, error)
	RequestSendTransaction(dApp signal.ConnectorDApp, chainID uint64, txArgs *transactions.SendTxArgs) (types.Hash, error)

	RequestAccountsAccepted(args RequestAccountsAcceptedArgs) error
	RequestAccountsRejected(args RejectedArgs) error
	SendTransactionAccepted(args SendTransactionAcceptedArgs) error
	SendTransactionRejected(args RejectedArgs) error
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
	if r.URL == "" || r.Name == "" {
		return ErrRequestMissingDAppData
	}
	return nil
}
