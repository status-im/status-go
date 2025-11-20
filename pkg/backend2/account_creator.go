package backend2

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	accsmanagement "github.com/status-im/status-go/accounts-management"
	accscommon "github.com/status-im/status-go/accounts-management/common"
	"github.com/status-im/status-go/accounts-management/generator"
	accsmanagementtypes "github.com/status-im/status-go/accounts-management/types"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/common/dbsetup"
	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	multiacccommon "github.com/status-im/status-go/multiaccounts/common"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	requests2 "github.com/status-im/status-go/pkg/backend/requests"
	identityutils "github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/identity/colorhash"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/walletdatabase"
)

const (
	walletAccountDefaultName                  = "Account 1"
	DefaultKeycardPairingDataFileRelativePath = "/keycard/pairings.json"
)

type MediaProvider interface {
	MakeAccountImageURL(keyUid string, imageType string, imageClock uint64) string
}

//type ServiceConfiguration struct {
//	MessengerEnabled bool
//	WalletEnabled    bool
//	NewsFeedEnabled  bool
//	ConnectorEnabled bool
//}
//
//type Persistence interface {
//	//MessengerServiceEnabled() (bool, error)
//	//WalletServiceEnabled() (bool, error)
//	//NewsFeedServiceEnabled() (bool, error)
//
//	ServicesEnabled() (*ServiceConfiguration, error)
//}

type AccountCreator struct {
	rootDataDir string

	multiaccountsDB *multiaccounts.Database // FIXME: Use persistence interface instead
	logger          *zap.Logger
	mediaProvider   MediaProvider

	//accsManager *accsmanagement.AccountsManager
}

func NewAccountCreator(rootDataDir string, logger *zap.Logger, db *multiaccounts.Database, mediaProvider MediaProvider) *AccountCreator {
	//accountsManager, err := accsmanagement.NewAccountsManager(logger.Named("accounts-manager"))
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed to create AccountsManager")
	//}
	//
	//accountsManager.SetRootDataDir(rootDataDir)

	return &AccountCreator{
		rootDataDir:     rootDataDir,
		multiaccountsDB: db,
		logger:          logger,
		mediaProvider:   mediaProvider,
		//accsManager:     accountsManager,
	}
}

func (b *AccountCreator) StartNodeWithChatKeyOrMnemonic(
	ctx context.Context,
	request *requests2.CreateAccount,
	mnemonic string, // empty mnemonic is used for keycard account, not empty for regular account
	keycardData *requests2.KeycardData, // nil for regular account, not nil for account with already set keycard
	restoreAccount bool,
) (*multiaccounts.Account, error) {
	var (
		isKeycard               = request.KeycardInstanceUID != ""
		keyUID                  string
		masterAddress           string
		chatPublicKey           string
		customizationColorClock uint64 // not sure if we need this customizationColorClock at all since the desktop app doesn't use it
		derivedAddresses        = map[string]generator.AccountInfo{
			accscommon.PathWalletRoot:           {},
			accscommon.PathEIP1581Root:          {},
			accscommon.PathEIP1581Chat:          {},
			accscommon.PathDefaultWalletAccount: {},
		}
		keypairToStoreDirectly *accsmanagementtypes.Keypair
	)

	if keycardData != nil { // means that the keycard is already set, details already on it
		keyUID = keycardData.KeyUID
		masterAddress = keycardData.Address

		derivedAddresses[accscommon.PathWalletRoot] = generator.AccountInfo{
			Address: keycardData.WalletRootAddress,
		}
		derivedAddresses[accscommon.PathEIP1581Root] = generator.AccountInfo{
			Address: keycardData.Eip1581Address,
		}
		derivedAddresses[accscommon.PathEIP1581Chat] = generator.AccountInfo{
			Address:    keycardData.WhisperAddress,
			PublicKey:  keycardData.WhisperPublicKey,
			PrivateKey: keycardData.WhisperPrivateKey,
		}
		derivedAddresses[accscommon.PathDefaultWalletAccount] = generator.AccountInfo{
			Address:   keycardData.WalletAddress,
			PublicKey: keycardData.WalletPublicKey,
		}
		derivedAddresses[accscommon.PathEIP1581Encryption] = generator.AccountInfo{
			PublicKey: keycardData.EncryptionPublicKey,
		}
	} else {
		genMasterAcc, err := generator.CreateAccountFromMnemonic(mnemonic, "")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create master account from mnemonic")
		}

		keyUID = genMasterAcc.KeyUID()
		masterAddress = genMasterAcc.Address().Hex()

		if !restoreAccount {
			customizationColorClock = 1
		}

		derivationPaths := []string{
			accscommon.PathWalletRoot,
			accscommon.PathEIP1581Root,
			accscommon.PathEIP1581Chat,
			accscommon.PathDefaultWalletAccount,
			accscommon.PathEIP1581Encryption,
		}
		_, derivedAddresses, err = b.generateDerivedAddresses(genMasterAcc, derivationPaths)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate derived addresses")
		}
	}

	if isKeycard {
		genChatAccount, err := generator.CreateAccountFromPrivateKey(derivedAddresses[accscommon.PathEIP1581Chat].PrivateKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create chat account from private key")
		}

		//chatPrivateKey = genChatAccount.PrivateKey()
		chatPublicKey = genChatAccount.PublicKeyHex()

		request.Password = derivedAddresses[accscommon.PathEIP1581Encryption].PublicKey
	} else {
		chatPublicKey = derivedAddresses[accscommon.PathEIP1581Chat].PublicKey
	}

	accountsCount, err := b.multiaccountsDB.GetAccountsCount()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get accounts count")
	}

	settings, err := prepareSettings(request, mnemonic, keyUID, masterAddress, derivedAddresses, restoreAccount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare settings")
	}

	acc, err := buildAccount(request, keyUID, customizationColorClock, chatPublicKey, accountsCount == 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build account")
	}

	if acc.Name == "" { // Use short form of public key as default account name
		acc.Name = settings.Name
	}

	//nodeConfig, err := b.prepareConfig(request, keyUID, settings.InstallationID)
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed to prepare node config")
	//}

	// Create app database
	appDB, err := b.createAppDatabase(acc.KeyUID, acc.KDFIterations, request.Password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create app database")
	}
	defer func() {
		err := appDB.Close()
		if err != nil {
			b.logger.Error("failed to close app database", zap.Error(err))
		}
	}()

	accdb, err := accounts.NewDB(appDB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create accounts db")
	}

	// Create wallet database
	walletDB, err := b.createWalletDatabase(acc.KeyUID, acc.KDFIterations, request.Password)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create wallet database")
	}
	defer func() {
		err := walletDB.Close()
		if err != nil {
			b.logger.Error("failed to close wallet database", zap.Error(err))
		}
	}()

	// Set accounts management persistence
	accountsManager, err := accsmanagement.NewAccountsManager(b.logger.Named("accounts-manager"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create AccountsManager")
	}

	accountsManager.SetRootDataDir(b.rootDataDir)
	accountsManager.SetPersistence(accdb)
	defer accountsManager.Logout()

	if isKeycard {
		err = b.prepareForKeycard(request, acc, settings)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare for keycard")
		}

		keypairToStoreDirectly, err = b.prepareKeypair(request, keyUID, masterAddress, derivedAddresses, restoreAccount)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare keypair")
		}
	} else {
		walletAccount := &accsmanagementtypes.AccountCreationDetails{
			Path:    accscommon.PathDefaultWalletAccount,
			Name:    walletAccountDefaultName,
			ColorID: request.CustomizationColor,
		}
		_, err := accountsManager.CreateKeypairFromMnemonicAndStore(mnemonic, request.Password,
			request.DisplayName, walletAccount, true, 0)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create wallet account")
		}
	}

	//err = b.StartNodeWithAccountAndInitialConfig(
	//	acc,
	//	request.Password,
	//	*settings,
	//	nodeConfig,
	//	keypairToStoreDirectly,
	//	chatPrivateKey,
	//)

	//err := b.ensureDBsOpened(*acc, password)
	//if err != nil {
	//	return err
	//}

	err = b.saveAccount(*acc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save account")
	}

	err = accdb.CreateSettings(*settings, params.NodeConfig{}) // FIXME: Remove deprecated NodeConfig argument
	if err != nil {
		return nil, errors.Wrap(err, "failed to create settings")
	}

	if keypairToStoreDirectly != nil {
		err = accdb.SaveOrUpdateKeypair(keypairToStoreDirectly)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save keypair")
		}
	}

	//err = b.saveKeypairAndSettings(settings, nodecfg, keypair)
	//if err != nil {
	//	return err
	//}

	//err = b.StartNodeWithAccount(*acc, password, nodecfg, chatKey)
	//if err != nil {
	//	b.logger.Error("start node with account and initial config", zap.Error(err))
	//	return err
	//}

	return acc, nil
}

func (s *AccountCreator) getAppDBPath(keyUID string) (string, error) {
	if len(s.rootDataDir) == 0 {
		return "", errors.New("empty root data dir")
	}

	return filepath.Join(s.rootDataDir, fmt.Sprintf("%s-v4.db", keyUID)), nil
}

func (s *AccountCreator) getWalletDBPath(keyUID string) (string, error) {
	if len(s.rootDataDir) == 0 {
		return "", errors.New("root datadir wasn't provided")
	}

	return filepath.Join(s.rootDataDir, fmt.Sprintf("%s-wallet.db", keyUID)), nil
}

func (s *AccountCreator) createAppDatabase(keyUID string, kdfIterations int, password string) (*sql.DB, error) {
	// WARNING: Decide if we want to drop this migration
	//dbFilePath, err := s.runDBFileMigrations(account, password)
	//if err != nil {
	//	return errors.New("Failed to migrate db file: " + err.Error())
	//}

	dbFilePath, err := s.getAppDBPath(keyUID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database file path")
	}

	db, err := appdatabase.InitializeDB(dbFilePath, password, kdfIterations)
	if err != nil {
		s.logger.Error("failed to initialize db", zap.Error(err))
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

func (s *AccountCreator) createWalletDatabase(keyUID string, kdfIterations int, password string) (*sql.DB, error) {
	dbWalletPath, err := s.getWalletDBPath(keyUID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get wallet database file path")
	}

	db, err := walletdatabase.InitializeDB(dbWalletPath, password, kdfIterations)
	if err != nil {
		s.logger.Error("failed to initialize wallet db", zap.Error(err))
		return nil, errors.Wrap(err, "failed to initialize wallet db")
	}

	return db, nil
}

func (s *AccountCreator) saveAccount(account multiaccounts.Account) error {
	return s.multiaccountsDB.SaveAccount(account)
}

func (s *AccountCreator) generateDerivedAddresses(genAcc *generator.Account, paths []string) (genDerivedAccounts map[string]*generator.Account, genDerivedAccountsInfo map[string]generator.AccountInfo, err error) {
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

func prepareSettings(request *requests2.CreateAccount, mnemonic string, keyUID string, masterAddress string,
	derivedAddresses map[string]generator.AccountInfo, restoreAccount bool) (*settings.Settings, error) {
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
	if !restoreAccount {
		newSettings.Mnemonic = &mnemonic
		newSettings.MnemonicWasNotShown = true
	}

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

func (s *AccountCreator) prepareForKeycard(request *requests2.CreateAccount, multiAccount *multiaccounts.Account, settings *settings.Settings) error {
	if request.KeycardInstanceUID == "" {
		return nil
	}

	if request.KeycardPairingKey != "" {
		// KeycardPairingKey is used only on mobile
		settings.KeycardPairing = request.KeycardPairingKey
		multiAccount.KeycardPairing = request.KeycardPairingKey
	} else {
		// KeycardPairingDataFile is used only on desktop
		keycardPairingDataFile := filepath.Join(s.rootDataDir, DefaultKeycardPairingDataFileRelativePath)
		if request.KeycardPairingDataFile != nil {
			keycardPairingDataFile = *request.KeycardPairingDataFile
		}

		kp := wallet.NewKeycardPairings()
		kp.SetKeycardPairingsFile(keycardPairingDataFile)
		pairings, err := kp.GetPairings()
		if err != nil {
			return errors.Wrap(err, "failed to get keycard pairings")
		}

		keycard, ok := pairings[request.KeycardInstanceUID]
		if !ok {
			return errors.New("keycard not found in pairings file")
		}

		settings.KeycardPairing = keycard.Key
		multiAccount.KeycardPairing = keycard.Key
	}

	settings.KeycardInstanceUID = request.KeycardInstanceUID
	settings.KeycardPairedOn = time.Now().Unix()

	return nil
}

func (s *AccountCreator) prepareKeypair(request *requests2.CreateAccount, keyUID string, masterAddress string,
	derivedAddresses map[string]generator.AccountInfo, restoreAccount bool) (keypair *accsmanagementtypes.Keypair, err error) {
	// set up keypair
	keypair = &accsmanagementtypes.Keypair{
		Name:                    request.DisplayName,
		KeyUID:                  keyUID,
		Type:                    accsmanagementtypes.KeypairTypeProfile,
		DerivedFrom:             masterAddress,
		LastUsedDerivationIndex: 0,
	}

	// add chat account
	chatDerivedAccount := derivedAddresses[accscommon.PathEIP1581Chat]
	keypair.Accounts = append(keypair.Accounts, &accsmanagementtypes.Account{
		PublicKey: types.Hex2Bytes(chatDerivedAccount.PublicKey),
		KeyUID:    keypair.KeyUID,
		Address:   types.HexToAddress(chatDerivedAccount.Address),
		Chat:      true,
		Path:      accscommon.PathEIP1581Chat,
		Position:  -1, // When creating a new account, the chat account should have position -1, cause it doesn't participate
		Operable:  accsmanagementtypes.AccountFullyOperable,
	})

	// add wallet account
	walletDerivedAccount := derivedAddresses[accscommon.PathDefaultWalletAccount]
	keypair.Accounts = append(keypair.Accounts, &accsmanagementtypes.Account{
		PublicKey:          types.Hex2Bytes(walletDerivedAccount.PublicKey),
		KeyUID:             keypair.KeyUID,
		Address:            types.HexToAddress(walletDerivedAccount.Address),
		ColorID:            multiacccommon.CustomizationColor(request.CustomizationColor),
		Wallet:             true,
		Path:               accscommon.PathDefaultWalletAccount,
		Name:               walletAccountDefaultName,
		AddressWasNotShown: !restoreAccount,
		Position:           0, // When creating a new account, the wallet account should have position 0, cause it's the default wallet account
		Operable:           accsmanagementtypes.AccountFullyOperable,
	})

	return
}
