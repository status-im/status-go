package commands

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

var (
	ErrParamsFromAddressIsNotShared = errors.New("from parameter address is not dApp's shared account")
	ErrNoTransactionParamsFound     = errors.New("no transaction in params found")
)

type SendTransactionCommand struct {
	Db            *sql.DB
	ClientHandler ClientSideHandlerInterface
}

func (r *RPCRequest) getSendTransactionParams() (*transactions.SendTxArgs, error) {
	if r.Params == nil || len(r.Params) == 0 {
		return nil, ErrEmptyRPCParams
	}

	paramMap, ok := r.Params[0].(map[string]interface{})
	if !ok {
		return nil, ErrNoTransactionParamsFound
	}

	paramBytes, err := json.Marshal(paramMap)
	if err != nil {
		return nil, fmt.Errorf("error marshalling first transaction param: %v", err)
	}

	var sendTxArgs transactions.SendTxArgs
	err = json.Unmarshal(paramBytes, &sendTxArgs)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling first transaction param to SendTxArgs: %v", err)
	}

	return &sendTxArgs, nil
}

func (c *SendTransactionCommand) Execute(request RPCRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return "", err
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, request.URL)
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

	hash, err := c.ClientHandler.RequestSendTransaction(signal.ConnectorDApp{
		URL:     request.URL,
		Name:    request.Name,
		IconURL: request.IconURL,
	}, dApp.ChainID, params)
	if err != nil {
		return "", err
	}
	return hash.String(), nil
}
