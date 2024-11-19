package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

const (
	Method_EthAccounts         = "eth_accounts"
	Method_EthRequestAccounts  = "eth_requestAccounts"
	Method_EthChainId          = "eth_chainId"
	Method_PersonalSign        = "personal_sign"
	Method_SignTypedDataV4     = "eth_signTypedData_v4"
	Method_EthSendTransaction  = "eth_sendTransaction"
	Method_RequestPermissions  = "wallet_requestPermissions"
	Method_RevokePermissions   = "wallet_revokePermissions"
	Method_SwitchEthereumChain = "wallet_switchEthereumChain"
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
	ChainID uint64        `json:"chainId"`
}

type RPCCommand interface {
	Execute(ctx context.Context, request RPCRequest) (interface{}, error)
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

type SignAcceptedArgs struct {
	RequestID string `json:"requestId"`
	Signature string `json:"signature"`
}

type RejectedArgs struct {
	RequestID string `json:"requestId"`
}

type RecallDAppPermissionsArgs struct {
	URL string `json:"url"`
}

type ClientSideHandlerInterface interface {
	RequestShareAccountForDApp(dApp signal.ConnectorDApp) (types.Address, uint64, error)
	RequestAccountsAccepted(args RequestAccountsAcceptedArgs) error
	RequestAccountsRejected(args RejectedArgs) error
	RecallDAppPermissions(args RecallDAppPermissionsArgs) error

	RequestSendTransaction(dApp signal.ConnectorDApp, chainID uint64, txArgs *transactions.SendTxArgs) (types.Hash, error)
	SendTransactionAccepted(args SendTransactionAcceptedArgs) error
	SendTransactionRejected(args RejectedArgs) error

	RequestSign(dApp signal.ConnectorDApp, challenge, address string, method string) (string, error)
	SignAccepted(args SignAcceptedArgs) error
	SignRejected(args RejectedArgs) error
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
