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

	ErrWalletResponseTimeout         = fmt.Errorf("timeout waiting for wallet response")
	ErrEmptyAccountsShared           = fmt.Errorf("empty accounts were shared by wallet")
	ErrRequestAccountsRejectedByUser = fmt.Errorf("request accounts was rejected by user")
	ErrSendTransactionRejectedByUser = fmt.Errorf("send transaction was rejected by user")
	ErrEmptyRequestID                = fmt.Errorf("empty requestID")
)

type MessageType int

const (
	RequestAccountsAccepted MessageType = iota
	SendTransactionAccepted
	Rejected
)

type Message struct {
	Type MessageType
	Data interface{}
}

type ClientSideHandler struct {
	responseChannel chan Message
}

func NewClientSideHandler() *ClientSideHandler {
	return &ClientSideHandler{
		responseChannel: make(chan Message, 1), // Buffer of 1 to avoid blocking
	}
}

func (c *ClientSideHandler) generateRequestID(dApp signal.ConnectorDApp) string {
	return fmt.Sprintf("%d%s", time.Now().UnixMilli(), dApp.URL)
}

func (c *ClientSideHandler) RequestShareAccountForDApp(dApp signal.ConnectorDApp) (types.Address, uint64, error) {
	requestID := c.generateRequestID(dApp)
	signal.SendConnectorSendRequestAccounts(dApp, requestID)

	timeout := time.After(WalletResponseMaxInterval)

	for {
		select {
		case msg := <-c.responseChannel:
			switch msg.Type {
			case RequestAccountsAccepted:
				response := msg.Data.(RequestAccountsAcceptedArgs)
				if response.RequestID == requestID {
					return response.Account, response.ChainID, nil
				}
			case Rejected:
				response := msg.Data.(RejectedArgs)
				if response.RequestID == requestID {
					return types.Address{}, 0, ErrRequestAccountsRejectedByUser
				}
			}
		case <-timeout:
			return types.Address{}, 0, ErrWalletResponseTimeout
		}
	}
}

func (c *ClientSideHandler) RequestSendTransaction(dApp signal.ConnectorDApp, chainID uint64, txArgs *transactions.SendTxArgs) (types.Hash, error) {
	txArgsJson, err := json.Marshal(txArgs)
	if err != nil {
		return types.Hash{}, fmt.Errorf("failed to marshal txArgs: %v", err)
	}

	requestID := c.generateRequestID(dApp)
	signal.SendConnectorSendTransaction(dApp, chainID, string(txArgsJson), requestID)

	timeout := time.After(WalletResponseMaxInterval)

	for {
		select {
		case msg := <-c.responseChannel:
			switch msg.Type {
			case SendTransactionAccepted:
				response := msg.Data.(SendTransactionAcceptedArgs)
				if response.RequestID == requestID {
					return response.Hash, nil
				}
			case Rejected:
				response := msg.Data.(RejectedArgs)
				if response.RequestID == requestID {
					return types.Hash{}, ErrSendTransactionRejectedByUser
				}
			}
		case <-timeout:
			return types.Hash{}, ErrWalletResponseTimeout
		}
	}
}

func (c *ClientSideHandler) RequestAccountsAccepted(args RequestAccountsAcceptedArgs) error {
	c.responseChannel <- Message{Type: RequestAccountsAccepted, Data: args}
	return nil
}

func (c *ClientSideHandler) RequestAccountsRejected(args RejectedArgs) error {
	if args.RequestID == "" {
		return ErrEmptyRequestID
	}

	c.responseChannel <- Message{Type: Rejected, Data: args}
	return nil
}

func (c *ClientSideHandler) SendTransactionAccepted(args SendTransactionAcceptedArgs) error {
	if args.RequestID == "" {
		return ErrEmptyRequestID
	}

	c.responseChannel <- Message{Type: SendTransactionAccepted, Data: args}
	return nil
}

func (c *ClientSideHandler) SendTransactionRejected(args RejectedArgs) error {
	if args.RequestID == "" {
		return ErrEmptyRequestID
	}

	c.responseChannel <- Message{Type: Rejected, Data: args}
	return nil
}
