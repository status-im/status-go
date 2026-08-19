package commands

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/signal"
	persistence "github.com/status-im/status-go/services/connector/database"
	"github.com/status-im/status-go/services/wallet/wallettypes"
)

var (
	WalletResponseMaxInterval = 20 * time.Minute

	ErrWalletResponseTimeout                  = fmt.Errorf("timeout waiting for wallet response")
	ErrEmptyAccountsShared                    = fmt.Errorf("empty accounts were shared by wallet")
	ErrRequestAccountsRejectedByUser          = fmt.Errorf("request accounts was rejected by user")
	ErrSendTransactionRejectedByUser          = fmt.Errorf("send transaction was rejected by user")
	ErrSignRejectedByUser                     = fmt.Errorf("sign was rejected by user")
	ErrEmptyRequestID                         = fmt.Errorf("empty requestID")
	ErrAnotherConnectorOperationIsAwaitingFor = fmt.Errorf("another connector operation is awaiting for user input")
	ErrEmptyUrl                               = fmt.Errorf("empty URL")
	ErrDAppDoesNotHavePermissions             = fmt.Errorf("dApp does not have permissions")
)

type MessageType int

const (
	RequestAccountsAccepted MessageType = iota
	SendTransactionAccepted
	SignAccepted
	Rejected
)

type Message struct {
	Type MessageType
	Data interface{}
}

type sharePendingKey struct {
	normalizedURL string
	clientID      string
}

type ClientSideHandler struct {
	Db                    *sql.DB
	wcSessionDisconnector WCSessionDisconnector
	responseChannel       chan Message
	isRequestRunning      int32

	sharePendingMu sync.Mutex
	sharePending   map[sharePendingKey]chan struct{}
}

func NewClientSideHandler(db *sql.DB, getClient WCClientGetter) *ClientSideHandler {
	return &ClientSideHandler{
		Db:                    db,
		wcSessionDisconnector: NewWCSessionDisconnector(db, getClient),
		responseChannel:       make(chan Message, 1),
		isRequestRunning:      0,
		sharePending:          make(map[sharePendingKey]chan struct{}),
	}
}

func (c *ClientSideHandler) sharePendingKey(url, clientID string) sharePendingKey {
	return sharePendingKey{
		normalizedURL: persistence.NormalizeURL(url),
		clientID:      clientID,
	}
}

// BeginPendingShareAccount marks that a share-account flow is in progress for this origin.
// Called from shareAndUpsertDApp before RequestShareAccountForDApp; End runs after UpsertDApp.
// At most one pending entry exists per (url, clientID) without ref-counting.
func (c *ClientSideHandler) BeginPendingShareAccount(url, clientID string) {
	key := c.sharePendingKey(url, clientID)
	c.sharePendingMu.Lock()
	defer c.sharePendingMu.Unlock()
	if _, exists := c.sharePending[key]; exists {
		return
	}
	c.sharePending[key] = make(chan struct{})
}

// EndPendingShareAccount clears the pending marker after the flow finished (including DB persist).
func (c *ClientSideHandler) EndPendingShareAccount(url, clientID string) {
	key := c.sharePendingKey(url, clientID)
	c.sharePendingMu.Lock()
	done, ok := c.sharePending[key]
	if ok {
		delete(c.sharePending, key)
	}
	c.sharePendingMu.Unlock()
	if ok {
		close(done)
	}
}

// WaitForPendingShareAccount blocks until there is no in-flight share flow for this origin, or ctx is cancelled.
// Returns true if a pending share entry was observed (the call may have blocked on it).
func (c *ClientSideHandler) WaitForPendingShareAccount(ctx context.Context, url, clientID string) bool {
	key := c.sharePendingKey(url, clientID)
	sawPending := false
	for {
		c.sharePendingMu.Lock()
		done := c.sharePending[key]
		c.sharePendingMu.Unlock()
		if done == nil {
			return sawPending
		}
		sawPending = true
		select {
		case <-ctx.Done():
			return sawPending
		case <-done:
			return sawPending
		}
	}
}

func (c *ClientSideHandler) generateRequestID(dApp signal.ConnectorDApp) string {
	rawID := fmt.Sprintf("%d%s%s", time.Now().UnixMilli(), dApp.URL, dApp.ClientID)
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

	dApp, err := persistence.SelectDApp(c.Db, args.URL, args.ClientID)
	if err != nil {
		return err
	}

	if dApp == nil {
		return ErrDAppDoesNotHavePermissions
	}

	// If this is a WalletConnect DApp, disconnect all WC sessions first
	if c.wcSessionDisconnector != nil && dApp.ClientID == persistence.WCClientID {
		sessions, err := persistence.SelectWCSessionsByDAppURL(c.Db, dApp.URL)
		if err == nil && len(sessions) > 0 {
			for _, session := range sessions {
				_ = c.wcSessionDisconnector.DisconnectSession(context.Background(), session.Topic)
			}
		}
	}

	err = persistence.DeleteDApp(c.Db, dApp.URL, dApp.ClientID)
	if err != nil {
		return err
	}

	signal.SendConnectorDAppPermissionRevoked(signal.ConnectorDApp{
		URL:      dApp.URL,
		Name:     dApp.Name,
		IconURL:  dApp.IconURL,
		ClientID: dApp.ClientID,
	})
	return nil
}

func (c *ClientSideHandler) RequestSendTransaction(dApp signal.ConnectorDApp, chainID uint64, txArgs *wallettypes.SendTxArgs) (types.Hash, error) {
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

func (c *ClientSideHandler) RequestSign(dApp signal.ConnectorDApp, challenge, address string, method string) (string, error) {
	if !c.setRequestRunning() {
		return "", ErrAnotherConnectorOperationIsAwaitingFor
	}
	defer c.clearRequestRunning()

	requestID := c.generateRequestID(dApp)
	signal.SendConnectorSign(dApp, requestID, challenge, address, method)

	timeout := time.After(WalletResponseMaxInterval)

	for {
		select {
		case msg := <-c.responseChannel:
			switch msg.Type {
			case SignAccepted:
				response := msg.Data.(SignAcceptedArgs)
				if response.RequestID == requestID {
					return response.Signature, nil
				}
			case Rejected:
				response := msg.Data.(RejectedArgs)
				if response.RequestID == requestID {
					return "", ErrSignRejectedByUser
				}
			}
		case <-timeout:
			return "", ErrWalletResponseTimeout
		}
	}
}

func (c *ClientSideHandler) SignAccepted(args SignAcceptedArgs) error {
	if args.RequestID == "" {
		return ErrEmptyRequestID
	}

	c.responseChannel <- Message{Type: SignAccepted, Data: args}
	return nil
}

func (c *ClientSideHandler) SignRejected(args RejectedArgs) error {
	if args.RequestID == "" {
		return ErrEmptyRequestID
	}

	c.responseChannel <- Message{Type: Rejected, Data: args}
	return nil
}
