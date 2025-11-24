package backend

import (
	"database/sql"
	errorsog "errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	accscommon "github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	accsmanagementtypes "github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	multiacccommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/params"
	requests2 "github.com/status-im/status-go/pkg/backend/requests"
	identityutils "github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/walletdatabase"
)

const (
	walletAccountDefaultName = "Account 1"
)

type ActiveAccount struct {
	account     *multiaccounts.Account
	appDB       *sql.DB
	walletDB    *sql.DB
	accsManager *accsmanagement.AccountsManager

	// nodeConfig is an old solution to keep node configuration.
	// It's value is calculated based on persisted settings and runtime overrides. It's calculated once and cached.
	// TODO: this should be removed. Instead, load ServicesConfig. And services should read their configs from database.
	// If any runtime overrides are needed, they should defined accordingly, and passed as "runtime overrides" login parameters.
	nodeConfig *params.NodeConfig

	// accountsDB is a wrapper around appDB
	//TODO: get rid of this wrapper, use a settings service instead.
	accountsDB *accounts.Database
}

func (a *ActiveAccount) Close() error {
	var errs []error

	err := a.appDB.Close()
	if err != nil {
		errs = append(errs, err)
	}

	err = a.walletDB.Close()
	if err != nil {
		errs = append(errs, err)
	}

	a.accsManager.Logout()

	a.appDB = nil
	a.walletDB = nil
	a.accsManager = nil

	return errorsog.Join(errs...)
}

func (a *ActiveAccount) GetSettings() (*settings.Settings, error) {
	s, err := a.accountsDB.GetSettings()
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (a *ActiveAccount) SetNodeConfig(rootDataDir string, request *requests2.Login) error {
	// FIXME: Problem is that currently we don't save NodeConfig into database.
	//        Hence, we can't read it from the database.

}

// GetNodeConfig returns a node configuration based on the persisted settings and given runtime overrides.
// TODO: This is a temporary solution. Instead, we should
func (a *ActiveAccount) GetNodeConfig() (*params.NodeConfig, error) {
	if a.nodeConfig != nil {
		// Used cached value
		return a.nodeConfig, nil
	}

	//return a.accountsDB.GetNodeConfig()
}

// ServicesConfig reads accounts settings from database and returns a configuration of services that should be started.
// NOTE: Currently this is an adapter to NodeConfig, later should be stored in the database as is.
func (a *ActiveAccount) ServicesConfig() (*ServicesConfig, error) {
	nodeConfig, err := a.GetNodeConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get node config")
	}

	cfg := &ServicesConfig{
		browserEnabled:            nodeConfig.BrowsersConfig.Enabled,
		permissionsServiceEnabled: nodeConfig.PermissionsConfig.Enabled,
		connectorEnabled:          nodeConfig.ConnectorConfig.Enabled,
		walletEnabled:             nodeConfig.WalletConfig.Enabled,
		wakuV2ExtEnabled:          nodeConfig.WakuV2Config.Enabled,
	}

	return cfg, nil
}

func createAccount(dataDir string, logger *zap.Logger, request *requests2.CreateAccount) (*ActiveAccount, error) {
	activeAccount := &ActiveAccount{}

	// 0. Generate mnemonic
	mnemonic, err := accscommon.CreateRandomMnemonicWithDefaultLength()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create random mnemonic")
	}

	// 1. Generate account from mnemonic
	genMasterAcc, err := generator.CreateAccountFromMnemonic(mnemonic, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create master account from mnemonic")
	}

	keyUID := genMasterAcc.KeyUID()
	masterAddress := genMasterAcc.Address().Hex()

	// 2. Generate derived addresses
	derivationPaths := []string{
		accscommon.PathWalletRoot,
		accscommon.PathEIP1581Root,
		accscommon.PathEIP1581Chat,
		accscommon.PathDefaultWalletAccount,
		accscommon.PathEIP1581Encryption,
	}
	_, derivedAddresses, err := generateDerivedAddresses(genMasterAcc, derivationPaths)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate derived addresses")
	}

	// 3. Get default settings
	settings, err := createSettings(request, mnemonic, keyUID, masterAddress, derivedAddresses)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare settings")
	}

	// 4. Apply request to default settings
	// NOTE: currently done as part of createSettings

	// 5. Build account
	customizationColorClock := uint64(1)
	chatPublicKey := derivedAddresses[accscommon.PathEIP1581Chat].PublicKey
	hasAcceptedTerms := true // (accountsCount == 0)
	activeAccount.account, err = buildAccount(request, keyUID, customizationColorClock, chatPublicKey, hasAcceptedTerms)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build account")
	}

	// 6. Create app database
	activeAccount.appDB, err = createAppDatabase(dataDir, activeAccount.account.KeyUID, activeAccount.account.KDFIterations, request.Password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create app database")
	}

	accdb, err := accounts.NewDB(activeAccount.appDB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create accounts db")
	}

	// 7. Create wallet database
	activeAccount.walletDB, err = createWalletDatabase(dataDir, activeAccount.account.KeyUID, activeAccount.account.KDFIterations, request.Password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wallet database")
	}

	// 8. Create accounts manager
	activeAccount.accsManager, err = createAccountsManager(logger.Named("accounts-manager"), dataDir, accdb)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create accounts manager")
	}

	// 9. Create and save keypair (using accounts manager
	walletAccount := &accsmanagementtypes.AccountCreationDetails{
		Path:    accscommon.PathDefaultWalletAccount,
		Name:    walletAccountDefaultName,
		ColorID: request.CustomizationColor,
	}
	_, err = activeAccount.accsManager.CreateKeypairFromMnemonicAndStore(
		mnemonic,
		request.Password,
		request.DisplayName,
		walletAccount,
		true,
		0,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wallet account")
	}

	// TEMP: write nodeConfig to database
	nodeConfig, err := createNodeConfig(request, "", activeAccount.account.KeyUID, dataDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create node config")
	}

	// 10. Save values to DB
	err = accdb.CreateSettings(*settings, *nodeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create settings")
	}

	// 11. Return created account
	return activeAccount, nil
}

func createKeycardAccount(dataDir string, logger *zap.Logger) (*ActiveAccount, error) {
	return nil, errors.New("not implemented")
}

// login opens given account databases, and creates an account manager.
// Returns ActiveAccount with all created objects.
// Note that it does not operate with multiaccounts database.
func login(dataDir string, logger *zap.Logger, account *multiaccounts.Account, password string) (*ActiveAccount, error) {
	var err error
	activeAccount := &ActiveAccount{
		account: account,
	}

	// 1. Open databases and run migrations
	activeAccount.appDB, err = openAppDatabase(dataDir, account, password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open app database")
	}

	activeAccount.walletDB, err = openWalletDatabase(dataDir, account, password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open wallet database")
	}

	// 2. Create accounts manager
	accountsDB, err := accounts.NewDB(activeAccount.appDB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new *Database instance")
	}

	activeAccount.accsManager, err = createAccountsManager(logger, dataDir, accountsDB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create accounts manager")
	}

	chatAddress, err := accountsDB.GetChatAddress()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get chat address")
	}

	activeAccount.accsManager.SetRootDataDir(dataDir)
	activeAccount.accsManager.SetPersistence(accountsDB)
	err = activeAccount.accsManager.SetChatAccount(chatAddress, password, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set chat account")
	}

	// 3. Return logged in account
	return activeAccount, nil
}

func restoreAccount(mnemonic string) error {
	//settings := createSettings()
	//settings.Mnemonic = ""
	//settings.MnemonicWasNotShown = false

	return errors.New("not implemented")
}

func generateDerivedAddresses(genAcc *generator.Account, paths []string) (genDerivedAccounts map[string]*generator.Account, genDerivedAccountsInfo map[string]generator.AccountInfo, err error) {
	genDerivedAccounts, err = generator.DeriveChildrenFromAccount(genAcc, paths)
	if err != nil {
		return
	}

	genDerivedAccountsInfo = make(map[string]generator.AccountInfo, 0)
	for path, acc := range genDerivedAccounts {
		genDerivedAccountsInfo[path] = acc.ToAccountInfo()
	}

	return
}

func getAppDBPath(rootDataDir string, keyUID string) (string, error) {
	if len(rootDataDir) == 0 {
		return "", errors.New("empty root data dir")
	}

	return filepath.Join(rootDataDir, fmt.Sprintf("%s-v4.db", keyUID)), nil
}

func getWalletDBPath(rootDataDir string, keyUID string) (string, error) {
	if len(rootDataDir) == 0 {
		return "", errors.New("root datadir wasn't provided")
	}

	return filepath.Join(rootDataDir, fmt.Sprintf("%s-wallet.db", keyUID)), nil
}

func createAppDatabase(rootDataDir string, keyUID string, kdfIterations int, password string) (*sql.DB, error) {
	// WARNING: Decide if we want to drop this migration
	//dbFilePath, err := s.runDBFileMigrations(account, password)
	//if err != nil {
	//	return errors.New("Failed to migrate db file: " + err.Error())
	//}

	dbFilePath, err := getAppDBPath(rootDataDir, keyUID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database file path")
	}

	db, err := appdatabase.InitializeDB(dbFilePath, password, kdfIterations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize db")
	}

	//accountsDB, err := accounts.NewDB(s.db)
	//if err != nil {
	//	s.logger.Error("failed to create new *Database instance", zap.Error(err))
	//	return
	//}
	//s.accountsManager.SetPersistence(accountsDB)

	return db, nil
}

func createWalletDatabase(rootDataDir string, keyUID string, kdfIterations int, password string) (*sql.DB, error) {
	dbWalletPath, err := getWalletDBPath(rootDataDir, keyUID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get wallet database file path")
	}

	db, err := walletdatabase.InitializeDB(dbWalletPath, password, kdfIterations)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize wallet db")
	}

	return db, nil
}

func createAccountsManager(logger *zap.Logger, rootDataDir string, accdb *accounts.Database) (*accsmanagement.AccountsManager, error) {
	accountsManager, err := accsmanagement.NewAccountsManager(logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AccountsManager")
	}

	accountsManager.SetRootDataDir(rootDataDir)
	accountsManager.SetPersistence(accdb)
	return accountsManager, nil
}

func openAppDatabase(rootDataDir string, account *multiaccounts.Account, password string) (*sql.DB, error) {
	// WARNING: Can we drop this migration already?
	//dbFilePath, err := b.runDBFileMigrations(account, password)
	//if err != nil {
	//	return errors.New("Failed to migrate db file: " + err.Error())
	//}

	dbFilePath := filepath.Join(rootDataDir, fmt.Sprintf("%s-v4.db", account.KeyUID))
	return appdatabase.InitializeDB(dbFilePath, password, account.KDFIterations)
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed to initialize db")
	//}

	//return appDB, nil
}

func openWalletDatabase(rootDataDir string, account *multiaccounts.Account, password string) (*sql.DB, error) {
	//dbWalletPath, err := b.getWalletDBPath(account.KeyUID)
	//if err != nil {
	//	return err
	//}

	dbFilePath := filepath.Join(rootDataDir, fmt.Sprintf("%s-wallet.db", account.KeyUID))
	return walletdatabase.InitializeDB(dbFilePath, password, account.KDFIterations)
	//if err != nil {
	//	b.logger.Error("failed to initialize wallet db", zap.Error(err))
	//	return err
	//}
	//b.statusNode.SetWalletDB(b.walletDB)
	//return nil
}

func createSettings(request *requests2.CreateAccount, mnemonic string, keyUID string, masterAddress string, derivedAddresses map[string]generator.AccountInfo) (*settings.Settings, error) {
	newSettings, err := api.DefaultSettings(keyUID, masterAddress, derivedAddresses)
	if err != nil {
		return nil, err
	}

	newSettings.DeviceName = request.DeviceName
	newSettings.DisplayName = request.DisplayName
	newSettings.PreviewPrivacy = request.PreviewPrivacy
	//newSettings.CurrentNetwork = request.CurrentNetwork
	//newSettings.TestNetworksEnabled = request.TestNetworksEnabled
	//newSettings.AutoRefreshTokensEnabled = request.AutoRefreshTokensEnabled

	newSettings.Mnemonic = &mnemonic
	newSettings.MnemonicWasNotShown = true

	//if request.WakuV2Fleet != "" {
	//	newSettings.Fleet = &request.WakuV2Fleet
	//}

	newSettings.ThirdpartyServicesEnabled = request.ThirdpartyServicesEnabled

	return newSettings, nil
}

func buildAccount(request *requests2.CreateAccount, keyUID string, customizationColorClock uint64, chatKey string, hasAcceptedTerms bool) (*multiaccounts.Account, error) {
	//err := s.OpenAccounts(request.ThirdpartyServicesEnabled)
	//if err != nil {
	//	return nil, err
	//}

	acc := &multiaccounts.Account{
		KeyUID:                  keyUID,
		Name:                    request.DisplayName,
		CustomizationColor:      multiacccommon.CustomizationColor(request.CustomizationColor),
		CustomizationColorClock: customizationColorClock,
		KDFIterations:           request.KdfIterations,
		Timestamp:               time.Now().Unix(),
	}

	if acc.KDFIterations == 0 {
		acc.KDFIterations = dbsetup.ReducedKDFIterationsNumber
	}

	if request.ImagePath != "" {
		imageCropRectangle := request.ImageCropRectangle
		if imageCropRectangle == nil {
			// Default crop rectangle used by mobile
			imageCropRectangle = &requests2.ImageCropRectangle{
				Ax: 0,
				Ay: 0,
				Bx: 1000,
				By: 1000,
			}
		}

		iis, err := images.GenerateIdentityImages(request.ImagePath,
			imageCropRectangle.Ax, imageCropRectangle.Ay, imageCropRectangle.Bx, imageCropRectangle.By)

		if err != nil {
			return nil, errors.Wrap(err, "failed to generate identity images")
		}
		acc.Images = iis
	}

	var err error
	acc.ColorHash, err = colorhash.GenerateFor(chatKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate color hash")
	}

	acc.ColorID, err = identityutils.ToColorID(chatKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate color id")
	}

	return acc, nil
}

func createNodeConfig(request *requests2.CreateAccount, installationID string, keyUID string, rootDataDir string) (*params.NodeConfig, error) {
	oldRequest := requests2.CreateAccountAdapter(request, rootDataDir)
	return api.DefaultNodeConfig(installationID, keyUID, oldRequest)
}

func (a *ActiveAccount) loginNodeConfig(rootDataDir string, request *requests2.Login) (*params.NodeConfig, error) {
	// NOTE: You might want to say that this function is very complex and should be refactored. And you'll be right.
	//       This is a copy of code in GethStatusBackend.loginAccount. I left it as is not to break anything.
	//       Some of the internal calls were decapsulated: loadNodeConfig, UpdateNodeConfigFleet.
	//       The whole function should be removed altogether with NodeConfig itself.

	defaultCfg := &params.NodeConfig{
		// why we need this? relate PR: https://github.com/status-im/status-go/pull/4014
		KeycardPairingDataFile: filepath.Join(rootDataDir, api.DefaultKeycardPairingDataFileRelativePath),
	}

	defaultCfg.WalletConfig = api.BuildWalletConfig(&request.RuntimeConfig.WalletConfig, &request.RuntimeConfig.WalletSecrets)

	accountSettings, err := a.GetSettings()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load accountSettings")
	}

	fleet := accountSettings.GetFleet()

	if !params.IsFleetSupported(fleet) {
		fleet = api.DefaultFleet
	}

	err = api.SetFleet(fleet, defaultCfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set fleet")
	}

	persistedNodeConfig, err := nodecfg.GetNodeConfigFromDB(a.appDB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get persisted node config")
	}
	if persistedNodeConfig == nil {
		return nil, errors.New("persisted node config is nil")
	}

	// If an installationID is provided, we override it
	if persistedNodeConfig.ShhextConfig.InstallationID != "" {
		defaultCfg.ShhextConfig.InstallationID = persistedNodeConfig.ShhextConfig.InstallationID
	}

	if err := mergo.Merge(persistedNodeConfig, defaultCfg, mergo.WithOverride); err != nil {
		return nil, err
	}

	persistedNodeConfig.Networks = defaultCfg.Networks

	err = nodecfg.SaveNodeConfig(a.appDB, persistedNodeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save node config")
	}

	persistedNodeConfig.RootDataDir = rootDataDir

	out := persistedNodeConfig

	if request.RuntimeOverrides.LogLevel != nil {
		out.LogLevel = *request.RuntimeOverrides.LogLevel
	}
	if request.RuntimeOverrides.WakuV2Nameserver != nil {
		out.WakuV2Config.Nameserver = *request.RuntimeOverrides.WakuV2Nameserver
	}
	if request.RuntimeOverrides.BandwidthStatsEnabled != nil {
		out.ShhextConfig.BandwidthStatsEnabled = *request.RuntimeOverrides.BandwidthStatsEnabled
	}

	out.Networks = api.BuildDefaultNetworks(&request.RuntimeConfig.WalletSecrets, accountSettings.ThirdpartyServicesEnabled)

	if request.RuntimeConfig.APIConfig != nil {
		api.OverrideApiConfig(out, request.RuntimeConfig.APIConfig)
	}

	return out, nil
}
