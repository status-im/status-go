package commands

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/status-im/status-go/eth-node/types"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

var (
	WalletResponseMaxInterval = 20 * time.Minute

	ErrWalletResponseTimeout                  = fmt.Errorf("timeout waiting for wallet response")
	ErrEmptyAccountsShared                    = fmt.Errorf("empty accounts were shared by wallet")
	ErrRequestAccountsRejectedByUser          = fmt.Errorf("request accounts was rejected by user")
	ErrSendTransactionRejectedByUser          = fmt.Errorf("send transaction was rejected by user")
	ErrPersonalSignRejectedByUser             = fmt.Errorf("personal sign was rejected by user")
	ErrEmptyRequestID                         = fmt.Errorf("empty requestID")
	ErrAnotherConnectorOperationIsAwaitingFor = fmt.Errorf("another connector operation is awaiting for user input")
	ErrEmptyUrl                               = fmt.Errorf("empty URL")
	ErrDAppDoesNotHavePermissions             = fmt.Errorf("dApp does not have permissions")
)

type MessageType int

const (
	RequestAccountsAccepted MessageType = iota
	SendTransactionAccepted
	PersonalSignAccepted
	Rejected
)

type Message struct {
	Type MessageType
	Data interface{}
}

type ClientSideHandler struct {
	Db               *sql.DB
	responseChannel  chan Message
	isRequestRunning int32
}

func NewClientSideHandler(db *sql.DB) *ClientSideHandler {
	return &ClientSideHandler{
		Db:               db,
		responseChannel:  make(chan Message, 1), // Buffer of 1 to avoid blocking
		isRequestRunning: 0,
	}
}

func (c *ClientSideHandler) generateRequestID(dApp signal.ConnectorDApp) string {
	rawID := fmt.Sprintf("%d%s", time.Now().UnixMilli(), dApp.URL)
	hash := sha256.Sum256([]byte(rawID))
	return hex.EncodeToString(hash[:])
}

func (c *ClientSideHandler) setRequestRunning() bool {
	return atomic.CompareAndSwapInt32(&c.isRequestRunning, 0, 1)
}

func (c *ClientSideHandler) clearRequestRunning() {
	atomic.StoreInt32(&c.isRequestRunning, 0)
}

func (c *ClientSideHandler) RequestShareAccountForDApp(dApp signal.ConnectorDApp) (types.Address, uint64, error) {
	if !c.setRequestRunning() {
		return types.Address{}, 0, ErrAnotherConnectorOperationIsAwaitingFor
	}
	defer c.clearRequestRunning()

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

func (c *ClientSideHandler) RequestAccountsAccepted(args RequestAccountsAcceptedArgs) error {
	if args.RequestID == "" {
		return ErrEmptyRequestID
	}

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

func (c *ClientSideHandler) RecallDAppPermissions(args RecallDAppPermissionsArgs) error {
	if args.URL == "" {
		return ErrEmptyUrl
	}

	dApp, err := persistence.SelectDAppByUrl(c.Db, args.URL)
	if err != nil {
		return err
	}

	if dApp == nil {
		return ErrDAppDoesNotHavePermissions
	}

	err = persistence.DeleteDApp(c.Db, dApp.URL)
	if err != nil {
		return err
	}

	signal.SendConnectorDAppPermissionRevoked(signal.ConnectorDApp{
		URL:     dApp.URL,
		Name:    dApp.Name,
		IconURL: dApp.IconURL,
	})
	return nil
}

func (c *ClientSideHandler) RequestSendTransaction(dApp signal.ConnectorDApp, chainID uint64, txArgs *transactions.SendTxArgs) (types.Hash, error) {
	if !c.setRequestRunning() {
		return types.Hash{}, ErrAnotherConnectorOperationIsAwaitingFor
	}
	defer c.clearRequestRunning()

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

func (c *ClientSideHandler) RequestPersonalSign(dApp signal.ConnectorDApp, challenge, address string) (string, error) {
	if !c.setRequestRunning() {
		return "", ErrAnotherConnectorOperationIsAwaitingFor
	}
	defer c.clearRequestRunning()

	requestID := c.generateRequestID(dApp)
	signal.SendConnectorPersonalSign(dApp, requestID, challenge, address)

	timeout := time.After(WalletResponseMaxInterval)

	for {
		select {
		case msg := <-c.responseChannel:
			switch msg.Type {
			case PersonalSignAccepted:
				response := msg.Data.(PersonalSignAcceptedArgs)
				if response.RequestID == requestID {
					return response.Signature, nil
				}
			case Rejected:
				response := msg.Data.(RejectedArgs)
				if response.RequestID == requestID {
					return "", ErrPersonalSignRejectedByUser
				}
			}
		case <-timeout:
			return "", ErrWalletResponseTimeout
		}
	}
}

func (c *ClientSideHandler) PersonalSignAccepted(args PersonalSignAcceptedArgs) error {
	if args.RequestID == "" {
		return ErrEmptyRequestID
	}

	c.responseChannel <- Message{Type: PersonalSignAccepted, Data: args}
	return nil
}

func (c *ClientSideHandler) PersonalSignRejected(args RejectedArgs) error {
	if args.RequestID == "" {
		return ErrEmptyRequestID
	}

	c.responseChannel <- Message{Type: Rejected, Data: args}
	return nil
}
