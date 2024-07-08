package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

var (
	WalletResponseMaxInterval = 20 * time.Minute

	ErrWalletResponseTimeout = fmt.Errorf("timeout waiting for wallet response")
	ErrEmptyAccountsShared   = fmt.Errorf("empty accounts were shared by wallet")
)

type ClientSideHandler struct {
	requestAccountsResponseChannel chan RequestAccountsFinishedArgs
	sendTransactionResponseChannel chan SendTransactionFinishedArgs
}

func NewClientSideHandler() *ClientSideHandler {
	return &ClientSideHandler{
		// Buffer of 1 to avoid blocking
		sendTransactionResponseChannel: make(chan SendTransactionFinishedArgs, 1),
		requestAccountsResponseChannel: make(chan RequestAccountsFinishedArgs, 1),
	}
}

func (c *ClientSideHandler) RequestShareAccountForDApp(dApp DAppData) (types.Address, error) {
	signal.SendConnectorSendRequestAccounts(dApp.Origin, dApp.Name, dApp.IconUrl)

	select {
	case response := <-c.requestAccountsResponseChannel:
		if response.Error != nil {
			return types.Address{}, *response.Error
		}
		if len(response.Accounts) == 0 {
			return types.Address{}, ErrEmptyAccountsShared
		}
		return response.Accounts[0], nil
	case <-time.After(WalletResponseMaxInterval):
		return types.Address{}, ErrWalletResponseTimeout
	}
}

func (c *ClientSideHandler) RequestSendTransaction(dApp DAppData, chainID uint64, txArgs *transactions.SendTxArgs) (types.Hash, error) {
	txArgsJson, err := json.Marshal(txArgs)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to marshal txArgs: %v", err)
	}

	signal.SendConnectorSendTransaction(dApp.Origin, chainID, string(txArgsJson))

	select {
	case response := <-c.sendTransactionResponseChannel:
		if response.Error != nil {
			return types.Hash{}, *response.Error
		}
		return response.Hash, nil
	case <-time.After(WalletResponseMaxInterval):
		return types.Hash{}, ErrWalletResponseTimeout
	}
}

func (c *ClientSideHandler) RequestAccountsFinished(args RequestAccountsFinishedArgs) error {
	// Notify RequestShareAccountForDApp with address or error
	c.requestAccountsResponseChannel <- args
	return nil
}

func (c *ClientSideHandler) SendTransactionFinished(args SendTransactionFinishedArgs) error {
	// Notify RequestSendTransaction with hash or error
	c.sendTransactionResponseChannel <- args
	return nil
}
