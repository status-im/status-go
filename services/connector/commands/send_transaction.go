package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/rpc"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/signal"
)

var (
	ErrParamsFromAddressIsNotShared = errors.New("from parameter address is not dApp's shared account")
	ErrNoTransactionParamsFound     = errors.New("no transaction in params found")
	ErrSendingParamsInvalid         = errors.New("sending params are invalid")
)

type SendTransactionCommand struct {
	RpcClient     rpc.ClientInterface
	Db            *sql.DB
	ClientHandler ClientSideHandlerInterface
}

func (r *RPCRequest) getSendTransactionParams() (*wallettypes.SendTxArgs, error) {
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

	var sendTxArgs wallettypes.SendTxArgs
	err = json.Unmarshal(paramBytes, &sendTxArgs)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling first transaction param to SendTxArgs: %v", err)
	}

	return &sendTxArgs, nil
}

func (c *SendTransactionCommand) Execute(ctx context.Context, request RPCRequest) (interface{}, error) {
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

	if !params.Valid() {
		return "", ErrSendingParamsInvalid
	}

	if params.From != dApp.SharedAccount {
		return "", ErrParamsFromAddressIsNotShared
	}

	if params.Value == nil {
		params.Value = (*hexutil.Big)(big.NewInt(0))
	}

	if params.GasPrice == nil || (params.MaxFeePerGas == nil && params.MaxPriorityFeePerGas == nil) {
		feeManager := &fees.FeeManager{
			RPCClient: c.RpcClient,
		}
		fetchedFees, err := feeManager.SuggestedFees(ctx, dApp.ChainID)
		if err != nil {
			return "", err
		}

		if !fetchedFees.EIP1559Enabled {
			params.GasPrice = (*hexutil.Big)(fetchedFees.GasPrice)
		} else {
			params.MaxFeePerGas = (*hexutil.Big)(fetchedFees.FeeFor(fees.GasFeeMedium))
			params.MaxPriorityFeePerGas = (*hexutil.Big)(fetchedFees.MaxPriorityFeePerGas)
		}
	}

	if params.Nonce == nil {
		ethClient, err := c.RpcClient.EthClient(dApp.ChainID)
		if err != nil {
			return "", err
		}

		nonce, err := ethClient.PendingNonceAt(ctx, common.Address(dApp.SharedAccount))
		if err != nil {
			return "", err
		}

		params.Nonce = (*hexutil.Uint64)(&nonce)
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
