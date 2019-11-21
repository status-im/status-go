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
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/crypto"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/mailserver/registry"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	protocol "github.com/status-im/status-go/protocol/types"
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

// StatusBackend implements Status.im service
type StatusBackend struct {
	mu sync.Mutex
	// rootDataDir is the same for all networks.
	rootDataDir     string
	appDB           *sql.DB
	statusNode      *node.StatusNode
	personalAPI     *personal.PublicAPI
	rpcFilters      *rpcfilters.Service
	multiaccountsDB *multiaccounts.Database
	accountManager  *account.Manager
	transactor      *transactions.Transactor
	connectionState connectionState
	appState        appState
	log             log.Logger
	allowAllRPC     bool // used only for tests, disables api method restrictions
}

// NewStatusBackend create a new StatusBackend instance
func NewStatusBackend() *StatusBackend {
	defer log.Info("Status backend initialized", "version", params.Version, "commit", params.GitCommit)

	statusNode := node.New()
	accountManager := account.NewManager()
	transactor := transactions.NewTransactor()
	personalAPI := personal.NewAPI()
	rpcFilters := rpcfilters.New(statusNode)
	return &StatusBackend{
		statusNode:     statusNode,
		accountManager: accountManager,
		transactor:     transactor,
		personalAPI:    personalAPI,
		rpcFilters:     rpcFilters,
		log:            log.New("package", "status-go/api.StatusBackend"),
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

func (b *StatusBackend) UpdateRootDataDir(datadir string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rootDataDir = datadir
}

func (b *StatusBackend) OpenAccounts() error {
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

func (b *StatusBackend) GetAccounts() ([]multiaccounts.Account, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return nil, errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.GetAccounts()
}

func (b *StatusBackend) SaveAccount(account multiaccounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}
	return b.multiaccountsDB.SaveAccount(account)
}

func (b *StatusBackend) ensureAppDBOpened(account multiaccounts.Account, password string) (err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}
	if len(b.rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}
	path := filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql", account.Address))
	b.appDB, err = appdatabase.InitializeDB(path, password)
	if err != nil {
		return err
	}
	return nil
}

func (b *StatusBackend) SaveAccountAndStartNodeWithKey(acc multiaccounts.Account, conf *params.NodeConfig, password string, keyHex string) error {
	err := b.SaveAccount(acc)
	if err != nil {
		return err
	}
	err = b.ensureAppDBOpened(acc, password)
	if err != nil {
		return err
	}
	err = b.saveNodeConfig(conf)
	if err != nil {
		return err
	}
	return b.StartNodeWithKey(acc, password, keyHex)
}

// StartNodeWithKey instead of loading addresses from database this method derives address from key
// and uses it in application.
func (b *StatusBackend) startNodeWithKey(acc multiaccounts.Account, password string, keyHex string) error {
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

	chatKey, err := ethcrypto.HexToECDSA(keyHex)
	if err != nil {
		return err
	}
	err = b.StartNode(conf)
	if err != nil {
		return err
	}
	b.accountManager.SetChatAccount(chatKey)
	chatAcc, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}
	b.accountManager.SetAccountAddresses(chatAcc.Address)
	err = b.injectAccountIntoServices()
	if err != nil {
		return err
	}
	err = b.multiaccountsDB.UpdateAccountTimestamp(acc.Address, time.Now().Unix())
	if err != nil {
		return err
	}
	return nil
}

func (b *StatusBackend) StartNodeWithKey(acc multiaccounts.Account, password string, keyHex string) error {
	err := b.startNodeWithKey(acc, password, keyHex)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	signal.SendLoggedIn(err)
	return err
}

func (b *StatusBackend) startNodeWithAccount(acc multiaccounts.Account, password string) error {
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
	err = b.multiaccountsDB.UpdateAccountTimestamp(acc.Address, time.Now().Unix())
	if err != nil {
		return err
	}
	return nil
}

func (b *StatusBackend) StartNodeWithAccount(acc multiaccounts.Account, password string) error {
	err := b.startNodeWithAccount(acc, password)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	signal.SendLoggedIn(err)
	return err
}

// StartNodeWithAccountAndConfig is used after account and config was generated.
// In current setup account name and config is generated on the client side. Once/if it will be generated on
// status-go side this flow can be simplified.
func (b *StatusBackend) StartNodeWithAccountAndConfig(account multiaccounts.Account, password string, conf *params.NodeConfig, subaccs []accounts.Account) error {
	err := b.SaveAccount(account)
	if err != nil {
		return err
	}
	err = b.ensureAppDBOpened(account, password)
	if err != nil {
		return err
	}
	err = b.saveNodeConfig(conf)
	if err != nil {
		return err
	}
	err = accounts.NewDB(b.appDB).SaveAccounts(subaccs)
	if err != nil {
		return err
	}
	return b.StartNodeWithAccount(account, password)
}

func (b *StatusBackend) saveNodeConfig(config *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return accounts.NewDB(b.appDB).SaveConfig(accounts.NodeConfigTag, config)
}

func (b *StatusBackend) loadNodeConfig() (*params.NodeConfig, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	conf := params.NodeConfig{}
	err := accounts.NewDB(b.appDB).GetConfig(accounts.NodeConfigTag, &conf)
	if err != nil {
		return nil, err
	}
	// NodeConfig.Version should be taken from params.Version
	// which is set at the compile time.
	// What's cached is usually outdated so we overwrite it here.
	conf.Version = params.Version
	return &conf, nil
}

func (b *StatusBackend) rpcFiltersService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return rpcfilters.New(b.statusNode), nil
	}
}

func (b *StatusBackend) subscriptionService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return subscriptions.New(b.statusNode), nil
	}
}

func (b *StatusBackend) accountsService(accountsFeed *event.Feed) gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return accountssvc.NewService(accounts.NewDB(b.appDB), b.multiaccountsDB, b.accountManager, accountsFeed), nil
	}
}

func (b *StatusBackend) browsersService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return browsers.NewService(browsers.NewDB(b.appDB)), nil
	}
}

func (b *StatusBackend) permissionsService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return permissions.NewService(permissions.NewDB(b.appDB)), nil
	}
}

func (b *StatusBackend) mailserversService() gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return mailservers.NewService(mailservers.NewDB(b.appDB)), nil
	}
}

func (b *StatusBackend) walletService(network uint64, accountsFeed *event.Feed) gethnode.ServiceConstructor {
	return func(*gethnode.ServiceContext) (gethnode.Service, error) {
		return wallet.NewService(wallet.NewDB(b.appDB, network), accountsFeed), nil
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

// GetNodesFromContract returns a list of nodes from the contract
func (b *StatusBackend) GetNodesFromContract(rpcEndpoint string, contractAddress string) ([]string, error) {
	var response []string

	ctx, cancel := context.WithTimeout(context.Background(), contractQueryTimeout)
	defer cancel()

	ethclient, err := ethclient.DialContext(ctx, rpcEndpoint)
	if err != nil {
		return response, err
	}

	contract, err := registry.NewNodes(gethcommon.HexToAddress(contractAddress), ethclient)
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
func (b *StatusBackend) CallPrivateRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCPrivateClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// SendTransaction creates a new transaction and waits until it's complete.
func (b *StatusBackend) SendTransaction(sendArgs transactions.SendTxArgs, password string) (hash gethcommon.Hash, err error) {
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
	verifiedAccount, err := b.getVerifiedWalletAccount(rpcParams.Address, rpcParams.Password)
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
func (b *StatusBackend) SignTypedData(typed typeddata.TypedData, address string, password string) (hexutil.Bytes, error) {
	account, err := b.getVerifiedWalletAccount(address, password)
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

// HashTypedData generates the hash of TypedData.
func (b *StatusBackend) HashTypedData(typed typeddata.TypedData) (common.Hash, error) {
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	hash, err := typeddata.ValidateAndHash(typed, chain)
	if err != nil {
		return common.Hash{}, err
	}
	return hash, err
}

func (b *StatusBackend) getVerifiedWalletAccount(address, password string) (*account.SelectedExtKey, error) {
	config := b.StatusNode().Config()

	db := accounts.NewDB(b.appDB)
	exists, err := db.AddressExists(common.HexToAddress(address))
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
func (b *StatusBackend) cleanupServices() error {
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

func (b *StatusBackend) closeAppDB() error {
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

// reSelectAccount selects previously selected account, often, after node restart.
func (b *StatusBackend) reSelectAccount() error {
	b.AccountManager().RemoveOnboarding()

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
func (b *StatusBackend) SelectAccount(loginParams account.LoginParams) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.AccountManager().RemoveOnboarding()

	err := b.accountManager.SelectAccount(loginParams)
	if err != nil {
		return err
	}

	return b.injectAccountIntoServices()
}

func (b *StatusBackend) injectAccountIntoServices() error {
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

		if err := st.InitProtocol(b.appDB); err != nil {
			return err
		}
	}
	return b.startWallet()
}

func (b *StatusBackend) startWallet() error {
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
	allAddresses[0] = mainAccountAddress
	copy(allAddresses[1:], watchAddresses)
	return wallet.StartReactor(
		b.statusNode.RPCClient().Ethclient(),
		allAddresses,
		new(big.Int).SetUint64(b.statusNode.Config().NetworkID))
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

		if err := st.InitProtocol(b.appDB); err != nil {
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
func (b *StatusBackend) DisableInstallation(installationID string) error {
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

// SignHash exposes vanilla ECDSA signing for signing a message for Swarm
func (b *StatusBackend) SignHash(hexEncodedHash string) (string, error) {
	hash, err := hexutil.Decode(hexEncodedHash)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not unmarshal the input: %v", err)
	}

	chatAccount, err := b.AccountManager().SelectedChatAccount()
	if err != nil {
		return "", fmt.Errorf("SignHash: could not select account: %v", err.Error())
	}

	signature, err := ethcrypto.Sign(hash, chatAccount.AccountKey.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("SignHash: could not sign the hash: %v", err)
	}

	hexEncodedSignature := protocol.EncodeHex(signature)
	return hexEncodedSignature, nil
}
