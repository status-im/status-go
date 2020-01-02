package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/mailserver/registry"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	accountssvc "github.com/status-im/status-go/services/accounts"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/services/permissions"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/rpcfilters"
	"github.com/status-im/status-go/services/subscriptions"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

const (
	contractQueryTimeout = 1000 * time.Millisecond
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

var _ StatusBackend = (*GethStatusBackend)(nil)

// GethStatusBackend implements the Status.im service over go-ethereum
type GethStatusBackend struct {
	mu sync.Mutex
	// rootDataDir is the same for all networks.
	rootDataDir             string
	appDB                   *sql.DB
	statusNode              *node.StatusNode
	personalAPI             *personal.PublicAPI
	rpcFilters              *rpcfilters.Service
	multiaccountsDB         *multiaccounts.Database
	accountManager          *account.GethManager
	transactor              *transactions.Transactor
	connectionState         connectionState
	appState                appState
	selectedAccountShhKeyID string
	log                     log.Logger
	allowAllRPC             bool // used only for tests, disables api method restrictions
}

// NewGethStatusBackend create a new GethStatusBackend instance
func NewGethStatusBackend() *GethStatusBackend {
	defer log.Info("Status backend initialized", "version", params.Version, "commit", params.GitCommit)

	statusNode := node.New()
	accountManager := account.NewManager()
	transactor := transactions.NewTransactor()
	personalAPI := personal.NewAPI()
	rpcFilters := rpcfilters.New(statusNode)
	return &GethStatusBackend{
		statusNode:     statusNode,
		accountManager: accountManager,
		transactor:     transactor,
		personalAPI:    personalAPI,
		rpcFilters:     rpcFilters,
		log:            log.New("package", "status-go/api.GethStatusBackend"),
	}
}

// StatusNode returns reference to node manager
func (b *GethStatusBackend) StatusNode() *node.StatusNode {
	return b.statusNode
}

// AccountManager returns reference to account manager
func (b *GethStatusBackend) AccountManager() *account.GethManager {
	return b.accountManager
}

// Transactor returns reference to a status transactor
func (b *GethStatusBackend) Transactor() *transactions.Transactor {
	return b.transactor
}

// SelectedAccountShhKeyID returns a Whisper key ID of the selected chat key pair.
func (b *GethStatusBackend) SelectedAccountShhKeyID() string {
	return b.selectedAccountShhKeyID
}

// IsNodeRunning confirm that node is running
func (b *GethStatusBackend) IsNodeRunning() bool {
	return b.statusNode.IsRunning()
}

// StartNode start Status node, fails if node is already started
func (b *GethStatusBackend) StartNode(config *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.startNode(config); err != nil {
		signal.SendNodeCrashed(err)
		return err
	}
	return nil
}

func (b *GethStatusBackend) UpdateRootDataDir(datadir string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rootDataDir = datadir
}

func (b *GethStatusBackend) OpenAccounts() error {
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

func (b *GethStatusBackend) GetAccounts() ([]multiaccounts.Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return nil, errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.GetAccounts()
}

func (b *GethStatusBackend) SaveAccount(account multiaccounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.SaveAccount(account)
}

func (b *GethStatusBackend) ensureAppDBOpened(account multiaccounts.Account, password string) (err error) {
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
func (b *GethStatusBackend) startNodeWithKey(acc multiaccounts.Account, password string, keyHex string) error {
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
	chatKey, err := ethcrypto.HexToECDSA(keyHex)
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

func (b *GethStatusBackend) StartNodeWithKey(acc multiaccounts.Account, password string, keyHex string) error {
	err := b.startNodeWithKey(acc, password, keyHex)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	signal.SendLoggedIn(err)
	return err
}

func (b *GethStatusBackend) startNodeWithAccount(acc multiaccounts.Account, password string) error {
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

func (b *GethStatusBackend) StartNodeWithAccount(acc multiaccounts.Account, password string) error {
	err := b.startNodeWithAccount(acc, password)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	signal.SendLoggedIn(err)
	return err
}

func (b *GethStatusBackend) SaveAccountAndStartNodeWithKey(acc multiaccounts.Account, password string, settings accounts.Settings, nodecfg *params.NodeConfig, subaccs []accounts.Account, keyHex string) error {
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
func (b *GethStatusBackend) StartNodeWithAccountAndConfig(
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

func (b *GethStatusBackend) saveAccountsAndSettings(settings accounts.Settings, nodecfg *params.NodeConfig, subaccs []accounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	accdb := accounts.NewDB(b.appDB)
	err := accdb.CreateSettings(settings, *nodecfg)
	if err != nil {
		return err
	}
	return accdb.SaveAccounts(subaccs)
}

func (b *GethStatusBackend) loadNodeConfig() (*params.NodeConfig, error) {
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
	return &conf, nil
}

func (b *GethStatusBackend) rpcFiltersService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return rpcfilters.New(b.statusNode), nil
	}
}

func (b *GethStatusBackend) subscriptionService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return subscriptions.New(b.statusNode), nil
	}
}

func (b *GethStatusBackend) accountsService(accountsFeed *event.Feed) gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return accountssvc.NewService(accounts.NewDB(b.appDB), b.multiaccountsDB, b.accountManager.Manager, accountsFeed), nil
	}
}

func (b *GethStatusBackend) browsersService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return browsers.NewService(browsers.NewDB(b.appDB)), nil
	}
}

func (b *GethStatusBackend) permissionsService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return permissions.NewService(permissions.NewDB(b.appDB)), nil
	}
}

func (b *GethStatusBackend) mailserversService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return mailservers.NewService(mailservers.NewDB(b.appDB)), nil
	}
}

func (b *GethStatusBackend) walletService(network uint64, accountsFeed *event.Feed) gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return wallet.NewService(wallet.NewDB(b.appDB, network), accountsFeed), nil
	}
}

func (b *GethStatusBackend) startNode(config *params.NodeConfig) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("node crashed on start: %v", err)
		}
	}()

	// Start by validating configuration
	if err := config.Validate(); err != nil {
		return err
	}
	accountsFeed := &event.Feed{}
	services := []gethnode.ServiceConstructor{}
	services = appendIf(config.UpstreamConfig.Enabled, services, b.rpcFiltersService())
	services = append(services, b.subscriptionService())
	services = appendIf(b.appDB != nil && b.multiaccountsDB != nil, services, b.accountsService(accountsFeed))
	services = appendIf(config.BrowsersConfig.Enabled, services, b.browsersService())
	services = appendIf(config.PermissionsConfig.Enabled, services, b.permissionsService())
	services = appendIf(config.MailserversConfig.Enabled, services, b.mailserversService())
	services = appendIf(config.WalletConfig.Enabled, services, b.walletService(config.NetworkID, accountsFeed))

	manager := b.accountManager.GetManager()
	if manager == nil {
		return errors.New("ethereum accounts.Manager is nil")
	}
	if err = b.statusNode.StartWithOptions(config, node.StartOptions{
		Services: services,
		// The peers discovery protocols are started manually after
		// `node.ready` signal is sent.
		// It was discussed in https://github.com/status-im/status-go/pull/1333.
		StartDiscovery:  false,
		AccountsManager: manager,
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

	if st, err := b.statusNode.StatusService(); err == nil {
		st.SetAccountManager(b.accountManager)
	}

	if st, err := b.statusNode.PeerService(); err == nil {
		st.SetDiscoverer(b.StatusNode())
	}

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

	if err := b.statusNode.StartDiscovery(); err != nil {
		return err
	}

	return nil
}

// StopNode stop Status node. Stopped node cannot be resumed.
func (b *GethStatusBackend) StopNode() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stopNode()
}

func (b *GethStatusBackend) stopNode() error {
	if !b.IsNodeRunning() {
		return node.ErrNoRunningNode
	}
	defer signal.SendNodeStopped()
	return b.statusNode.Stop()
}

// RestartNode restart running Status node, fails if node is not running
func (b *GethStatusBackend) RestartNode() error {
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
func (b *GethStatusBackend) ResetChainData() error {
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
func (b *GethStatusBackend) CallRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// GetNodesFromContract returns a list of nodes from the contract
func (b *GethStatusBackend) GetNodesFromContract(rpcEndpoint string, contractAddress string) ([]string, error) {
	var response []string

	ctx, cancel := context.WithTimeout(context.Background(), contractQueryTimeout)
	defer cancel()

	ethclient, err := ethclient.DialContext(ctx, rpcEndpoint)
	if err != nil {
		return response, err
	}

	contract, err := registry.NewNodes(common.HexToAddress(contractAddress), ethclient)
	if err != nil {
		return response, err
	}

	nodeCount, err := contract.NodeCount(nil)
	if err != nil {
		return response, err
	}

	one := big.NewInt(1)
	for i := big.NewInt(0); i.Cmp(nodeCount) < 0; i.Add(i, one) {
		node, err := contract.Nodes(nil, i)
		if err != nil {
			return response, err
		}
		response = append(response, node)
	}

	return response, nil
}

// CallPrivateRPC executes public and private RPC requests on node's in-proc RPC server.
func (b *GethStatusBackend) CallPrivateRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCPrivateClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// SendTransaction creates a new transaction and waits until it's complete.
func (b *GethStatusBackend) SendTransaction(sendArgs transactions.SendTxArgs, password string) (hash types.Hash, err error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(sendArgs.From.String(), password)
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

func (b *GethStatusBackend) SendTransactionWithSignature(sendArgs transactions.SendTxArgs, sig []byte) (hash types.Hash, err error) {
	hash, err = b.transactor.SendTransactionWithSignature(sendArgs, sig)
	if err != nil {
		return
	}

	go b.rpcFilters.TriggerTransactionSentToUpstreamEvent(hash)

	return
}

// HashTransaction validate the transaction and returns new sendArgs and the transaction hash.
func (b *GethStatusBackend) HashTransaction(sendArgs transactions.SendTxArgs) (transactions.SendTxArgs, types.Hash, error) {
	return b.transactor.HashTransaction(sendArgs)
}

// SignMessage checks the pwd vs the selected account and passes on the signParams
// to personalAPI for message signature
func (b *GethStatusBackend) SignMessage(rpcParams personal.SignParams) (types.HexBytes, error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(rpcParams.Address, rpcParams.Password)
	if err != nil {
		return types.HexBytes{}, err
	}
	return b.personalAPI.Sign(rpcParams, verifiedAccount)
}

// Recover calls the personalAPI to return address associated with the private
// key that was used to calculate the signature in the message
func (b *GethStatusBackend) Recover(rpcParams personal.RecoverParams) (types.Address, error) {
	return b.personalAPI.Recover(rpcParams)
}

// SignTypedData accepts data and password. Gets verified account and signs typed data.
func (b *GethStatusBackend) SignTypedData(typed typeddata.TypedData, address string, password string) (types.HexBytes, error) {
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
func (b *GethStatusBackend) HashTypedData(typed typeddata.TypedData) (types.Hash, error) {
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	hash, err := typeddata.ValidateAndHash(typed, chain)
	if err != nil {
		return types.Hash{}, err
	}
	return types.Hash(hash), err
}

func (b *GethStatusBackend) getVerifiedWalletAccount(address, password string) (*account.SelectedExtKey, error) {
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
func (b *GethStatusBackend) registerHandlers() error {
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
func (b *GethStatusBackend) ConnectionChange(typ string, expensive bool) {
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
func (b *GethStatusBackend) AppStateChange(state string) {
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
func (b *GethStatusBackend) Logout() error {
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

	b.AccountManager().Logout()

	return nil
}

// cleanupServices stops parts of services that doesn't managed by a node and removes injected data from services.
func (b *GethStatusBackend) cleanupServices() error {
	whisperService, err := b.statusNode.WhisperService()
	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.DeleteKeyPairs(); err != nil {
			return fmt.Errorf("%s: %v", ErrWhisperClearIdentitiesFailure, err)
		}
		b.selectedAccountShhKeyID = ""
	default:
		return err
	}
	if b.statusNode.Config().WalletConfig.Enabled {
		wallet, err := b.statusNode.WalletService()
		switch err {
		case node.ErrServiceUnknown:
		case nil:
			err = wallet.StopReactor()
			if err != nil {
				return err
			}
		default:
			return err
		}
	}
	return nil
}

func (b *GethStatusBackend) closeAppDB() error {
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
func (b *GethStatusBackend) SelectAccount(loginParams account.LoginParams) error {
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

	if err := b.startWallet(); err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) injectAccountIntoServices() error {
	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}

	identity := chatAccount.AccountKey.PrivateKey
	whisperService, err := b.statusNode.WhisperService()

	switch err {
	case node.ErrServiceUnknown: // Whisper was never registered
	case nil:
		if err := whisperService.DeleteKeyPairs(); err != nil { // err is not possible; method return value is incorrect
			return err
		}
		b.selectedAccountShhKeyID, err = whisperService.AddKeyPair(identity)
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

func (b *GethStatusBackend) startWallet() error {
	if !b.statusNode.Config().WalletConfig.Enabled {
		return nil
	}

	wallet, err := b.statusNode.WalletService()
	if err != nil {
		return err
	}

	watchAddresses := b.accountManager.WatchAddresses()
	mainAccountAddress, err := b.accountManager.MainAccountAddress()
	if err != nil {
		return err
	}

	allAddresses := make([]common.Address, len(watchAddresses)+1)
	allAddresses[0] = common.Address(mainAccountAddress)
	for i, addr := range watchAddresses {
		allAddresses[1+i] = common.Address(addr)
	}
	return wallet.StartReactor(
		b.statusNode.RPCClient().Ethclient(),
		allAddresses,
		new(big.Int).SetUint64(b.statusNode.Config().NetworkID),
	)
}

// InjectChatAccount selects the current chat account using chatKeyHex and injects the key into whisper.
// TODO: change the interface and omit the last argument.
func (b *GethStatusBackend) InjectChatAccount(chatKeyHex, _ string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.accountManager.Logout()

	chatKey, err := ethcrypto.HexToECDSA(chatKeyHex)
	if err != nil {
		return err
	}
	b.accountManager.SetChatAccount(chatKey)

	return b.injectAccountIntoServices()
}

func appendIf(condition bool, services []gethnode.ServiceConstructor, service gethnode.ServiceConstructor) []gethnode.ServiceConstructor {
	if !condition {
		return services
	}
	return append(services, service)
}

// ExtractGroupMembershipSignatures extract signatures from tuples of content/signature
func (b *GethStatusBackend) ExtractGroupMembershipSignatures(signaturePairs [][2]string) ([]string, error) {
	return crypto.ExtractSignatures(signaturePairs)
}

// SignGroupMembership signs a piece of data containing membership information
func (b *GethStatusBackend) SignGroupMembership(content string) (string, error) {
	selectedChatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return "", err
	}

	return crypto.SignStringAsHex(content, selectedChatAccount.AccountKey.PrivateKey)
}

// EnableInstallation enables an installation for multi-device sync.
func (b *GethStatusBackend) EnableInstallation(installationID string) error {
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
func (b *GethStatusBackend) DisableInstallation(installationID string) error {
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

// UpdateMailservers on ShhExtService.
func (b *GethStatusBackend) UpdateMailservers(enodes []string) error {
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

// SignHash exposes vanilla ECDSA signing for signing a message for Swarm
func (b *GethStatusBackend) SignHash(hexEncodedHash string) (string, error) {
	hash, err := hexutil.Decode(hexEncodedHash)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not unmarshal the input: %v", err)
	}

	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return "", fmt.Errorf("SignHash: could not select account: %v", err.Error())
	}

	signature, err := ethcrypto.Sign(hash, chatAccount.AccountKey.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not sign the hash: %v", err)
	}

	hexEncodedSignature := types.EncodeHex(signature)
	return hexEncodedSignature, nil
}
