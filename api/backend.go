package api

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	gethnode "github.com/ethereum/go-ethereum/node"

	fcmlib "github.com/NaySoftware/go-fcm"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/notifications/push/fcm"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/rpcfilters"
	"github.com/status-im/status-go/sign"
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
	defer log.Info("Status backend initialized")

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

	services := []gethnode.ServiceConstructor{}
	services = appendIf(config.UpstreamConfig.Enabled, services, b.rpcFiltersService())

	if err = b.statusNode.Start(config, services...); err != nil {
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
func (b *StatusBackend) CallRPC(inputJSON string) string {
	client := b.statusNode.RPCClient()
	return client.CallRaw(inputJSON)
}

// CallPrivateRPC executes public and private RPC requests on node's in-proc RPC server.
func (b *StatusBackend) CallPrivateRPC(inputJSON string) string {
	client := b.statusNode.RPCPrivateClient()
	return client.CallRaw(inputJSON)
}

// SendTransaction creates a new transaction and waits until it's complete.
func (b *StatusBackend) SendTransaction(sendArgs transactions.SendTxArgs, password string) sign.Result {
	verifiedAccount, err := b.getVerifiedAccount(password)
	if err != nil {
		return sign.NewErrResult(err)
	}
	result := b.transactor.SendTransaction(sendArgs, verifiedAccount)
	if result.Error != nil {
		return result
	}
	go b.rpcFilters.TriggerTransactionSentToUpstreamEvent(result.Response.Hash())
	return result
}

// SignMessage checks the pwd vs the selected account and passes on the signParams
// to personalAPI for message signature
func (b *StatusBackend) SignMessage(rpcParams personal.SignParams, password string) sign.Result {
	verifiedAccount, err := b.getVerifiedAccount(password)
	if err != nil {
		return sign.NewErrResult(err)
	}
	return b.personalAPI.Sign(rpcParams, verifiedAccount)
}

// Recover calls the personalAPI to return address associated with the private
// key that was used to calculate the signature in the message
func (b *StatusBackend) Recover(rpcParams personal.RecoverParams) sign.Result {
	return b.personalAPI.Recover(rpcParams)
}

func (b *StatusBackend) getVerifiedAccount(password string) (*account.SelectedExtKey, error) {
	selectedAccount, err := b.accountManager.SelectedAccount()
	if err != nil {
		b.log.Error("failed to get a selected account", "err", err)
		return nil, err
	}
	config := b.StatusNode().Config()
	_, err = b.accountManager.VerifyAccountPassword(config.KeyStoreDir, selectedAccount.Address.String(), password)
	if err != nil {
		b.log.Error("failed to verify account", "account", selectedAccount.Address.String(), "error", err)
		return nil, err
	}
	return selectedAccount, nil
}

// registerHandlers attaches Status callback handlers to running node
func (b *StatusBackend) registerHandlers() error {
	rpcClient := b.StatusNode().RPCClient()
	if rpcClient == nil {
		return errors.New("RPC client unavailable")
	}

	rpcClient.RegisterHandler(params.AccountsMethodName, func(context.Context, ...interface{}) (interface{}, error) {
		return b.AccountManager().Accounts()
	})

	rpcClient.RegisterHandler(params.SendTransactionMethodName, unsupportedMethodHandler)

	rpcClient.RegisterHandler(params.PersonalSignMethodName, unsupportedMethodHandler)
	rpcClient.RegisterHandler(params.PersonalRecoverMethodName, unsupportedMethodHandler)

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
	selectedAccount, err := b.AccountManager().SelectedAccount()
	if selectedAccount == nil || err == account.ErrNoAccountSelected {
		return nil
	}
	whisperService, err := b.statusNode.WhisperService()
	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.SelectKeyPair(selectedAccount.AccountKey.PrivateKey); err != nil {
			return ErrWhisperIdentityInjectionFailure
		}
	default:
		return err
	}

	return nil
}

// SelectAccount selects current account, by verifying that address has corresponding account which can be decrypted
// using provided password. Once verification is done, decrypted key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (b *StatusBackend) SelectAccount(address, password string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := b.accountManager.SelectAccount(address, password)
	if err != nil {
		return err
	}
	acc, err := b.accountManager.SelectedAccount()
	if err != nil {
		return err
	}

	whisperService, err := b.statusNode.WhisperService()
	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.SelectKeyPair(acc.AccountKey.PrivateKey); err != nil {
			return ErrWhisperIdentityInjectionFailure
		}
	default:
		return err
	}

	return nil
}

// NotifyUsers sends push notifications to users.
func (b *StatusBackend) NotifyUsers(message string, payload fcmlib.NotificationPayload, tokens ...string) error {
	err := b.newNotification().Send(message, payload, tokens...)
	if err != nil {
		b.log.Error("Notify failed", "error", err)
	}

	return err
}

func appendIf(condition bool, services []gethnode.ServiceConstructor, service gethnode.ServiceConstructor) []gethnode.ServiceConstructor {
	if !condition {
		return services
	}
	return append(services, service)
}
