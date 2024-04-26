package pairing

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/requests"

	"gopkg.in/go-playground/validator.v9"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/eth-node/keystore"
)

func newValidate() (*validator.Validate, error) {
	var validate = validator.New()
	var keyUIDPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
	if err := validate.RegisterValidation("keyuid", func(fl validator.FieldLevel) bool {
		return keyUIDPattern.MatchString(fl.Field().String())
	}); err != nil {
		return nil, err
	}

	if err := validate.RegisterValidation("keystorepath", func(fl validator.FieldLevel) bool {
		keyUIDField := fl.Parent()
		if keyUIDField.Kind() == reflect.Ptr {
			keyUIDField = keyUIDField.Elem()
		}

		keyUID := keyUIDField.FieldByName("KeyUID").String()
		return strings.HasSuffix(fl.Field().String(), keyUID)
	}); err != nil {
		return nil, err
	}

	if err := validate.RegisterValidation("not_end_keyuid", func(fl validator.FieldLevel) bool {
		keystorePath := fl.Field().String()
		return len(keystorePath) <= 66 || !keyUIDPattern.MatchString(keystorePath[len(keystorePath)-66:])
	}); err != nil {
		return nil, err
	}
	return validate, nil
}

func validateKeys(keys map[string][]byte, password string) error {
	for _, key := range keys {
		k, err := keystore.DecryptKey(key, password)
		if err != nil {
			return err
		}

		err = generator.ValidateKeystoreExtendedKey(k)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadKeys(keys map[string][]byte, keyStorePath string) error {
	fileWalker := func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if dirEntry.IsDir() || filepath.Dir(path) != keyStorePath {
			return nil
		}

		rawKeyFile, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("invalid account key file: %v", err)
		}

		keys[dirEntry.Name()] = rawKeyFile

		return nil
	}

	err := filepath.WalkDir(keyStorePath, fileWalker)
	if err != nil {
		return fmt.Errorf("cannot traverse key store folder: %v", err)
	}

	return nil
}

func validate(s interface{}) error {
	v, err := newValidate()
	if err != nil {
		return err
	}

	return v.Struct(s)
}

func validateAndVerifyPassword(s interface{}, senderConfig *SenderConfig) error {
	err := validate(s)
	if err != nil {
		return err
	}

	keys := make(map[string][]byte)
	err = loadKeys(keys, senderConfig.KeystorePath)
	if err != nil {
		return err
	}

	return validateKeys(keys, senderConfig.Password)
}

func validateAndVerifyNodeConfig(s interface{}, receiverConfig *ReceiverConfig) error {
	err := validate(s)
	if err != nil {
		return err
	}

	if receiverConfig.NodeConfig == nil {
		return fmt.Errorf("node config is required for receiver config")
	}

	if receiverConfig.NodeConfig.RootDataDir == "" {
		return fmt.Errorf("root data dir is required for node config")
	}

	return receiverConfig.NodeConfig.Validate()
}

func emptyDir(dir string) error {
	// Open the directory
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	// Get all the directory entries
	entries, err := d.Readdir(-1)
	if err != nil {
		return err
	}

	// Remove all the files and directories
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		path := filepath.Join(dir, name)
		if entry.IsDir() {
			err = os.RemoveAll(path)
			if err != nil {
				return err
			}
		} else {
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func validateReceivedKeystoreFiles(expectedKeys []string, keys map[string][]byte, password string) error {
	for _, searchKey := range expectedKeys {
		found := false
		for key := range keys {
			if strings.Contains(key, strings.ToLower(searchKey)) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("one or more expected keystore files are not found among the sent files")
		}
	}

	return validateKeys(keys, password)
}

func validateKeystoreFilesConfig(backend *api.GethStatusBackend, conf interface{}) error {
	var (
		loggedInKeyUID string
		password       string
		numOfKeypairs  int
		keystorePath   string
	)

	switch c := conf.(type) {
	case *KeystoreFilesSenderServerConfig:
		loggedInKeyUID = c.SenderConfig.LoggedInKeyUID
		password = c.SenderConfig.Password
		numOfKeypairs = len(c.SenderConfig.KeypairsToExport)
		keystorePath = c.SenderConfig.KeystorePath
	case *KeystoreFilesReceiverClientConfig:
		loggedInKeyUID = c.ReceiverConfig.LoggedInKeyUID
		password = c.ReceiverConfig.Password
		numOfKeypairs = len(c.ReceiverConfig.KeypairsToImport)
		keystorePath = c.ReceiverConfig.KeystorePath
	default:
		return fmt.Errorf("unknown config type: %v", reflect.TypeOf(conf))
	}

	accountService := backend.StatusNode().AccountService()
	if accountService == nil {
		return fmt.Errorf("cannot resolve accounts service instance")
	}

	if !accountService.GetMessenger().HasPairedDevices() {
		return fmt.Errorf("there are no known paired devices")
	}

	selectedAccount, err := backend.GetActiveAccount()
	if err != nil {
		return err
	}

	if selectedAccount.KeyUID != loggedInKeyUID {
		return fmt.Errorf("configuration is not meant for the logged in account")
	}

	if selectedAccount.KeycardPairing == "" {
		if !accountService.VerifyPassword(password) {
			return fmt.Errorf("provided password is not correct")
		}
	}

	if numOfKeypairs == 0 {
		return fmt.Errorf("it should be at least a single keypair set a keystore files are transferred for")
	}

	if keystorePath == "" {
		return fmt.Errorf("keyStorePath can not be empty")
	}

	return nil
}

// setDefaultNodeConfig sets default values for the node configuration.
// Config Values still needed from the mobile include
// VerifyTransactionURL/VerifyENSURL/VerifyENSContractAddress/VerifyTransactionChainID
// LogEnabled/LogDir/RootDataDir/LightClient/Nameserver
func setDefaultNodeConfig(c *params.NodeConfig) error {
	if c == nil {
		return nil
	}

	err := c.UpdateWithDefaults()
	if err != nil {
		return err
	}

	// following specifiedXXX variables are used to check if frontend has specified the value
	// if not, the default value is set. NOTE: we also check 2 extra fields: WakuV2Config(LightClient|Nameserver)
	// see api.SetFleet for more details
	specifiedVerifyTransactionURL := c.ShhextConfig.VerifyTransactionURL
	specifiedVerifyENSURL := c.ShhextConfig.VerifyENSURL
	specifiedVerifyENSContractAddress := c.ShhextConfig.VerifyENSContractAddress
	specifiedVerifyTransactionChainID := c.ShhextConfig.VerifyTransactionChainID
	specifiedNetworkID := c.NetworkID
	specifiedNetworks := c.Networks
	specifiedUpstreamConfigURL := c.UpstreamConfig.URL
	specifiedLogEnabled := c.LogEnabled
	specifiedLogLevel := c.LogLevel
	specifiedFleet := c.ClusterConfig.Fleet
	specifiedInstallationID := c.ShhextConfig.InstallationID
	specifiedTorrentConfigEnabled := c.TorrentConfig.Enabled
	specifiedTorrentConfigPort := c.TorrentConfig.Port

	if len(specifiedNetworks) == 0 {
		c.Networks = api.BuildDefaultNetworks(&requests.WalletSecretsConfig{})
	}

	if specifiedNetworkID == 0 {
		c.NetworkID = c.Networks[0].ChainID
	}

	c.UpstreamConfig.Enabled = true
	if specifiedUpstreamConfigURL == "" {
		c.UpstreamConfig.URL = c.Networks[0].RPCURL
	}

	if specifiedLogEnabled && specifiedLogLevel == "" {
		c.LogLevel = api.DefaultLogLevel
	}
	c.LogFile = api.DefaultLogFile

	c.Name = api.DefaultNodeName
	c.DataDir = api.DefaultDataDir
	c.KeycardPairingDataFile = api.DefaultKeycardPairingDataFile
	c.Rendezvous = false
	c.NoDiscovery = true
	c.MaxPeers = api.DefaultMaxPeers
	c.MaxPendingPeers = api.DefaultMaxPendingPeers

	c.LocalNotificationsConfig = params.LocalNotificationsConfig{Enabled: true}
	c.BrowsersConfig = params.BrowsersConfig{Enabled: true}
	c.PermissionsConfig = params.PermissionsConfig{Enabled: true}
	c.MailserversConfig = params.MailserversConfig{Enabled: true}

	c.ListenAddr = api.DefaultListenAddr

	if specifiedFleet == "" {
		err = api.SetDefaultFleet(c)
		if err != nil {
			return err
		}
	}

	if specifiedInstallationID == "" {
		specifiedInstallationID = uuid.New().String()
	}

	c.ShhextConfig = params.ShhextConfig{
		BackupDisabledDataDir:      c.RootDataDir,
		InstallationID:             specifiedInstallationID,
		MaxMessageDeliveryAttempts: api.DefaultMaxMessageDeliveryAttempts,
		MailServerConfirmations:    true,
		DataSyncEnabled:            true,
		PFSEnabled:                 true,
		VerifyTransactionURL:       specifiedVerifyTransactionURL,
		VerifyENSURL:               specifiedVerifyENSURL,
		VerifyENSContractAddress:   specifiedVerifyENSContractAddress,
	}
	if specifiedVerifyTransactionChainID == 0 {
		c.ShhextConfig.VerifyTransactionChainID = int64(c.Networks[0].ChainID)
	}

	if specifiedVerifyTransactionURL == "" {
		c.ShhextConfig.VerifyTransactionURL = c.Networks[0].FallbackURL
	}
	if specifiedVerifyENSURL == "" {
		c.ShhextConfig.VerifyENSURL = c.Networks[0].FallbackURL
	}

	c.TorrentConfig = params.TorrentConfig{
		Enabled:    specifiedTorrentConfigEnabled,
		Port:       specifiedTorrentConfigPort,
		DataDir:    filepath.Join(c.RootDataDir, api.DefaultArchivesRelativePath),
		TorrentDir: filepath.Join(c.RootDataDir, api.DefaultTorrentTorrentsRelativePath),
	}

	return nil
}
