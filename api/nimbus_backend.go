// +build nimbus

package api

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/event"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	accountssvc "github.com/status-im/status-go/services/accounts"
	nimbussvc "github.com/status-im/status-go/services/nimbus"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/rpcfilters"
	"github.com/status-im/status-go/services/subscriptions"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

// const (
// 	contractQueryTimeout = 1000 * time.Millisecond
// )

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

var _ StatusBackend = (*nimbusStatusBackend)(nil)

// nimbusStatusBackend implements the Status.im service over Nimbus
type nimbusStatusBackend struct {
	StatusBackend

	mu sync.Mutex
	// rootDataDir is the same for all networks.
	rootDataDir string
	appDB       *sql.DB
	statusNode  *node.NimbusStatusNode
	// personalAPI     *personal.PublicAPI
	// rpcFilters      *rpcfilters.Service
	multiaccountsDB *multiaccounts.Database
	accountManager  *account.GethManager
	// transactor      *transactions.Transactor
	connectionState      connectionState
	appState             appState
	selectedAccountKeyID string
	log                  log.Logger
	allowAllRPC          bool // used only for tests, disables api method restrictions
}

// NewNimbusStatusBackend create a new nimbusStatusBackend instance
func NewNimbusStatusBackend() *nimbusStatusBackend {
	defer log.Info("Status backend initialized", "backend", "nimbus", "version", params.Version, "commit", params.GitCommit)

	statusNode := node.NewNimbus()
	accountManager := account.NewGethManager()
	// transactor := transactions.NewTransactor()
	// personalAPI := personal.NewAPI()
	// rpcFilters := rpcfilters.New(statusNode)
	return &nimbusStatusBackend{
		statusNode:     statusNode,
		accountManager: accountManager,
		// transactor:     transactor,
		// personalAPI:    personalAPI,
		// rpcFilters:     rpcFilters,
		log: log.New("package", "status-go/api.nimbusStatusBackend"),
	}
}

// StatusNode returns reference to node manager
func (b *nimbusStatusBackend) StatusNode() *node.NimbusStatusNode {
	return b.statusNode
}

// AccountManager returns reference to account manager
func (b *nimbusStatusBackend) AccountManager() *account.GethManager {
	return b.accountManager
}

// // Transactor returns reference to a status transactor
// func (b *nimbusStatusBackend) Transactor() *transactions.Transactor {
// 	return b.transactor
// }

// SelectedAccountKeyID returns a Whisper key ID of the selected chat key pair.
func (b *nimbusStatusBackend) SelectedAccountKeyID() string {
	return b.selectedAccountKeyID
}

// IsNodeRunning confirm that node is running
func (b *nimbusStatusBackend) IsNodeRunning() bool {
	return b.statusNode.IsRunning()
}

// StartNode start Status node, fails if node is already started
func (b *nimbusStatusBackend) StartNode(config *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.startNode(config); err != nil {
		signal.SendNodeCrashed(err)
		return err
	}
	return nil
}

func (b *nimbusStatusBackend) UpdateRootDataDir(datadir string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rootDataDir = datadir
}

func (b *nimbusStatusBackend) OpenAccounts() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB != nil {
		return nil
	}
	db, err := multiaccounts.InitializeDB(filepath.Join(b.rootDataDir, "accounts.sql"))
	if err != nil {
		return err
	}
	b.multiaccountsDB = db
	return nil
}

func (b *nimbusStatusBackend) GetAccounts() ([]multiaccounts.Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return nil, errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.GetAccounts()
}

func (b *nimbusStatusBackend) SaveAccount(account multiaccounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.SaveAccount(account)
}

func (b *nimbusStatusBackend) ensureAppDBOpened(account multiaccounts.Account, password string) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}
	if len(b.rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}
	path := filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql", account.KeyUID))
	b.appDB, err = appdatabase.InitializeDB(path, password)
	if err != nil {
		return err
	}
	return nil
}

// StartNodeWithKey instead of loading addresses from database this method derives address from key
// and uses it in application.
// TODO: we should use a proper struct with optional values instead of duplicating the regular functions
// with small variants for keycard, this created too many bugs
func (b *nimbusStatusBackend) startNodeWithKey(acc multiaccounts.Account, password string, keyHex string) error {
	err := b.ensureAppDBOpened(acc, password)
	if err != nil {
		return err
	}
	conf, err := b.loadNodeConfig()
	if err != nil {
		return err
	}
	if err := logutils.OverrideRootLogWithConfig(conf, false); err != nil {
		return err
	}
	accountsDB := accounts.NewDB(b.appDB)
	walletAddr, err := accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}
	watchAddrs, err := accountsDB.GetAddresses()
	if err != nil {
		return err
	}
	chatKey, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return err
	}
	err = b.StartNode(conf)
	if err != nil {
		return err
	}
	b.accountManager.SetChatAccount(chatKey)
	_, err = b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}
	b.accountManager.SetAccountAddresses(walletAddr, watchAddrs...)
	err = b.injectAccountIntoServices()
	if err != nil {
		return err
	}
	err = b.multiaccountsDB.UpdateAccountTimestamp(acc.KeyUID, time.Now().Unix())
	if err != nil {
		return err
	}
	return nil
}

func (b *nimbusStatusBackend) StartNodeWithKey(acc multiaccounts.Account, password string, keyHex string) error {
	err := b.startNodeWithKey(acc, password, keyHex)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	signal.SendLoggedIn(err)
	return err
}

func (b *nimbusStatusBackend) startNodeWithAccount(acc multiaccounts.Account, password string) error {
	err := b.ensureAppDBOpened(acc, password)
	if err != nil {
		return err
	}
	conf, err := b.loadNodeConfig()
	if err != nil {
		return err
	}
	if err := logutils.OverrideRootLogWithConfig(conf, false); err != nil {
		return err
	}
	accountsDB := accounts.NewDB(b.appDB)
	chatAddr, err := accountsDB.GetChatAddress()
	if err != nil {
		return err
	}
	walletAddr, err := accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}
	watchAddrs, err := accountsDB.GetAddresses()
	if err != nil {
		return err
	}
	login := account.LoginParams{
		Password:       password,
		ChatAddress:    chatAddr,
		WatchAddresses: watchAddrs,
		MainAccount:    walletAddr,
	}
	err = b.StartNode(conf)
	if err != nil {
		return err
	}
	err = b.SelectAccount(login)
	if err != nil {
		return err
	}
	err = b.multiaccountsDB.UpdateAccountTimestamp(acc.KeyUID, time.Now().Unix())
	if err != nil {
		return err
	}
	return nil
}

func (b *nimbusStatusBackend) StartNodeWithAccount(acc multiaccounts.Account, password string) error {
	err := b.startNodeWithAccount(acc, password)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	signal.SendLoggedIn(err)
	return err
}

func (b *nimbusStatusBackend) SaveAccountAndStartNodeWithKey(acc multiaccounts.Account, password string, settings accounts.Settings, nodecfg *params.NodeConfig, subaccs []accounts.Account, keyHex string) error {
	err := b.SaveAccount(acc)
	if err != nil {
		return err
	}
	err = b.ensureAppDBOpened(acc, password)
	if err != nil {
		return err
	}
	err = b.saveAccountsAndSettings(settings, nodecfg, subaccs)
	if err != nil {
		return err
	}
	return b.StartNodeWithKey(acc, password, keyHex)
}

// StartNodeWithAccountAndConfig is used after account and config was generated.
// In current setup account name and config is generated on the client side. Once/if it will be generated on
// status-go side this flow can be simplified.
func (b *nimbusStatusBackend) StartNodeWithAccountAndConfig(
	account multiaccounts.Account,
	password string,
	settings accounts.Settings,
	nodecfg *params.NodeConfig,
	subaccs []accounts.Account,
) error {
	err := b.SaveAccount(account)
	if err != nil {
		return err
	}
	err = b.ensureAppDBOpened(account, password)
	if err != nil {
		return err
	}
	err = b.saveAccountsAndSettings(settings, nodecfg, subaccs)
	if err != nil {
		return err
	}
	return b.StartNodeWithAccount(account, password)
}

func (b *nimbusStatusBackend) saveAccountsAndSettings(settings accounts.Settings, nodecfg *params.NodeConfig, subaccs []accounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	accdb := accounts.NewDB(b.appDB)
	err := accdb.CreateSettings(settings, *nodecfg)
	if err != nil {
		return err
	}
	return accdb.SaveAccounts(subaccs)
}

func (b *nimbusStatusBackend) loadNodeConfig() (*params.NodeConfig, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var conf params.NodeConfig
	err := accounts.NewDB(b.appDB).GetNodeConfig(&conf)
	if err != nil {
		return nil, err
	}
	// NodeConfig.Version should be taken from params.Version
	// which is set at the compile time.
	// What's cached is usually outdated so we overwrite it here.
	conf.Version = params.Version

	// Replace all relative paths with absolute
	conf.DataDir = filepath.Join(b.rootDataDir, conf.DataDir)
	conf.ShhextConfig.BackupDisabledDataDir = filepath.Join(b.rootDataDir, conf.ShhextConfig.BackupDisabledDataDir)
	if len(conf.LogDir) == 0 {
		conf.LogFile = filepath.Join(b.rootDataDir, conf.LogFile)
	} else {
		conf.LogFile = filepath.Join(conf.LogDir, conf.LogFile)
	}
	conf.KeyStoreDir = filepath.Join(b.rootDataDir, conf.KeyStoreDir)

	return &conf, nil
}

func (b *nimbusStatusBackend) rpcFiltersService() nimbussvc.ServiceConstructor {
	return func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
		return rpcfilters.New(b.statusNode), nil
	}
}

func (b *nimbusStatusBackend) subscriptionService() nimbussvc.ServiceConstructor {
	return func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
		return subscriptions.New(func() *rpc.Client { return b.statusNode.RPCPrivateClient() }), nil
	}
}

func (b *nimbusStatusBackend) accountsService(accountsFeed *event.Feed) nimbussvc.ServiceConstructor {
	return func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
		return accountssvc.NewService(accounts.NewDB(b.appDB), b.multiaccountsDB, b.accountManager.Manager, accountsFeed), nil
	}
}

// func (b *nimbusStatusBackend) browsersService() nimbussvc.ServiceConstructor {
// 	return func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
// 		return browsers.NewService(browsers.NewDB(b.appDB)), nil
// 	}
// }

// func (b *nimbusStatusBackend) permissionsService() nimbussvc.ServiceConstructor {
// 	return func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
// 		return permissions.NewService(permissions.NewDB(b.appDB)), nil
// 	}
// }

// func (b *nimbusStatusBackend) mailserversService() nimbussvc.ServiceConstructor {
// 	return func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
// 		return mailservers.NewService(mailservers.NewDB(b.appDB)), nil
// 	}
// }

// func (b *nimbusStatusBackend) walletService(network uint64, accountsFeed *event.Feed) nimbussvc.ServiceConstructor {
// 	return func(*nimbussvc.ServiceContext) (nimbussvc.Service, error) {
// 		return wallet.NewService(wallet.NewDB(b.appDB, network), accountsFeed), nil
// 	}
// }

func (b *nimbusStatusBackend) startNode(config *params.NodeConfig) (err error) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		err = fmt.Errorf("node crashed on start: %v", err)
	// 	}
	// }()

	// Start by validating configuration
	if err := config.Validate(); err != nil {
		return err
	}

	accountsFeed := &event.Feed{}
	services := []nimbussvc.ServiceConstructor{}
	services = appendIf(config.UpstreamConfig.Enabled, services, b.rpcFiltersService())
	services = append(services, b.subscriptionService())
	services = appendIf(b.appDB != nil && b.multiaccountsDB != nil, services, b.accountsService(accountsFeed))
	// services = appendIf(config.BrowsersConfig.Enabled, services, b.browsersService())
	// services = appendIf(config.PermissionsConfig.Enabled, services, b.permissionsService())
	// services = appendIf(config.MailserversConfig.Enabled, services, b.mailserversService())
	// services = appendIf(config.WalletConfig.Enabled, services, b.walletService(config.NetworkID, accountsFeed))

	// manager := b.accountManager.GetManager()
	// if manager == nil {
	// 	return errors.New("ethereum accounts.Manager is nil")
	// }
	if err = b.statusNode.StartWithOptions(config, node.NimbusStartOptions{
		Services: services,
		// The peers discovery protocols are started manually after
		// `node.ready` signal is sent.
		// It was discussed in https://github.com/status-im/status-go/pull/1333.
		StartDiscovery: false,
		// AccountsManager: manager,
	}); err != nil {
		return
	}
	signal.SendNodeStarted()

	// b.transactor.SetNetworkID(config.NetworkID)
	// b.transactor.SetRPC(b.statusNode.RPCClient(), rpc.DefaultCallTimeout)
	// b.personalAPI.SetRPC(b.statusNode.RPCPrivateClient(), rpc.DefaultCallTimeout)

	if err = b.registerHandlers(); err != nil {
		b.log.Error("Handler registration failed", "err", err)
		return
	}
	b.log.Info("Handlers registered")

	if st, err := b.statusNode.StatusService(); err == nil {
		st.SetAccountManager(b.accountManager)
	}

	// if st, err := b.statusNode.PeerService(); err == nil {
	// 	st.SetDiscoverer(b.StatusNode())
	// }

	// Handle a case when a node is stopped and resumed.
	// If there is no account selected, an error is returned.
	if _, err := b.accountManager.SelectedChatAccount(); err == nil {
		if err := b.injectAccountIntoServices(); err != nil {
			return err
		}
	} else if err != account.ErrNoAccountSelected {
		return err
	}

	signal.SendNodeReady()

	// if err := b.statusNode.StartDiscovery(); err != nil {
	// 	return err
	// }

	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (b *nimbusStatusBackend) StopNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stopNode()
}

func (b *nimbusStatusBackend) stopNode() error {
	if !b.IsNodeRunning() {
		return node.ErrNoRunningNode
	}
	defer signal.SendNodeStopped()
	return b.statusNode.Stop()
}

// RestartNode restart running Status node, fails if node is not running
func (b *nimbusStatusBackend) RestartNode() error {
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
func (b *nimbusStatusBackend) ResetChainData() error {
	panic("ResetChainData")
	// b.mu.Lock()
	// defer b.mu.Unlock()
	// newcfg := *(b.statusNode.Config())
	// if err := b.stopNode(); err != nil {
	// 	return err
	// }
	// // config is cleaned when node is stopped
	// if err := b.statusNode.ResetChainData(&newcfg); err != nil {
	// 	return err
	// }
	// signal.SendChainDataRemoved()
	// return b.startNode(&newcfg)
}

// CallRPC executes public RPC requests on node's in-proc RPC server.
func (b *nimbusStatusBackend) CallRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// GetNodesFromContract returns a list of nodes from the contract
func (b *nimbusStatusBackend) GetNodesFromContract(rpcEndpoint string, contractAddress string) ([]string, error) {
	panic("GetNodesFromContract")
	// var response []string

	// ctx, cancel := context.WithTimeout(context.Background(), contractQueryTimeout)
	// defer cancel()

	// ethclient, err := ethclient.DialContext(ctx, rpcEndpoint)
	// if err != nil {
	// 	return response, err
	// }

	// contract, err := registry.NewNodes(types.HexToAddress(contractAddress), ethclient)
	// if err != nil {
	// 	return response, err
	// }

	// nodeCount, err := contract.NodeCount(nil)
	// if err != nil {
	// 	return response, err
	// }

	// one := big.NewInt(1)
	// for i := big.NewInt(0); i.Cmp(nodeCount) < 0; i.Add(i, one) {
	// 	node, err := contract.Nodes(nil, i)
	// 	if err != nil {
	// 		return response, err
	// 	}
	// 	response = append(response, node)
	// }

	// return response, nil
}

// CallPrivateRPC executes public and private RPC requests on node's in-proc RPC server.
func (b *nimbusStatusBackend) CallPrivateRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCPrivateClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// SendTransaction creates a new transaction and waits until it's complete.
func (b *nimbusStatusBackend) SendTransaction(sendArgs transactions.SendTxArgs, password string) (hash types.Hash, err error) {
	panic("SendTransaction")
	// verifiedAccount, err := b.getVerifiedWalletAccount(sendArgs.From.String(), password)
	// if err != nil {
	// 	return hash, err
	// }

	// hash, err = b.transactor.SendTransaction(sendArgs, verifiedAccount)
	// if err != nil {
	// 	return
	// }

	// go b.rpcFilters.TriggerTransactionSentToUpstreamEvent(hash)

	// return
}

func (b *nimbusStatusBackend) SendTransactionWithSignature(sendArgs transactions.SendTxArgs, sig []byte) (hash types.Hash, err error) {
	panic("SendTransactionWithSignature")
	// hash, err = b.transactor.SendTransactionWithSignature(sendArgs, sig)
	// if err != nil {
	// 	return
	// }

	// go b.rpcFilters.TriggerTransactionSentToUpstreamEvent(hash)

	// return
}

// HashTransaction validate the transaction and returns new sendArgs and the transaction hash.
func (b *nimbusStatusBackend) HashTransaction(sendArgs transactions.SendTxArgs) (transactions.SendTxArgs, types.Hash, error) {
	panic("HashTransaction")
	// return b.transactor.HashTransaction(sendArgs)
}

// SignMessage checks the pwd vs the selected account and passes on the signParams
// to personalAPI for message signature
func (b *nimbusStatusBackend) SignMessage(rpcParams personal.SignParams) (types.HexBytes, error) {
	panic("SignMessage")
	// verifiedAccount, err := b.getVerifiedWalletAccount(rpcParams.Address, rpcParams.Password)
	// if err != nil {
	// 	return types.Bytes{}, err
	// }
	// return b.personalAPI.Sign(rpcParams, verifiedAccount)
}

// Recover calls the personalAPI to return address associated with the private
// key that was used to calculate the signature in the message
func (b *nimbusStatusBackend) Recover(rpcParams personal.RecoverParams) (types.Address, error) {
	panic("Recover")
	// return b.personalAPI.Recover(rpcParams)
}

// SignTypedData accepts data and password. Gets verified account and signs typed data.
func (b *nimbusStatusBackend) SignTypedData(typed typeddata.TypedData, address string, password string) (types.HexBytes, error) {
	account, err := b.getVerifiedWalletAccount(address, password)
	if err != nil {
		return types.HexBytes{}, err
	}
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	sig, err := typeddata.Sign(typed, account.AccountKey.PrivateKey, chain)
	if err != nil {
		return types.HexBytes{}, err
	}
	return types.HexBytes(sig), err
}

// HashTypedData generates the hash of TypedData.
func (b *nimbusStatusBackend) HashTypedData(typed typeddata.TypedData) (types.Hash, error) {
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	hash, err := typeddata.ValidateAndHash(typed, chain)
	if err != nil {
		return types.Hash{}, err
	}
	return types.Hash(hash), err
}

func (b *nimbusStatusBackend) getVerifiedWalletAccount(address, password string) (*account.SelectedExtKey, error) {
	config := b.StatusNode().Config()

	db := accounts.NewDB(b.appDB)
	exists, err := db.AddressExists(types.HexToAddress(address))
	if err != nil {
		b.log.Error("failed to query db for a given address", "address", address, "error", err)
		return nil, err
	}

	if !exists {
		b.log.Error("failed to get a selected account", "err", transactions.ErrInvalidTxSender)
		return nil, transactions.ErrAccountDoesntExist
	}

	key, err := b.accountManager.VerifyAccountPassword(config.KeyStoreDir, address, password)
	if err != nil {
		b.log.Error("failed to verify account", "account", address, "error", err)
		return nil, err
	}

	return &account.SelectedExtKey{
		Address:    key.Address,
		AccountKey: key,
	}, nil
}

// registerHandlers attaches Status callback handlers to running node
func (b *nimbusStatusBackend) registerHandlers() error {
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
				return b.accountManager.Accounts()
			},
		)

		if b.allowAllRPC {
			// this should only happen in unit-tests, this variable is not available outside this package
			continue
		}
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
func (b *nimbusStatusBackend) ConnectionChange(typ string, expensive bool) {
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
func (b *nimbusStatusBackend) AppStateChange(state string) {
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
func (b *nimbusStatusBackend) Logout() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := b.cleanupServices()
	if err != nil {
		return err
	}
	err = b.closeAppDB()
	if err != nil {
		return err
	}

	b.accountManager.Logout()

	return nil
}

// cleanupServices stops parts of services that doesn't managed by a node and removes injected data from services.
func (b *nimbusStatusBackend) cleanupServices() error {
	whisperService, err := b.statusNode.WhisperService()
	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.Whisper.DeleteKeyPairs(); err != nil {
			return fmt.Errorf("%s: %v", ErrWhisperClearIdentitiesFailure, err)
		}
		b.selectedAccountKeyID = ""
	default:
		return err
	}
	// if b.statusNode.Config().WalletConfig.Enabled {
	// 	wallet, err := b.statusNode.WalletService()
	// 	switch err {
	// 	case node.ErrServiceUnknown:
	// 	case nil:
	// 		err = wallet.StopReactor()
	// 		if err != nil {
	// 			return err
	// 		}
	// 	default:
	// 		return err
	// 	}
	// }
	return nil
}

func (b *nimbusStatusBackend) closeAppDB() error {
	if b.appDB != nil {
		err := b.appDB.Close()
		if err != nil {
			return err
		}
		b.appDB = nil
		return nil
	}
	return nil
}

// SelectAccount selects current wallet and chat accounts, by verifying that each address has corresponding account which can be decrypted
// using provided password. Once verification is done, the decrypted chat key is injected into Whisper (as a single identity,
// all previous identities are removed).
func (b *nimbusStatusBackend) SelectAccount(loginParams account.LoginParams) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.AccountManager().RemoveOnboarding()

	err := b.accountManager.SelectAccount(loginParams)
	if err != nil {
		return err
	}

	if err := b.injectAccountIntoServices(); err != nil {
		return err
	}

	// if err := b.startWallet(); err != nil {
	// 	return err
	// }

	return nil
}

func (b *nimbusStatusBackend) injectAccountIntoServices() error {
	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}

	identity := chatAccount.AccountKey.PrivateKey
	whisperService, err := b.statusNode.WhisperService()

	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.Whisper.DeleteKeyPairs(); err != nil { // err is not possible; method return value is incorrect
			return err
		}
		b.selectedAccountKeyID, err = whisperService.Whisper.AddKeyPair(identity)
		if err != nil {
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

		if err := st.InitProtocol(identity, b.appDB); err != nil {
			return err
		}
	}
	return nil
}

// func (b *nimbusStatusBackend) startWallet() error {
// 	if !b.statusNode.Config().WalletConfig.Enabled {
// 		return nil
// 	}

// 	wallet, err := b.statusNode.WalletService()
// 	if err != nil {
// 		return err
// 	}

// 	watchAddresses := b.accountManager.WatchAddresses()
// 	mainAccountAddress, err := b.accountManager.MainAccountAddress()
// 	if err != nil {
// 		return err
// 	}

// 	allAddresses := make([]types.Address, len(watchAddresses)+1)
// 	allAddresses[0] = mainAccountAddress
// 	copy(allAddresses[1:], watchAddresses)
// 	return wallet.StartReactor(
// 		b.statusNode.RPCClient().Ethclient(),
// 		allAddresses,
// 		new(big.Int).SetUint64(b.statusNode.Config().NetworkID),
//	)
// }

// InjectChatAccount selects the current chat account using chatKeyHex and injects the key into whisper.
// TODO: change the interface and omit the last argument.
func (b *nimbusStatusBackend) InjectChatAccount(chatKeyHex, _ string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.accountManager.Logout()

	chatKey, err := crypto.HexToECDSA(chatKeyHex)
	if err != nil {
		return err
	}
	b.accountManager.SetChatAccount(chatKey)

	return b.injectAccountIntoServices()
}

func appendIf(condition bool, services []nimbussvc.ServiceConstructor, service nimbussvc.ServiceConstructor) []nimbussvc.ServiceConstructor {
	if !condition {
		return services
	}
	return append(services, service)
}

// ExtractGroupMembershipSignatures extract signatures from tuples of content/signature
func (b *nimbusStatusBackend) ExtractGroupMembershipSignatures(signaturePairs [][2]string) ([]string, error) {
	return crypto.ExtractSignatures(signaturePairs)
}

// SignGroupMembership signs a piece of data containing membership information
func (b *nimbusStatusBackend) SignGroupMembership(content string) (string, error) {
	selectedChatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return "", err
	}

	return crypto.SignStringAsHex(content, selectedChatAccount.AccountKey.PrivateKey)
}

// EnableInstallation enables an installation for multi-device sync.
func (b *nimbusStatusBackend) EnableInstallation(installationID string) error {
	st, err := b.statusNode.ShhExtService()
	if err != nil {
		return err
	}

	if err := st.EnableInstallation(installationID); err != nil {
		b.log.Error("error enabling installation", "err", err)
		return err
	}

	return nil
}

// DisableInstallation disables an installation for multi-device sync.
func (b *nimbusStatusBackend) DisableInstallation(installationID string) error {
	st, err := b.statusNode.ShhExtService()
	if err != nil {
		return err
	}

	if err := st.DisableInstallation(installationID); err != nil {
		b.log.Error("error disabling installation", "err", err)
		return err
	}

	return nil
}

// SignHash exposes vanilla ECDSA signing for signing a message for Swarm
func (b *nimbusStatusBackend) SignHash(hexEncodedHash string) (string, error) {
	hash, err := types.DecodeHex(hexEncodedHash)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not unmarshal the input: %v", err)
	}

	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return "", fmt.Errorf("SignHash: could not select account: %v", err.Error())
	}

	signature, err := crypto.Sign(hash, chatAccount.AccountKey.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not sign the hash: %v", err)
	}

	hexEncodedSignature := types.EncodeHex(signature)
	return hexEncodedSignature, nil
}
