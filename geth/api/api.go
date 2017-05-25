package api

import (
	"sync"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
)

// StatusAPI provides API to access Status related functionality.
type StatusAPI struct {
	sync.Mutex
	b *StatusBackend
}

// NewStatusAPI create a new StatusAPI instance
func NewStatusAPI() *StatusAPI {
	return &StatusAPI{
		b: NewStatusBackend(),
	}
}

// NodeManager returns reference to node manager
func (api *StatusAPI) NodeManager() common.NodeManager {
	api.Lock()
	defer api.Unlock()

	return api.b.NodeManager()
}

// AccountManager returns reference to account manager
func (api *StatusAPI) AccountManager() common.AccountManager {
	api.Lock()
	defer api.Unlock()

	return api.b.AccountManager()
}

// JailManager returns reference to jail
func (api *StatusAPI) JailManager() common.JailManager {
	api.Lock()
	defer api.Unlock()

	return api.b.JailManager()
}

// StartNode start Status node, fails if node is already started
func (api *StatusAPI) StartNode(config *params.NodeConfig) error {
	api.Lock()
	defer api.Unlock()

	nodeStarted, err := api.b.StartNode(config)
	if err != nil {
		return err
	}

	<-nodeStarted // do not return up until backend is ready
	return nil
}

// StartNodeNonBlocking start Status node, fails if node is already started
// Returns immediately w/o waiting for node to start (relies on listening
// for node.started signal)
func (api *StatusAPI) StartNodeNonBlocking(config *params.NodeConfig) error {
	api.Lock()
	defer api.Unlock()

	_, err := api.b.StartNode(config)
	if err != nil {
		return err
	}

	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (api *StatusAPI) StopNode() error {
	api.Lock()
	defer api.Unlock()

	return api.b.StopNode()
}

// RestartNode restart running Status node, fails if node is not running
func (api *StatusAPI) RestartNode() error {
	api.Lock()
	defer api.Unlock()

	nodeStarted, err := api.b.RestartNode()
	if err != nil {
		return err
	}
	<-nodeStarted // do not return up until backend is ready
	return nil
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (api *StatusAPI) ResetChainData() error {
	api.Lock()
	defer api.Unlock()

	nodeStarted, err := api.b.ResetChainData()
	if err != nil {
		return err
	}
	<-nodeStarted // do not return up until backend is ready
	return nil
}

// PopulateStaticPeers connects current node with our publicly available LES/SHH/Swarm cluster
func (api *StatusAPI) PopulateStaticPeers() error {
	api.Lock()
	defer api.Unlock()

	return api.b.nodeManager.PopulateStaticPeers()
}

// AddPeer adds new static peer node
func (api *StatusAPI) AddPeer(url string) error {
	api.Lock()
	defer api.Unlock()

	return api.b.nodeManager.AddPeer(url)
}

// CreateAccount creates an internal geth account
// BIP44-compatible keys are generated: CKD#1 is stored as account key, CKD#2 stored as sub-account root
// Public key of CKD#1 is returned, with CKD#2 securely encoded into account key file (to be used for
// sub-account derivations)
func (api *StatusAPI) CreateAccount(password string) (address, pubKey, mnemonic string, err error) {
	api.Lock()
	defer api.Unlock()

	return api.b.CreateAccount(password)
}

// CreateChildAccount creates sub-account for an account identified by parent address.
// CKD#2 is used as root for master accounts (when parentAddress is "").
// Otherwise (when parentAddress != ""), child is derived directly from parent.
func (api *StatusAPI) CreateChildAccount(parentAddress, password string) (address, pubKey string, err error) {
	api.Lock()
	defer api.Unlock()

	return api.b.CreateChildAccount(parentAddress, password)
}

// RecoverAccount re-creates master key using given details.
// Once master key is re-generated, it is inserted into keystore (if not already there).
func (api *StatusAPI) RecoverAccount(password, mnemonic string) (address, pubKey string, err error) {
	api.Lock()
	defer api.Unlock()

	return api.b.RecoverAccount(password, mnemonic)
}

// VerifyAccountPassword tries to decrypt a given account key file, with a provided password.
// If no error is returned, then account is considered verified.
func (api *StatusAPI) VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error) {
	api.Lock()
	defer api.Unlock()

	return api.b.VerifyAccountPassword(keyStoreDir, address, password)
}

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (api *StatusAPI) SelectAccount(address, password string) error {
	api.Lock()
	defer api.Unlock()

	return api.b.SelectAccount(address, password)
}

// Logout clears whisper identities
func (api *StatusAPI) Logout() error {
	api.Lock()
	defer api.Unlock()

	return api.b.Logout()
}

// CompleteTransaction instructs backend to complete sending of a given transaction
func (api *StatusAPI) CompleteTransaction(id, password string) (gethcommon.Hash, error) {
	api.Lock()
	defer api.Unlock()

	return api.b.CompleteTransaction(id, password)
}

// CompleteTransactions instructs backend to complete sending of multiple transactions
func (api *StatusAPI) CompleteTransactions(ids, password string) map[string]common.RawCompleteTransactionResult {
	api.Lock()
	defer api.Unlock()

	return api.b.CompleteTransactions(ids, password)
}

// DiscardTransaction discards a given transaction from transaction queue
func (api *StatusAPI) DiscardTransaction(id string) error {
	api.Lock()
	defer api.Unlock()

	return api.b.DiscardTransaction(id)
}

// DiscardTransactions discards given multiple transactions from transaction queue
func (api *StatusAPI) DiscardTransactions(ids string) map[string]common.RawDiscardTransactionResult {
	api.Lock()
	defer api.Unlock()

	return api.b.DiscardTransactions(ids)
}

// Parse creates a new jail cell context, with the given chatID as identifier.
// New context executes provided JavaScript code, right after the initialization.
func (api *StatusAPI) JailParse(chatID string, js string) string {
	api.Lock()
	defer api.Unlock()

	return api.b.jailManager.Parse(chatID, js)
}

// Call executes given JavaScript function w/i a jail cell context identified by the chatID.
// Jail cell is clonned before call is executed i.e. all calls execute w/i their own contexts.
func (api *StatusAPI) JailCall(chatID string, path string, args string) string {
	api.Lock()
	defer api.Unlock()

	return api.b.jailManager.Call(chatID, path, args)

}

// BaseJS allows to setup initial JavaScript to be loaded on each jail.Parse()
func (api *StatusAPI) JailBaseJS(js string) {
	api.Lock()
	defer api.Unlock()

	api.b.jailManager.BaseJS(js)
}
