package api

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/notifications/push/fcm"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/rpcfilters"
	"github.com/status-im/status-go/services/shhext/chat"
	"github.com/status-im/status-go/services/shhext/chat/crypto"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

const (
	//todo(jeka): should be removed
	fcmServerKey = "AAAAxwa-r08:APA91bFtMIToDVKGAmVCm76iEXtA4dn9MPvLdYKIZqAlNpLJbd12EgdBI9DSDSXKdqvIAgLodepmRhGVaWvhxnXJzVpE6MoIRuKedDV3kfHSVBhWFqsyoLTwXY4xeufL9Sdzb581U-lx"
)

var (
	// ErrWhisperClearIdentitiesFailure clearing whisper identities has failed.
	ErrWhisperClearIdentitiesFailure = errors.New("failed to clear whisper identities")
	// ErrWhisperIdentityInjectionFailure injecting whisper identities has failed.
	ErrWhisperIdentityInjectionFailure = errors.New("failed to inject identity into Whisper")
	// ErrUnsupportedRPCMethod is for methods not supported by the RPC interface
	ErrUnsupportedRPCMethod = errors.New("method is unsupported by RPC interface")
	// ErrRPCClientUnavailable is returned if an RPC client can't be retrieved.
	// This is a normal situation when a node is stopped.
	ErrRPCClientUnavailable = errors.New("JSON-RPC client is unavailable")
)

// StatusBackend implements Status.im service
type StatusBackend struct {
	mu              sync.Mutex
	statusNode      *node.StatusNode
	personalAPI     *personal.PublicAPI
	rpcFilters      *rpcfilters.Service
	accountManager  *account.Manager
	transactor      *transactions.Transactor
	newNotification fcm.NotificationConstructor
	connectionState connectionState
	appState        appState
	log             log.Logger
}

// NewStatusBackend create a new NewStatusBackend instance
func NewStatusBackend() *StatusBackend {
	defer log.Info("Status backend initialized", "version", params.Version, "commit", params.GitCommit)

	statusNode := node.New()
	accountManager := account.NewManager(statusNode)
	transactor := transactions.NewTransactor()
	personalAPI := personal.NewAPI()
	notificationManager := fcm.NewNotification(fcmServerKey)
	rpcFilters := rpcfilters.New(statusNode)

	return &StatusBackend{
		statusNode:      statusNode,
		accountManager:  accountManager,
		transactor:      transactor,
		personalAPI:     personalAPI,
		rpcFilters:      rpcFilters,
		newNotification: notificationManager,
		log:             log.New("package", "status-go/api.StatusBackend"),
	}
}

// StatusNode returns reference to node manager
func (b *StatusBackend) StatusNode() *node.StatusNode {
	return b.statusNode
}

// AccountManager returns reference to account manager
func (b *StatusBackend) AccountManager() *account.Manager {
	return b.accountManager
}

// Transactor returns reference to a status transactor
func (b *StatusBackend) Transactor() *transactions.Transactor {
	return b.transactor
}

// IsNodeRunning confirm that node is running
func (b *StatusBackend) IsNodeRunning() bool {
	return b.statusNode.IsRunning()
}

// StartNode start Status node, fails if node is already started
func (b *StatusBackend) StartNode(config *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.startNode(config); err != nil {
		signal.SendNodeCrashed(err)

		return err
	}

	return nil
}

func (b *StatusBackend) rpcFiltersService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return rpcfilters.New(b.statusNode), nil
	}
}

func (b *StatusBackend) startNode(config *params.NodeConfig) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("node crashed on start: %v", err)
		}
	}()

	// Start by validating configuration
	if err := config.Validate(); err != nil {
		return err
	}

	services := []gethnode.ServiceConstructor{}
	services = appendIf(config.UpstreamConfig.Enabled, services, b.rpcFiltersService())

	if err = b.statusNode.StartWithOptions(config, node.StartOptions{
		Services: services,
		// The peers discovery protocols are started manually after
		// `node.ready` signal is sent.
		// It was discussed in https://github.com/status-im/status-go/pull/1333.
		StartDiscovery: false,
	}); err != nil {
		return
	}
	signal.SendNodeStarted()

	b.transactor.SetNetworkID(config.NetworkID)
	b.transactor.SetRPC(b.statusNode.RPCClient(), rpc.DefaultCallTimeout)
	b.personalAPI.SetRPC(b.statusNode.RPCPrivateClient(), rpc.DefaultCallTimeout)

	if err = b.registerHandlers(); err != nil {
		b.log.Error("Handler registration failed", "err", err)
		return
	}
	b.log.Info("Handlers registered")

	if err = b.reSelectAccount(); err != nil {
		b.log.Error("Reselect account failed", "err", err)
		return
	}
	b.log.Info("Account reselected")

	if st, err := b.statusNode.StatusService(); err == nil {
		st.SetAccountManager(b.AccountManager())
	}

	if st, err := b.statusNode.PeerService(); err == nil {
		st.SetDiscoverer(b.StatusNode())
	}

	signal.SendNodeReady()

	if err := b.statusNode.StartDiscovery(); err != nil {
		return err
	}

	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (b *StatusBackend) StopNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stopNode()
}

func (b *StatusBackend) stopNode() error {
	if !b.IsNodeRunning() {
		return node.ErrNoRunningNode
	}
	defer signal.SendNodeStopped()
	return b.statusNode.Stop()
}

// RestartNode restart running Status node, fails if node is not running
func (b *StatusBackend) RestartNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.IsNodeRunning() {
		return node.ErrNoRunningNode
	}

	newcfg := *(b.statusNode.Config())
	if err := b.stopNode(); err != nil {
		return err
	}
	return b.startNode(&newcfg)
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (b *StatusBackend) ResetChainData() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	newcfg := *(b.statusNode.Config())
	if err := b.stopNode(); err != nil {
		return err
	}
	// config is cleaned when node is stopped
	if err := b.statusNode.ResetChainData(&newcfg); err != nil {
		return err
	}
	signal.SendChainDataRemoved()
	return b.startNode(&newcfg)
}

// CallRPC executes public RPC requests on node's in-proc RPC server.
func (b *StatusBackend) CallRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// CallPrivateRPC executes public and private RPC requests on node's in-proc RPC server.
func (b *StatusBackend) CallPrivateRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCPrivateClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// SendTransaction creates a new transaction and waits until it's complete.
func (b *StatusBackend) SendTransaction(sendArgs transactions.SendTxArgs, password string) (hash gethcommon.Hash, err error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(password)
	if err != nil {
		return hash, err
	}

	hash, err = b.transactor.SendTransaction(sendArgs, verifiedAccount)
	if err != nil {
		return
	}

	go b.rpcFilters.TriggerTransactionSentToUpstreamEvent(hash)

	return
}

func (b *StatusBackend) SendTransactionWithSignature(sendArgs transactions.SendTxArgs, sig []byte) (hash gethcommon.Hash, err error) {
	hash, err = b.transactor.SendTransactionWithSignature(sendArgs, sig)
	if err != nil {
		return
	}

	go b.rpcFilters.TriggerTransactionSentToUpstreamEvent(hash)

	return
}

// HashTransaction validate the transaction and returns new sendArgs and the transaction hash.
func (b *StatusBackend) HashTransaction(sendArgs transactions.SendTxArgs) (transactions.SendTxArgs, gethcommon.Hash, error) {
	return b.transactor.HashTransaction(sendArgs)
}

// SignMessage checks the pwd vs the selected account and passes on the signParams
// to personalAPI for message signature
func (b *StatusBackend) SignMessage(rpcParams personal.SignParams) (hexutil.Bytes, error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(rpcParams.Password)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return b.personalAPI.Sign(rpcParams, verifiedAccount)
}

// Recover calls the personalAPI to return address associated with the private
// key that was used to calculate the signature in the message
func (b *StatusBackend) Recover(rpcParams personal.RecoverParams) (gethcommon.Address, error) {
	return b.personalAPI.Recover(rpcParams)
}

// SignTypedData accepts data and password. Gets verified account and signs typed data.
func (b *StatusBackend) SignTypedData(typed typeddata.TypedData, password string) (hexutil.Bytes, error) {
	account, err := b.getVerifiedWalletAccount(password)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	sig, err := typeddata.Sign(typed, account.AccountKey.PrivateKey, chain)
	if err != nil {
		return hexutil.Bytes{}, err
	}
	return hexutil.Bytes(sig), err
}

func (b *StatusBackend) getVerifiedWalletAccount(password string) (*account.SelectedExtKey, error) {
	selectedWalletAccount, err := b.accountManager.SelectedWalletAccount()
	if err != nil {
		b.log.Error("failed to get a selected account", "err", err)
		return nil, err
	}
	config := b.StatusNode().Config()
	_, err = b.accountManager.VerifyAccountPassword(config.KeyStoreDir, selectedWalletAccount.Address.String(), password)
	if err != nil {
		b.log.Error("failed to verify account", "account", selectedWalletAccount.Address.String(), "error", err)
		return nil, err
	}
	return selectedWalletAccount, nil
}

// registerHandlers attaches Status callback handlers to running node
func (b *StatusBackend) registerHandlers() error {
	var clients []*rpc.Client

	if c := b.StatusNode().RPCClient(); c != nil {
		clients = append(clients, c)
	} else {
		return errors.New("RPC client unavailable")
	}

	if c := b.StatusNode().RPCPrivateClient(); c != nil {
		clients = append(clients, c)
	} else {
		return errors.New("RPC private client unavailable")
	}

	for _, client := range clients {
		client.RegisterHandler(
			params.AccountsMethodName,
			func(context.Context, ...interface{}) (interface{}, error) {
				return b.AccountManager().Accounts()
			},
		)

		client.RegisterHandler(params.SendTransactionMethodName, unsupportedMethodHandler)
		client.RegisterHandler(params.PersonalSignMethodName, unsupportedMethodHandler)
		client.RegisterHandler(params.PersonalRecoverMethodName, unsupportedMethodHandler)
	}

	return nil
}

func unsupportedMethodHandler(ctx context.Context, rpcParams ...interface{}) (interface{}, error) {
	return nil, ErrUnsupportedRPCMethod
}

// ConnectionChange handles network state changes logic.
func (b *StatusBackend) ConnectionChange(typ string, expensive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := connectionState{
		Type:      newConnectionType(typ),
		Expensive: expensive,
	}
	if typ == none {
		state.Offline = true
	}

	b.log.Info("Network state change", "old", b.connectionState, "new", state)

	b.connectionState = state

	// logic of handling state changes here
	// restart node? force peers reconnect? etc
}

// AppStateChange handles app state changes (background/foreground).
// state values: see https://facebook.github.io/react-native/docs/appstate.html
func (b *StatusBackend) AppStateChange(state string) {
	s, err := parseAppState(state)
	if err != nil {
		log.Error("AppStateChange failed, ignoring", "error", err)
		return // and do nothing
	}

	b.log.Info("App State changed", "new-state", s)
	b.appState = s

	// TODO: put node in low-power mode if the app is in background (or inactive)
	// and normal mode if the app is in foreground.
}

// Logout clears whisper identities.
func (b *StatusBackend) Logout() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	whisperService, err := b.statusNode.WhisperService()
	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.DeleteKeyPairs(); err != nil {
			return fmt.Errorf("%s: %v", ErrWhisperClearIdentitiesFailure, err)
		}
	default:
		return err
	}

	b.AccountManager().Logout()

	return nil
}

// reSelectAccount selects previously selected account, often, after node restart.
func (b *StatusBackend) reSelectAccount() error {
	selectedChatAccount, err := b.AccountManager().SelectedChatAccount()
	if selectedChatAccount == nil || err == account.ErrNoAccountSelected {
		return nil
	}

	whisperService, err := b.statusNode.WhisperService()
	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.SelectKeyPair(selectedChatAccount.AccountKey.PrivateKey); err != nil {
			return ErrWhisperIdentityInjectionFailure
		}
	default:
		return err
	}

	return nil
}

// SelectAccount selects current wallet and chat accounts, by verifying that each address has corresponding account which can be decrypted
// using provided password. Once verification is done, the decrypted chat key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (b *StatusBackend) SelectAccount(walletAddress, chatAddress, password string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := b.accountManager.SelectAccount(walletAddress, chatAddress, password)
	if err != nil {
		return err
	}

	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}

	whisperService, err := b.statusNode.WhisperService()
	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.SelectKeyPair(chatAccount.AccountKey.PrivateKey); err != nil {
			return ErrWhisperIdentityInjectionFailure
		}
	default:
		return err
	}

	if whisperService != nil {
		st, err := b.statusNode.ShhExtService()
		if err != nil {
			return err
		}

		if err := st.InitProtocolWithPassword(chatAddress, password); err != nil {
			return err
		}
	}

	return nil
}

// SendDataNotification sends data push notifications to users.
// dataPayloadJSON is a JSON string that looks like this:
// {
// 	"data": {
// 		"msg-v2": {
// 			"from": "0x2cea3bd5", // hash of sender (first 10 characters/4 bytes of sha3 hash)
// 			"to": "0xb1f89744", // hash of recipient (first 10 characters/4 bytes of sha3 hash)
// 			"id": "0x872653ad", // message ID hash (first 10 characters/4 bytes of sha3 hash)
// 		}
// 	}
// }
func (b *StatusBackend) SendDataNotification(dataPayloadJSON string, tokens ...string) error {
	log.Debug("sending data push notification")

	err := b.newNotification().Send(dataPayloadJSON, tokens...)
	if err != nil {
		b.log.Error("SendDataNotification failed", "dataPayloadJSON", dataPayloadJSON, "error", err)
	}

	return err
}

// InjectChatAccount selects the current chat account using chatKeyHex and injects the key into whisper.
func (b *StatusBackend) InjectChatAccount(chatKeyHex, encryptionKeyHex string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.accountManager.Logout()

	chatKey, err := ethcrypto.HexToECDSA(chatKeyHex)
	if err != nil {
		return err
	}

	b.accountManager.SetChatAccount(chatKey)
	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}

	whisperService, err := b.statusNode.WhisperService()
	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.SelectKeyPair(chatAccount.AccountKey.PrivateKey); err != nil {
			return ErrWhisperIdentityInjectionFailure
		}
	default:
		return err
	}

	if whisperService != nil {
		st, err := b.statusNode.ShhExtService()
		if err != nil {
			return err
		}

		if err := st.InitProtocolWithEncyptionKey(chatAccount.Address.Hex(), encryptionKeyHex); err != nil {
			return err
		}
	}

	return nil
}

func appendIf(condition bool, services []gethnode.ServiceConstructor, service gethnode.ServiceConstructor) []gethnode.ServiceConstructor {
	if !condition {
		return services
	}
	return append(services, service)
}

// CreateContactCode create or return the latest contact code
func (b *StatusBackend) CreateContactCode() (string, error) {
	selectedChatAccount, err := b.AccountManager().SelectedChatAccount()
	if err != nil {
		return "", err
	}

	st, err := b.statusNode.ShhExtService()
	if err != nil {
		return "", err
	}

	bundle, err := st.GetBundle(selectedChatAccount.AccountKey.PrivateKey)
	if err != nil {
		return "", err
	}

	return bundle.ToBase64()
}

// GetContactCode return the latest contact code
func (b *StatusBackend) GetContactCode(identity string) (string, error) {
	st, err := b.statusNode.ShhExtService()
	if err != nil {
		return "", err
	}

	publicKeyBytes, err := hex.DecodeString(identity)
	if err != nil {
		return "", err
	}

	publicKey, err := ethcrypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		return "", err
	}

	bundle, err := st.GetPublicBundle(publicKey)
	if err != nil {
		return "", err
	}

	if bundle == nil {
		return "", nil
	}

	return bundle.ToBase64()
}

// ProcessContactCode process and adds the someone else's bundle
func (b *StatusBackend) ProcessContactCode(contactCode string) error {
	selectedChatAccount, err := b.AccountManager().SelectedChatAccount()
	if err != nil {
		return err
	}

	st, err := b.statusNode.ShhExtService()
	if err != nil {
		return err
	}

	bundle, err := chat.FromBase64(contactCode)
	if err != nil {
		b.log.Error("error decoding base64", "err", err)
		return err
	}

	if _, err := st.ProcessPublicBundle(selectedChatAccount.AccountKey.PrivateKey, bundle); err != nil {
		b.log.Error("error adding bundle", "err", err)
		return err
	}

	return nil
}

// ExtractIdentityFromContactCode extract the identity of the user generating the contact code
func (b *StatusBackend) ExtractIdentityFromContactCode(contactCode string) (string, error) {
	bundle, err := chat.FromBase64(contactCode)
	if err != nil {
		return "", err
	}

	return chat.ExtractIdentity(bundle)
}

// ExtractGroupMembershipSignatures extract signatures from tuples of content/signature
func (b *StatusBackend) ExtractGroupMembershipSignatures(signaturePairs [][2]string) ([]string, error) {
	return crypto.ExtractSignatures(signaturePairs)
}

// SignGroupMembership signs a piece of data containing membership information
func (b *StatusBackend) SignGroupMembership(content string) (string, error) {
	selectedChatAccount, err := b.AccountManager().SelectedChatAccount()
	if err != nil {
		return "", err
	}

	return crypto.Sign(content, selectedChatAccount.AccountKey.PrivateKey)
}

// EnableInstallation enables an installation for multi-device sync.
func (b *StatusBackend) EnableInstallation(installationID string) error {
	selectedChatAccount, err := b.AccountManager().SelectedChatAccount()
	if err != nil {
		return err
	}

	st, err := b.statusNode.ShhExtService()
	if err != nil {
		return err
	}

	if err := st.EnableInstallation(&selectedChatAccount.AccountKey.PrivateKey.PublicKey, installationID); err != nil {
		b.log.Error("error enabling installation", "err", err)
		return err
	}

	return nil
}

// DisableInstallation disables an installation for multi-device sync.
func (b *StatusBackend) DisableInstallation(installationID string) error {
	selectedChatAccount, err := b.AccountManager().SelectedChatAccount()
	if err != nil {
		return err
	}

	st, err := b.statusNode.ShhExtService()
	if err != nil {
		return err
	}

	if err := st.DisableInstallation(&selectedChatAccount.AccountKey.PrivateKey.PublicKey, installationID); err != nil {
		b.log.Error("error disabling installation", "err", err)
		return err
	}

	return nil
}

// UpdateMailservers on ShhExtService.
func (b *StatusBackend) UpdateMailservers(enodes []string) error {
	st, err := b.statusNode.ShhExtService()
	if err != nil {
		return err
	}
	nodes := make([]*enode.Node, len(enodes))
	for i, rawurl := range enodes {
		node, err := enode.ParseV4(rawurl)
		if err != nil {
			return err
		}
		nodes[i] = node
	}
	return st.UpdateMailservers(nodes)
}

// HashMessage calculates the hash of a message to be safely signed by the keycard
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
// This gives context to the signed message and prevents signing of transactions.
func (b *StatusBackend) HashMessage(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return ethcrypto.Keccak256([]byte(msg))
}
