package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/imdario/mergo"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/connection"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/services/personal"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

var (
	// ErrWhisperClearIdentitiesFailure clearing whisper identities has failed.
	ErrWhisperClearIdentitiesFailure = errors.New("failed to clear whisper identities")
	// ErrWhisperIdentityInjectionFailure injecting whisper identities has failed.
	ErrWhisperIdentityInjectionFailure = errors.New("failed to inject identity into Whisper")
	// ErrWakuIdentityInjectionFailure injecting whisper identities has failed.
	ErrWakuIdentityInjectionFailure = errors.New("failed to inject identity into waku")
	// ErrUnsupportedRPCMethod is for methods not supported by the RPC interface
	ErrUnsupportedRPCMethod = errors.New("method is unsupported by RPC interface")
	// ErrRPCClientUnavailable is returned if an RPC client can't be retrieved.
	// This is a normal situation when a node is stopped.
	ErrRPCClientUnavailable = errors.New("JSON-RPC client is unavailable")
	// ErrDBNotAvailable is returned if a method is called before the DB is available for usage
	ErrDBNotAvailable = errors.New("DB is unavailable")
	// ErrConfigNotAvailable is returned if a method is called before the nodeconfig is set
	ErrConfigNotAvailable = errors.New("NodeConfig is not available")
)

var _ StatusBackend = (*GethStatusBackend)(nil)

// GethStatusBackend implements the Status.im service over go-ethereum
type GethStatusBackend struct {
	mu sync.Mutex
	// rootDataDir is the same for all networks.
	rootDataDir string
	appDB       *sql.DB
	config      *params.NodeConfig

	statusNode           *node.StatusNode
	personalAPI          *personal.PublicAPI
	multiaccountsDB      *multiaccounts.Database
	account              *multiaccounts.Account
	accountManager       *account.GethManager
	transactor           *transactions.Transactor
	connectionState      connection.State
	appState             appState
	selectedAccountKeyID string
	log                  log.Logger
	allowAllRPC          bool // used only for tests, disables api method restrictions

}

// NewGethStatusBackend create a new GethStatusBackend instance
func NewGethStatusBackend() *GethStatusBackend {
	defer log.Info("Status backend initialized", "backend", "geth", "version", params.Version, "commit", params.GitCommit, "IpfsGatewayURL", params.IpfsGatewayURL)

	backend := &GethStatusBackend{}
	backend.initialize()
	return backend
}

func (b *GethStatusBackend) initialize() {
	accountManager := account.NewGethManager()
	transactor := transactions.NewTransactor()
	personalAPI := personal.NewAPI()
	statusNode := node.New(transactor)

	b.statusNode = statusNode
	b.accountManager = accountManager
	b.transactor = transactor
	b.personalAPI = personalAPI
	b.statusNode.SetMultiaccountsDB(b.multiaccountsDB)
	b.log = log.New("package", "status-go/api.GethStatusBackend")
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

// SelectedAccountKeyID returns a Whisper key ID of the selected chat key pair.
func (b *GethStatusBackend) SelectedAccountKeyID() string {
	return b.selectedAccountKeyID
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

func (b *GethStatusBackend) GetMultiaccountDB() *multiaccounts.Database {
	return b.multiaccountsDB
}

func (b *GethStatusBackend) OpenAccounts() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB != nil {
		return nil
	}
	db, err := multiaccounts.InitializeDB(filepath.Join(b.rootDataDir, "accounts.sql"))
	if err != nil {
		b.log.Error("failed to initialize accounts db", "err", err)
		return err
	}
	b.multiaccountsDB = db
	// Probably we should iron out a bit better how to create/dispose of the status-service
	b.statusNode.SetMultiaccountsDB(db)
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

func (b *GethStatusBackend) DeleteMultiaccount(keyUID string, keyStoreDir string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.multiaccountsDB == nil {
		return errors.New("accounts db wasn't initialized")
	}

	err := b.multiaccountsDB.DeleteAccount(keyUID)
	if err != nil {
		return err
	}

	dbFiles := []string{
		filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql-shm", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql-wal", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db-shm", keyUID)),
		filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db-wal", keyUID)),
	}
	for _, path := range dbFiles {
		if _, err := os.Stat(path); err == nil {
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}
	}

	return os.RemoveAll(keyStoreDir)
}

func (b *GethStatusBackend) DeleteImportedKey(address, password, keyStoreDir string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := filepath.Walk(keyStoreDir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(fileInfo.Name(), address) {
			_, err := b.accountManager.VerifyAccountPassword(keyStoreDir, "0x"+address, password)
			if err != nil {
				b.log.Error("failed to verify account", "account", address, "error", err)
				return err
			}

			return os.Remove(path)
		}
		return nil
	})

	return err
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

	// Migrate file path to fix issue https://github.com/status-im/status-go/issues/2027
	oldPath := filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql", account.KeyUID))
	newPath := filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db", account.KeyUID))

	_, err = os.Stat(oldPath)
	if err == nil {
		err := os.Rename(oldPath, newPath)
		if err != nil {
			return err
		}

		// rename journals as well, but ignore errors
		_ = os.Rename(oldPath+"-shm", newPath+"-shm")
		_ = os.Rename(oldPath+"-wal", newPath+"-wal")
	}

	b.appDB, err = appdatabase.InitializeDB(newPath, password, account.KDFIterations)
	if err != nil {
		b.log.Error("failed to initialize db", "err", err)
		return err
	}
	b.statusNode.SetAppDB(b.appDB)
	return nil
}

func (b *GethStatusBackend) setupLogSettings() error {
	logSettings := logutils.LogSettings{
		Enabled:         b.config.LogEnabled,
		MobileSystem:    b.config.LogMobileSystem,
		Level:           b.config.LogLevel,
		File:            b.config.LogFile,
		MaxSize:         b.config.LogMaxSize,
		MaxBackups:      b.config.LogMaxBackups,
		CompressRotated: b.config.LogCompressRotated,
	}
	if err := logutils.OverrideRootLogWithConfig(logSettings, false); err != nil {
		return err
	}
	return nil
}

// StartNodeWithKey instead of loading addresses from database this method derives address from key
// and uses it in application.
// TODO: we should use a proper struct with optional values instead of duplicating the regular functions
// with small variants for keycard, this created too many bugs
func (b *GethStatusBackend) startNodeWithKey(acc multiaccounts.Account, password string, keyHex string) error {
	if acc.KDFIterations == 0 {
		kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(acc.KeyUID)
		if err != nil {
			return err
		}

		acc.KDFIterations = kdfIterations
	}

	err := b.ensureAppDBOpened(acc, password)
	if err != nil {
		return err
	}

	err = b.loadNodeConfig(nil)
	if err != nil {
		return err
	}

	err = b.setupLogSettings()
	if err != nil {
		return err
	}

	b.account = &acc
	accountsDB, err := accounts.NewDB(b.appDB)
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
	chatKey, err := ethcrypto.HexToECDSA(keyHex)
	if err != nil {
		return err
	}
	err = b.StartNode(b.config)
	if err != nil {
		return err
	}
	if err := b.accountManager.SetChatAccount(chatKey); err != nil {
		return err
	}
	_, err = b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}
	b.accountManager.SetAccountAddresses(walletAddr, watchAddrs...)
	err = b.injectAccountsIntoServices()
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

func (b *GethStatusBackend) OverwriteNodeConfigValues(conf *params.NodeConfig, n *params.NodeConfig) (*params.NodeConfig, error) {
	if err := mergo.Merge(conf, n, mergo.WithOverride); err != nil {
		return nil, err
	}

	conf.Networks = n.Networks

	if err := b.saveNodeConfig(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func (b *GethStatusBackend) startNodeWithAccount(acc multiaccounts.Account, password string, inputNodeCfg *params.NodeConfig) error {
	err := b.ensureAppDBOpened(acc, password)
	if err != nil {
		return err
	}

	err = b.loadNodeConfig(inputNodeCfg)
	if err != nil {
		return err
	}

	err = b.setupLogSettings()
	if err != nil {
		return err
	}

	b.account = &acc
	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}
	chatAddr, err := accountsDB.GetChatAddress()
	if err != nil {
		return err
	}
	walletAddr, err := accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}
	watchAddrs, err := accountsDB.GetWalletAddresses()
	if err != nil {
		return err
	}
	login := account.LoginParams{
		Password:       password,
		ChatAddress:    chatAddr,
		WatchAddresses: watchAddrs,
		MainAccount:    walletAddr,
	}

	err = b.StartNode(b.config)
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

func (b *GethStatusBackend) MigrateKeyStoreDir(acc multiaccounts.Account, password, oldDir, newDir string) error {
	err := b.ensureAppDBOpened(acc, password)
	if err != nil {
		return err
	}

	accountDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}
	accounts, err := accountDB.GetAccounts()
	if err != nil {
		return err
	}
	settings, err := accountDB.GetSettings()
	if err != nil {
		return err
	}
	addresses := []string{settings.EIP1581Address.Hex(), settings.WalletRootAddress.Hex()}
	for _, account := range accounts {
		addresses = append(addresses, account.Address.Hex())
	}
	err = b.accountManager.MigrateKeyStoreDir(oldDir, newDir, addresses)
	if err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) StartNodeWithAccount(acc multiaccounts.Account, password string, nodecfg *params.NodeConfig) error {
	err := b.startNodeWithAccount(acc, password, nodecfg)
	if err != nil {
		// Stop node for clean up
		_ = b.StopNode()
	}
	signal.SendLoggedIn(err)
	return err
}

func (b *GethStatusBackend) ExportUnencryptedDatabase(acc multiaccounts.Account, password, directory string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}
	if len(b.rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}

	// Migrate file path to fix issue https://github.com/status-im/status-go/issues/2027
	oldPath := filepath.Join(b.rootDataDir, fmt.Sprintf("app-%x.sql", acc.KeyUID))
	newPath := filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db", acc.KeyUID))

	_, err := os.Stat(oldPath)
	if err == nil {
		err := os.Rename(oldPath, newPath)
		if err != nil {
			return err
		}

		// rename journals as well, but ignore errors
		_ = os.Rename(oldPath+"-shm", newPath+"-shm")
		_ = os.Rename(oldPath+"-wal", newPath+"-wal")
	}

	err = appdatabase.DecryptDatabase(newPath, directory, password, acc.KDFIterations)
	if err != nil {
		b.log.Error("failed to initialize db", "err", err)
		return err
	}
	return nil
}

func (b *GethStatusBackend) ImportUnencryptedDatabase(acc multiaccounts.Account, password, databasePath string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.appDB != nil {
		return nil
	}
	if len(b.rootDataDir) == 0 {
		return errors.New("root datadir wasn't provided")
	}

	path := filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db", acc.KeyUID))

	err := appdatabase.EncryptDatabase(databasePath, path, password, acc.KDFIterations)
	if err != nil {
		b.log.Error("failed to initialize db", "err", err)
		return err
	}
	return nil
}

func (b *GethStatusBackend) ChangeDatabasePassword(keyUID string, password string, newPassword string) error {
	dbPath := filepath.Join(b.rootDataDir, fmt.Sprintf("%s.db", keyUID))
	config := b.StatusNode().Config()

	if config != nil {
		keyDir := config.KeyStoreDir
		err := b.accountManager.ReEncryptKeyStoreDir(keyDir, password, newPassword)
		if err != nil {
			return fmt.Errorf("ReEncryptKeyStoreDir error: %v", err)
		}
	}

	kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(keyUID)
	if err != nil {
		return err
	}
	err = appdatabase.ChangeDatabasePassword(dbPath, password, kdfIterations, newPassword)
	if err != nil {
		if config != nil {
			keyDir := config.KeyStoreDir
			// couldn't change db password so undo keystore changes to mainitain consistency
			_ = b.accountManager.ReEncryptKeyStoreDir(keyDir, newPassword, password)
		}
		return err
	}
	return nil
}

func (b *GethStatusBackend) tryToDeleteAccount(address types.Address, password, newPassword string) error {
	err := b.accountManager.DeleteAccount(address, newPassword)
	if err != nil {
		err = b.accountManager.DeleteAccount(address, password)
		if err != nil && err.Error() != "no key for given address or file" {
			b.log.Error("error on deleting account", "err", err)
			return err
		}
	}

	return nil
}

func (b *GethStatusBackend) ConvertToKeycardAccount(account multiaccounts.Account, s settings.Settings, password string, newPassword string) error {
	err := b.multiaccountsDB.UpdateAccountKeycardPairing(account.KeyUID, account.KeycardPairing)
	if err != nil {
		return err
	}

	err = b.ensureAppDBOpened(account, password)
	if err != nil {
		return err
	}

	accountDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}
	err = accountDB.SaveSettingField(settings.KeycardInstanceUID, s.KeycardInstanceUID)
	if err != nil {
		return err
	}

	err = accountDB.SaveSettingField(settings.KeycardPairedOn, s.KeycardPairedOn)
	if err != nil {
		return err
	}

	err = accountDB.SaveSettingField(settings.KeycardPairing, s.KeycardPairing)
	if err != nil {
		return err
	}

	err = accountDB.SaveSettingField(settings.Mnemonic, nil)
	if err != nil {
		return err
	}

	knownAccounts, err := accountDB.GetAccounts()
	if err != nil {
		return err
	}

	masterAddress, err := accountDB.GetMasterAddress()
	if err != nil {
		return err
	}

	dappsAddress, err := accountDB.GetDappsAddress()
	if err != nil {
		return err
	}

	eip1581Address, err := accountDB.GetEIP1581Address()
	if err != nil {
		return err
	}

	walletRootAddress, err := accountDB.GetWalletRootAddress()
	if err != nil {
		return err
	}

	err = b.closeAppDB()
	if err != nil {
		return err
	}

	err = b.ChangeDatabasePassword(account.KeyUID, password, newPassword)
	if err != nil {
		return err
	}

	// We need to delete all accounts for the keypair which is being migrated
	// no need to check for the error below cause if this action fails from
	// whichever reason the account is still successfully migrated
	for _, acc := range knownAccounts {
		if account.KeyUID == acc.KeyUID {
			err = b.tryToDeleteAccount(acc.Address, password, newPassword)
			if err != nil {
				return err
			}
		}
	}

	err = b.tryToDeleteAccount(masterAddress, password, newPassword)
	if err != nil {
		return err
	}

	err = b.tryToDeleteAccount(dappsAddress, password, newPassword)
	if err != nil {
		return err
	}

	err = b.tryToDeleteAccount(eip1581Address, password, newPassword)
	if err != nil {
		return err
	}

	err = b.tryToDeleteAccount(walletRootAddress, password, newPassword)
	if err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) ConvertToRegularAccount(mnemonic string, currPassword string, newPassword string) error {
	mnemonicNoExtraSpaces := strings.Join(strings.Fields(mnemonic), " ")
	accountInfo, err := b.accountManager.AccountsGenerator().ImportMnemonic(mnemonicNoExtraSpaces, "")
	if err != nil {
		return err
	}

	kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(accountInfo.KeyUID)
	if err != nil {
		return err
	}

	err = b.ensureAppDBOpened(multiaccounts.Account{KeyUID: accountInfo.KeyUID, KDFIterations: kdfIterations}, newPassword)
	if err != nil {
		return err
	}

	db, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	knownAccounts, err := db.GetAccounts()
	if err != nil {
		return err
	}

	// We add these two paths, cause others will be added via `StoreAccount` function call
	const pathWalletRoot = "m/44'/60'/0'/0"
	const pathEIP1581 = "m/43'/60'/1581'"
	var paths []string
	paths = append(paths, pathWalletRoot, pathEIP1581)
	for _, acc := range knownAccounts {
		if accountInfo.KeyUID == acc.KeyUID {
			paths = append(paths, acc.Path)
		}
	}

	_, err = b.accountManager.AccountsGenerator().StoreAccount(accountInfo.ID, currPassword)
	if err != nil {
		return err
	}

	_, err = b.accountManager.AccountsGenerator().StoreDerivedAccounts(accountInfo.ID, currPassword, paths)
	if err != nil {
		return err
	}

	err = b.multiaccountsDB.UpdateAccountKeycardPairing(accountInfo.KeyUID, "")
	if err != nil {
		return err
	}

	err = db.DeleteKeypair(accountInfo.KeyUID)
	if err != nil {
		return err
	}

	err = db.SaveSettingField(settings.KeycardInstanceUID, "")
	if err != nil {
		return err
	}

	err = db.SaveSettingField(settings.KeycardPairedOn, 0)
	if err != nil {
		return err
	}

	err = db.SaveSettingField(settings.KeycardPairing, "")
	if err != nil {
		return err
	}

	err = b.closeAppDB()
	if err != nil {
		return err
	}

	return b.ChangeDatabasePassword(accountInfo.KeyUID, currPassword, newPassword)
}

func (b *GethStatusBackend) VerifyDatabasePassword(keyUID string, password string) error {
	kdfIterations, err := b.multiaccountsDB.GetAccountKDFIterationsNumber(keyUID)
	if err != nil {
		return err
	}

	err = b.ensureAppDBOpened(multiaccounts.Account{KeyUID: keyUID, KDFIterations: kdfIterations}, password)
	if err != nil {
		return err
	}

	accountsDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}
	_, err = accountsDB.GetWalletAddress()
	if err != nil {
		return err
	}

	err = b.closeAppDB()
	if err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) SaveAccountAndStartNodeWithKey(acc multiaccounts.Account, password string, settings settings.Settings, nodecfg *params.NodeConfig, subaccs []*accounts.Account, keyHex string) error {
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

// StartNodeWithAccountAndInitialConfig is used after account and config was generated.
// In current setup account name and config is generated on the client side. Once/if it will be generated on
// status-go side this flow can be simplified.
func (b *GethStatusBackend) StartNodeWithAccountAndInitialConfig(
	account multiaccounts.Account,
	password string,
	settings settings.Settings,
	nodecfg *params.NodeConfig,
	subaccs []*accounts.Account,
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
	return b.StartNodeWithAccount(account, password, nodecfg)
}

func (b *GethStatusBackend) saveAccountsAndSettings(settings settings.Settings, nodecfg *params.NodeConfig, subaccs []*accounts.Account) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	accdb, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}
	err = accdb.CreateSettings(settings, *nodecfg)
	if err != nil {
		return err
	}

	// In case of setting up new account either way (creating new, importing seed phrase, keycard account...) we should not
	// back up any data after login, as it was the case before, that's the reason why we're setting last backup time to the time
	// when an account was created.
	now := time.Now().Unix()
	err = accdb.SetLastBackup(uint64(now))
	if err != nil {
		return err
	}

	return accdb.SaveAccounts(subaccs)
}

func (b *GethStatusBackend) loadNodeConfig(inputNodeCfg *params.NodeConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	conf, err := nodecfg.GetNodeConfigFromDB(b.appDB)
	if err != nil {
		return err
	}

	if inputNodeCfg != nil {
		conf, err = b.OverwriteNodeConfigValues(conf, inputNodeCfg)
		if err != nil {
			return err
		}
	}

	// Start WakuV1 if WakuV2 is not enabled
	conf.WakuConfig.Enabled = !conf.WakuV2Config.Enabled
	// NodeConfig.Version should be taken from params.Version
	// which is set at the compile time.
	// What's cached is usually outdated so we overwrite it here.
	conf.Version = params.Version
	conf.RootDataDir = b.rootDataDir
	conf.DataDir = filepath.Join(b.rootDataDir, conf.DataDir)
	conf.ShhextConfig.BackupDisabledDataDir = filepath.Join(b.rootDataDir, conf.ShhextConfig.BackupDisabledDataDir)
	if len(conf.LogDir) == 0 {
		conf.LogFile = filepath.Join(b.rootDataDir, conf.LogFile)
	} else {
		conf.LogFile = filepath.Join(conf.LogDir, conf.LogFile)
	}
	conf.KeyStoreDir = filepath.Join(b.rootDataDir, conf.KeyStoreDir)

	b.config = conf

	return nil
}

func (b *GethStatusBackend) saveNodeConfig(n *params.NodeConfig) error {
	err := nodecfg.SaveNodeConfig(b.appDB, n)
	if err != nil {
		return err
	}
	return nil
}

func (b *GethStatusBackend) GetNodeConfig() (*params.NodeConfig, error) {
	return nodecfg.GetNodeConfigFromDB(b.appDB)
}

func (b *GethStatusBackend) startNode(config *params.NodeConfig) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("node crashed on start: %v", err)
		}
	}()

	b.log.Info("status-go version details", "version", params.Version, "commit", params.GitCommit)
	b.log.Debug("starting node with config", "config", config)
	// Update config with some defaults.
	if err := config.UpdateWithDefaults(); err != nil {
		return err
	}

	// Updating node config
	b.config = config

	b.log.Debug("updated config with defaults", "config", config)

	// Start by validating configuration
	if err := config.Validate(); err != nil {
		return err
	}

	if b.accountManager.GetManager() == nil {
		err = b.accountManager.InitKeystore(config.KeyStoreDir)
		if err != nil {
			return err
		}
	}

	manager := b.accountManager.GetManager()
	if manager == nil {
		return errors.New("ethereum accounts.Manager is nil")
	}

	if err = b.statusNode.StartWithOptions(config, node.StartOptions{
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
	b.personalAPI.SetRPC(b.statusNode.RPCClient(), rpc.DefaultCallTimeout)

	if err = b.registerHandlers(); err != nil {
		b.log.Error("Handler registration failed", "err", err)
		return
	}
	b.log.Info("Handlers registered")

	// Handle a case when a node is stopped and resumed.
	// If there is no account selected, an error is returned.
	if _, err := b.accountManager.SelectedChatAccount(); err == nil {
		if err := b.injectAccountsIntoServices(); err != nil {
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
	if b.statusNode == nil || !b.IsNodeRunning() {
		return nil
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

	if err := b.stopNode(); err != nil {
		return err
	}

	return b.startNode(b.config)
}

// ResetChainData remove chain data from data directory.
// Node is stopped, and new node is started, with clean data directory.
func (b *GethStatusBackend) ResetChainData() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.stopNode(); err != nil {
		return err
	}
	// config is cleaned when node is stopped
	if err := b.statusNode.ResetChainData(b.config); err != nil {
		return err
	}
	signal.SendChainDataRemoved()
	return b.startNode(b.config)
}

// CallRPC executes public RPC requests on node's in-proc RPC server.
func (b *GethStatusBackend) CallRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCClient()
	if client == nil {
		return "", ErrRPCClientUnavailable
	}
	return client.CallRaw(inputJSON), nil
}

// CallPrivateRPC executes public and private RPC requests on node's in-proc RPC server.
func (b *GethStatusBackend) CallPrivateRPC(inputJSON string) (string, error) {
	client := b.statusNode.RPCClient()
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

	go b.statusNode.RPCFiltersService().TriggerTransactionSentToUpstreamEvent(hash)

	return
}

func (b *GethStatusBackend) SendTransactionWithChainID(chainID uint64, sendArgs transactions.SendTxArgs, password string) (hash types.Hash, err error) {
	verifiedAccount, err := b.getVerifiedWalletAccount(sendArgs.From.String(), password)
	if err != nil {
		return hash, err
	}

	hash, err = b.transactor.SendTransactionWithChainID(chainID, sendArgs, verifiedAccount)
	if err != nil {
		return
	}

	go b.statusNode.RPCFiltersService().TriggerTransactionSentToUpstreamEvent(hash)

	return
}

func (b *GethStatusBackend) SendTransactionWithSignature(sendArgs transactions.SendTxArgs, sig []byte) (hash types.Hash, err error) {
	hash, err = b.transactor.SendTransactionWithSignature(sendArgs, sig)
	if err != nil {
		return
	}

	go b.statusNode.RPCFiltersService().TriggerTransactionSentToUpstreamEvent(hash)

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

// SignTypedDataV4 accepts data and password. Gets verified account and signs typed data.
func (b *GethStatusBackend) SignTypedDataV4(typed signercore.TypedData, address string, password string) (types.HexBytes, error) {
	account, err := b.getVerifiedWalletAccount(address, password)
	if err != nil {
		return types.HexBytes{}, err
	}
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	sig, err := typeddata.SignTypedDataV4(typed, account.AccountKey.PrivateKey, chain)
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

// HashTypedDataV4 generates the hash of TypedData.
func (b *GethStatusBackend) HashTypedDataV4(typed signercore.TypedData) (types.Hash, error) {
	chain := new(big.Int).SetUint64(b.StatusNode().Config().NetworkID)
	hash, err := typeddata.HashTypedDataV4(typed, chain)
	if err != nil {
		return types.Hash{}, err
	}
	return types.Hash(hash), err
}

func (b *GethStatusBackend) getVerifiedWalletAccount(address, password string) (*account.SelectedExtKey, error) {
	config := b.StatusNode().Config()
	db, err := accounts.NewDB(b.appDB)
	if err != nil {
		b.log.Error("failed to create new *Database instance", "error", err)
		return nil, err
	}
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
	if _, ok := err.(*account.ErrCannotLocateKeyFile); ok {
		key, err = b.generatePartialAccountKey(db, address, password)
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		b.log.Error("failed to verify account", "account", address, "error", err)
		return nil, err
	}

	return &account.SelectedExtKey{
		Address:    key.Address,
		AccountKey: key,
	}, nil
}

func (b *GethStatusBackend) generatePartialAccountKey(db *accounts.Database, address string, password string) (*types.Key, error) {
	dbPath, err := db.GetPath(types.HexToAddress(address))
	path := "m/" + dbPath[strings.LastIndex(dbPath, "/")+1:]
	if err != nil {
		b.log.Error("failed to get path for given account address", "account", address, "error", err)
		return nil, err
	}

	rootAddress, err := db.GetWalletRootAddress()
	if err != nil {
		return nil, err
	}
	info, err := b.accountManager.AccountsGenerator().LoadAccount(rootAddress.Hex(), password)
	if err != nil {
		return nil, err
	}
	masterID := info.ID

	accInfosMap, err := b.accountManager.AccountsGenerator().StoreDerivedAccounts(masterID, password, []string{path})
	if err != nil {
		return nil, err
	}

	_, key, err := b.accountManager.AddressToDecryptedAccount(accInfosMap[path].Address, password)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// registerHandlers attaches Status callback handlers to running node
func (b *GethStatusBackend) registerHandlers() error {
	var clients []*rpc.Client

	if c := b.StatusNode().RPCClient(); c != nil {
		clients = append(clients, c)
	} else {
		return errors.New("RPC client unavailable")
	}

	for _, client := range clients {
		client.RegisterHandler(
			params.AccountsMethodName,
			func(context.Context, uint64, ...interface{}) (interface{}, error) {
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

func unsupportedMethodHandler(ctx context.Context, chainID uint64, rpcParams ...interface{}) (interface{}, error) {
	return nil, ErrUnsupportedRPCMethod
}

// ConnectionChange handles network state changes logic.
func (b *GethStatusBackend) ConnectionChange(typ string, expensive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := connection.State{
		Type:      connection.NewType(typ),
		Expensive: expensive,
	}
	if typ == connection.None {
		state.Offline = true
	}

	b.log.Info("Network state change", "old", b.connectionState, "new", state)

	b.connectionState = state
	b.statusNode.ConnectionChanged(state)

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

	if b.statusNode != nil {
		wakuext := b.statusNode.WakuExtService()

		if wakuext != nil {
			messenger := wakuext.Messenger()

			if messenger != nil {
				if s == appStateForeground {
					messenger.ToForeground()
				} else {
					messenger.ToBackground()
				}
			}
		}

	}

	// TODO: put node in low-power mode if the app is in background (or inactive)
	// and normal mode if the app is in foreground.
}

func (b *GethStatusBackend) StopLocalNotifications() error {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.StopLocalNotifications()
}

func (b *GethStatusBackend) StartLocalNotifications() error {
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.StartLocalNotifications()

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
	b.appDB = nil

	if b.statusNode != nil {
		if err := b.statusNode.Stop(); err != nil {
			return err
		}
		b.statusNode = nil
	}
	// re-initialize the node, at some point we should better manage the lifecycle
	b.initialize()
	return nil
}

// cleanupServices stops parts of services that doesn't managed by a node and removes injected data from services.
func (b *GethStatusBackend) cleanupServices() error {
	b.selectedAccountKeyID = ""
	if b.statusNode == nil {
		return nil
	}
	return b.statusNode.Cleanup()
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

	if loginParams.MultiAccount != nil {
		b.account = loginParams.MultiAccount
	}

	if err := b.injectAccountsIntoServices(); err != nil {
		return err
	}

	return nil
}

func (b *GethStatusBackend) GetActiveAccount() (*multiaccounts.Account, error) {
	if b.account == nil {
		return nil, errors.New("master key account is nil in the GethStatusBackend")
	}

	return b.account, nil
}

func (b *GethStatusBackend) injectAccountsIntoWakuService(w types.WakuKeyManager, st *ext.Service) error {
	chatAccount, err := b.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}

	identity := chatAccount.AccountKey.PrivateKey

	acc, err := b.GetActiveAccount()
	if err != nil {
		return err
	}

	if err := w.DeleteKeyPairs(); err != nil { // err is not possible; method return value is incorrect
		return err
	}
	b.selectedAccountKeyID, err = w.AddKeyPair(identity)
	if err != nil {
		return ErrWakuIdentityInjectionFailure
	}

	if st != nil {
		if err := st.InitProtocol(b.statusNode.GethNode().Config().Name, identity, b.appDB, b.statusNode.HTTPServer(), b.multiaccountsDB, acc, logutils.ZapLogger()); err != nil {
			return err
		}
		// Set initial connection state
		st.ConnectionChanged(b.connectionState)

		messenger := st.Messenger()
		// Init public status api
		b.statusNode.StatusPublicService().Init(messenger)
		b.statusNode.AccountService().Init(messenger)
		// Init chat service
		accDB, err := accounts.NewDB(b.appDB)
		if err != nil {
			return err
		}
		b.statusNode.ChatService(accDB).Init(messenger)
	}

	return nil
}

func (b *GethStatusBackend) injectAccountsIntoServices() error {
	if b.statusNode.WakuService() != nil {
		return b.injectAccountsIntoWakuService(b.statusNode.WakuService(), func() *ext.Service {
			if b.statusNode.WakuExtService() == nil {
				return nil
			}
			return b.statusNode.WakuExtService().Service
		}())
	}

	if b.statusNode.WakuV2Service() != nil {
		return b.injectAccountsIntoWakuService(b.statusNode.WakuV2Service(), func() *ext.Service {
			if b.statusNode.WakuV2ExtService() == nil {
				return nil
			}
			return b.statusNode.WakuV2ExtService().Service
		}())
	}

	return nil
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

func (b *GethStatusBackend) Messenger() *protocol.Messenger {
	node := b.StatusNode()
	if node != nil {
		accountService := node.AccountService()
		if accountService != nil {
			return accountService.GetMessenger()
		}
	}
	return nil
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

func (b *GethStatusBackend) SwitchFleet(fleet string, conf *params.NodeConfig) error {
	if b.appDB == nil {
		return ErrDBNotAvailable
	}

	accountDB, err := accounts.NewDB(b.appDB)
	if err != nil {
		return err
	}

	err = accountDB.SaveSetting("fleet", fleet)
	if err != nil {
		return err
	}

	err = nodecfg.SaveNodeConfig(b.appDB, conf)
	if err != nil {
		return err
	}

	return nil
}
