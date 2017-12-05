package api

import (
	"context"
	"fmt"
	"os"

	"github.com/NaySoftware/go-fcm"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
	validator "gopkg.in/go-playground/validator.v9"
)

// StatusAPI provides API to access Status related functionality.
type StatusAPI struct {
	b *StatusBackend
}

// NewStatusAPI creates a new StatusAPI instance
func NewStatusAPI() *StatusAPI {
	return NewStatusAPIWithBackend(NewStatusBackend())
}

// NewStatusAPIWithBackend creates a new StatusAPI instance using
// the passed backend.
func NewStatusAPIWithBackend(b *StatusBackend) *StatusAPI {
	return &StatusAPI{
		b: b,
	}
}

// NodeManager returns reference to node manager
func (api *StatusAPI) NodeManager() common.NodeManager {
	return api.b.NodeManager()
}

// AccountManager returns reference to account manager
func (api *StatusAPI) AccountManager() common.AccountManager {
	return api.b.AccountManager()
}

// JailManager returns reference to jail
func (api *StatusAPI) JailManager() common.JailManager {
	return api.b.JailManager()
}

// TxQueueManager returns reference to account manager
func (api *StatusAPI) TxQueueManager() common.TxQueueManager {
	return api.b.TxQueueManager()
}

// StartNode start Status node, fails if node is already started
func (api *StatusAPI) StartNode(config *params.NodeConfig) error {
	nodeStarted, err := api.b.StartNode(config)
	if err != nil {
		return err
	}
	<-nodeStarted
	return nil
}

// StartNodeAsync start Status node, fails if node is already started
// Returns immediately w/o waiting for node to start (see node.ready)
func (api *StatusAPI) StartNodeAsync(config *params.NodeConfig) (<-chan struct{}, error) {
	return api.b.StartNode(config)
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (api *StatusAPI) StopNode() error {
	nodeStopped, err := api.b.StopNode()
	if err != nil {
		return err
	}
	<-nodeStopped
	return nil
}

// StopNodeAsync stop Status node. Stopped node cannot be resumed.
// Returns immediately, w/o waiting for node to stop (see node.stopped)
func (api *StatusAPI) StopNodeAsync() (<-chan struct{}, error) {
	return api.b.StopNode()
}

// RestartNode restart running Status node, fails if node is not running
func (api *StatusAPI) RestartNode() error {
	nodeStarted, err := api.b.RestartNode()
	if err != nil {
		return err
	}
	<-nodeStarted // do not return up until backend is ready
	return nil
}

// RestartNodeAsync restart running Status node, in async manner
func (api *StatusAPI) RestartNodeAsync() (<-chan struct{}, error) {
	return api.b.RestartNode()
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (api *StatusAPI) ResetChainData() error {
	nodeStarted, err := api.b.ResetChainData()
	if err != nil {
		return err
	}
	<-nodeStarted // do not return up until backend is ready
	return nil
}

// ResetChainDataAsync remove chain data from data directory, in async manner
func (api *StatusAPI) ResetChainDataAsync() (<-chan struct{}, error) {
	return api.b.ResetChainData()
}

// CallRPC executes RPC request on node's in-proc RPC server
func (api *StatusAPI) CallRPC(inputJSON string) string {
	return api.b.CallRPC(inputJSON)
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (api *StatusAPI) CreateAccount(password string) common.AccountInfo {
	address, pubKey, mnemonic, err := api.b.AccountManager().CreateAccount(password)
	errString := processError(err)
	return common.AccountInfo{
		Address:    address,
		PubKey:     pubKey,
		Mnemonic:   mnemonic,
		ErrorValue: err,
		Error:      errString,
	}
}

// CreateChildAccount creates sub-account for an account identified by parent address.
// CKD#2 is used as root for master accounts (when parentAddress is "").
// Otherwise (when parentAddress != ""), child is derived directly from parent.
func (api *StatusAPI) CreateChildAccount(parentAddress, password string) common.AccountInfo {
	address, pubKey, err := api.b.AccountManager().CreateChildAccount(parentAddress, password)
	errString := processError(err)
	return common.AccountInfo{
		Address:    address,
		PubKey:     pubKey,
		ErrorValue: err,
		Error:      errString,
	}

}

// RecoverAccount re-creates master key using given details.
// Once master key is re-generated, it is inserted into keystore (if not already there).
func (api *StatusAPI) RecoverAccount(password, mnemonic string) common.AccountInfo {
	address, pubKey, err := api.b.AccountManager().RecoverAccount(password, mnemonic)
	errString := processError(err)
	return common.AccountInfo{
		Address:    address,
		PubKey:     pubKey,
		Mnemonic:   mnemonic,
		ErrorValue: err,
		Error:      errString,
	}

}

// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
// If no error is returned, then account is considered verified.
func (api *StatusAPI) VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error) {
	return api.b.AccountManager().VerifyAccountPassword(keyStoreDir, address, password)
}

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (api *StatusAPI) SelectAccount(address, password string) error {
	// FIXME(oleg-raev): This method doesn't make stop, it rather resets its cells to an initial state
	// and should be properly renamed, for example: ResetCells
	api.b.jailManager.Stop()
	return api.b.AccountManager().SelectAccount(address, password)
}

// Logout clears whisper identities
func (api *StatusAPI) Logout() error {
	api.b.jailManager.Stop()
	return api.b.AccountManager().Logout()
}

// SendTransaction creates a new transaction and waits until it's complete.
func (api *StatusAPI) SendTransaction(ctx context.Context, args common.SendTxArgs) (gethcommon.Hash, error) {
	return api.b.SendTransaction(ctx, args)
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (api *StatusAPI) CompleteTransaction(id common.QueuedTxID, password string) (gethcommon.Hash, error) {
	return api.b.txQueueManager.CompleteTransaction(id, password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (api *StatusAPI) CompleteTransactions(ids []common.QueuedTxID, password string) map[common.QueuedTxID]common.RawCompleteTransactionResult {
	return api.b.txQueueManager.CompleteTransactions(ids, password)
}

// DiscardTransaction discards a given transaction from transaction queue
func (api *StatusAPI) DiscardTransaction(id common.QueuedTxID) error {
	return api.b.txQueueManager.DiscardTransaction(id)
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (api *StatusAPI) DiscardTransactions(ids []common.QueuedTxID) map[common.QueuedTxID]common.RawDiscardTransactionResult {
	return api.b.txQueueManager.DiscardTransactions(ids)
}

// JailParse creates a new jail cell context, with the given chatID as identifier.
// New context executes provided JavaScript code, right after the initialization.
// DEPRECATED in favour of CreateAndInitCell.
func (api *StatusAPI) JailParse(chatID string, js string) string {
	return api.b.jailManager.Parse(chatID, js)
}

// CreateAndInitCell creates a new jail cell context, with the given chatID as identifier.
// New context executes provided JavaScript code, right after the initialization.
func (api *StatusAPI) CreateAndInitCell(chatID, js string) string {
	return api.b.jailManager.CreateAndInitCell(chatID, js)
}

// JailCall executes given JavaScript function w/i a jail cell context identified by the chatID.
func (api *StatusAPI) JailCall(chatID, this, args string) string {
	return api.b.jailManager.Call(chatID, this, args)
}

// JailExecute allows to run arbitrary JS code within a jail cell.
func (api *StatusAPI) JailExecute(chatID, code string) string {
	return api.b.jailManager.Execute(chatID, code)
}

// SetJailBaseJS allows to setup initial JavaScript to be loaded on each jail.CreateAndInitCell().
func (api *StatusAPI) SetJailBaseJS(js string) {
	api.b.jailManager.SetBaseJS(js)
}

// Notify sends a push notification to the device with the given token.
// @deprecated
func (api *StatusAPI) Notify(token string) string {
	log.Debug("Notify", "token", token)
	message := "Hello World1"

	tokens := []string{token}

	err := api.b.newNotification().Send(message, fcm.NotificationPayload{}, tokens...)
	if err != nil {
		log.Error("Notify failed:", err)
	}

	return token
}

// NotifyUsers send notifications to users.
func (api *StatusAPI) NotifyUsers(message string, payload fcm.NotificationPayload, tokens ...string) error {
	log.Debug("Notify", "tokens", tokens)

	err := api.b.newNotification().Send(message, payload, tokens...)
	if err != nil {
		log.Error("Notify failed:", err)
	}

	return err
}

func (api *StatusAPI) ValidateJSONConfig(configJSON string) common.APIDetailedResponse {
	var resp common.APIDetailedResponse
	_, err := params.LoadNodeConfig(configJSON)

	// Convert errors to common.APIDetailedResponse
	switch err := err.(type) {
	case validator.ValidationErrors:
		resp = common.APIDetailedResponse{
			Message:     "validation: validation failed",
			FieldErrors: make([]common.APIFieldError, len(err)),
		}

		for i, ve := range err {
			resp.FieldErrors[i] = common.APIFieldError{
				Parameter: ve.Namespace(),
				Errors: []common.APIError{
					{
						Message: fmt.Sprintf("field validation failed on the '%s' tag", ve.Tag()),
					},
				},
			}
		}
	case error:
		resp = common.APIDetailedResponse{
			Message: fmt.Sprintf("validation: %s", err.Error()),
		}
	case nil:
		resp = common.APIDetailedResponse{
			Status: true,
		}
	}

	return resp
}

func processError(err error) string {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err.Error()
	}
	return ""
}
