package commands

import (
	"database/sql"
	"encoding/json"
	"errors"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/transactions"
)

var (
	ErrNoSendTransactionParams      = errors.New("no send transaction params")
	ErrParamsFromAddressIsNotShared = errors.New("from parameter address is not dApp's shared account")
)

type SendTransactionCommand struct {
	Db            *sql.DB
	ClientHandler ClientSideHandlerInterface
}

func (r *RPCRequest) getSendTransactionParams() (*transactions.SendTxArgs, error) {
	if r.Params == nil || len(r.Params) == 0 {
		return nil, ErrEmptyRPCParams
	}

	paramsRaw, ok := r.Params[0].(string)
	if !ok {
		return nil, ErrNoSendTransactionParams
	}

	params := &transactions.SendTxArgs{}
	err := json.Unmarshal([]byte(paramsRaw), params)
	if err != nil {
		return nil, err
	}

	return params, nil
}

func (c *SendTransactionCommand) Execute(request RPCRequest) (string, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}
	dAppData := request.GetDAppData()

	dApp, err := persistence.SelectDAppByUrl(c.Db, dAppData.Origin)
	if err != nil {
		return "", err
	}

	if dApp == nil {
		return "", ErrDAppIsNotPermittedByUser
	}

	params, err := request.getSendTransactionParams()
	if err != nil {
		return "", err
	}

	if params.From != dApp.SharedAccount {
		return "", ErrParamsFromAddressIsNotShared
	}

	hash, err := c.ClientHandler.RequestSendTransaction(dAppData, dApp.ChainID, params)
	if err != nil {
		return "", err
	}
	return hash.String(), nil
}
