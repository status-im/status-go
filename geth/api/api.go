package api

import (
	"context"

	"github.com/NaySoftware/go-fcm"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/account"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/transactions"
	"github.com/status-im/status-go/sign"
)

// StatusAPI provides API to access Status related functionality.
type StatusAPI struct {
	b   *StatusBackend
	log log.Logger
}

// NewStatusAPI creates a new StatusAPI instance
func NewStatusAPI() *StatusAPI {
	return NewStatusAPIWithBackend(NewStatusBackend())
}

// NewStatusAPIWithBackend creates a new StatusAPI instance using
// the passed backend.
func NewStatusAPIWithBackend(b *StatusBackend) *StatusAPI {
	return &StatusAPI{
		b:   b,
		log: log.New("package", "status-go/geth/api.StatusAPI"),
	}
}

// StatusNode returns reference to StatusNode.
func (api *StatusAPI) StatusNode() *node.StatusNode {
	return api.b.StatusNode()
}

// AccountManager returns reference to account manager
func (api *StatusAPI) AccountManager() *account.Manager {
	return api.b.AccountManager()
}

// JailManager returns reference to jail
func (api *StatusAPI) JailManager() jail.Manager {
	return api.b.JailManager()
}

// Transactor returns reference to a status transactor
func (api *StatusAPI) Transactor() *transactions.Transactor {
	return api.b.Transactor()
}

// PendingSignRequests returns reference to a list of current sign requests
func (api *StatusAPI) PendingSignRequests() *sign.PendingRequests {
	return api.b.PendingSignRequests()
}

// StartNode start Status node, fails if node is already started
func (api *StatusAPI) StartNode(config *params.NodeConfig) error {
	return api.b.StartNode(config)
}

// StartNodeAsync start Status node, fails if node is already started
// Returns immediately w/o waiting for node to start (see node.ready)
func (api *StatusAPI) StartNodeAsync(config *params.NodeConfig) <-chan error {
	return runAsync(func() error { return api.StartNode(config) })
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (api *StatusAPI) StopNode() error {
	return api.b.StopNode()
}

// StopNodeAsync stop Status node. Stopped node cannot be resumed.
// Returns immediately, w/o waiting for node to stop (see node.stopped)
func (api *StatusAPI) StopNodeAsync() <-chan error {
	return runAsync(api.StopNode)
}

// RestartNode restart running Status node, fails if node is not running
func (api *StatusAPI) RestartNode() error {
	return api.b.RestartNode()
}

// RestartNodeAsync restart running Status node, in async manner
func (api *StatusAPI) RestartNodeAsync() <-chan error {
	return runAsync(api.RestartNode)
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (api *StatusAPI) ResetChainData() error {
	return api.b.ResetChainData()
}

// ResetChainDataAsync remove chain data from data directory, in async manner
func (api *StatusAPI) ResetChainDataAsync() <-chan error {
	return runAsync(api.ResetChainData)
}

// CallRPC executes public RPC requests on node's in-proc RPC server.
func (api *StatusAPI) CallRPC(inputJSON string) string {
	return api.b.CallRPC(inputJSON)
}

// CallPrivateRPC executes public and private RPC requests on node's in-proc RPC server.
func (api *StatusAPI) CallPrivateRPC(inputJSON string) string {
	return api.b.CallPrivateRPC(inputJSON)
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (api *StatusAPI) CreateAccount(password string) (address, pubKey, mnemonic string, err error) {
	return api.b.AccountManager().CreateAccount(password)
}

// CreateChildAccount creates sub-account for an account identified by parent address.
// CKD#2 is used as root for master accounts (when parentAddress is "").
// Otherwise (when parentAddress != ""), child is derived directly from parent.
func (api *StatusAPI) CreateChildAccount(parentAddress, password string) (address, pubKey string, err error) {
	return api.b.AccountManager().CreateChildAccount(parentAddress, password)
}

// RecoverAccount re-creates master key using given details.
// Once master key is re-generated, it is inserted into keystore (if not already there).
func (api *StatusAPI) RecoverAccount(password, mnemonic string) (address, pubKey string, err error) {
	return api.b.AccountManager().RecoverAccount(password, mnemonic)
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
	return api.b.SelectAccount(address, password)
}

// Logout clears whisper identities
func (api *StatusAPI) Logout() error {
	return api.b.Logout()
}

// SendTransaction creates a new transaction and waits until it's complete.
func (api *StatusAPI) SendTransaction(ctx context.Context, args transactions.SendTxArgs) (gethcommon.Hash, error) {
	return api.b.SendTransaction(ctx, args)
}

// ApproveSignRequest instructs backend to complete sending of a given transaction
func (api *StatusAPI) ApproveSignRequest(id string, password string) sign.Result {
	return api.b.ApproveSignRequest(id, password)
}

// ApproveSignRequests instructs backend to complete sending of multiple transactions
func (api *StatusAPI) ApproveSignRequests(ids []string, password string) map[string]sign.Result {
	return api.b.ApproveSignRequests(ids, password)
}

// DiscardSignRequest discards a given transaction from transaction queue
func (api *StatusAPI) DiscardSignRequest(id string) error {
	return api.b.DiscardSignRequest(id)
}

// DiscardSignRequests discards given multiple transactions from transaction queue
func (api *StatusAPI) DiscardSignRequests(ids []string) map[string]error {
	return api.b.DiscardSignRequests(ids)
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

// NotifyUsers send notifications to users.
func (api *StatusAPI) NotifyUsers(message string, payload fcm.NotificationPayload, tokens ...string) error {
	return api.b.NotifyUsers(message, payload, tokens...)
}

// ConnectionChange handles network state changes logic.
func (api *StatusAPI) ConnectionChange(typ string, expensive bool) {
	api.b.ConnectionChange(typ, expensive)
}

// AppStateChange handles app state changes (background/foreground).
// state values: see https://facebook.github.io/react-native/docs/appstate.html
func (api *StatusAPI) AppStateChange(state string) {
	api.b.AppStateChange(state)
}
