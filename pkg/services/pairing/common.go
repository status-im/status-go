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

	"github.com/status-im/status-go/internal/accounts-management/common"
	"github.com/status-im/status-go/internal/accounts-management/generator"
	"github.com/status-im/status-go/internal/accounts-management/keystore"
	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/pkg/backend"

	"gopkg.in/go-playground/validator.v9"
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

	return validate, nil
}

func validateKeys(keys map[string][]byte, password string) error {
	for _, key := range keys {
		k, err := common.DecryptKey(key, password)
		if err != nil {
			return err
		}

		_, err = generator.CreateAccountFromKey(k)
		if err != nil {
			return err
		}
	}

	return nil
}

// rootDataDirFromKeystorePath derives the root data dir from a profile keystore path
// which is either `<root>/keystore/<keyUID>` or `<root>/keystore`
func rootDataDirFromKeystorePath(keystorePath, keyUID string) string {
	keystorePath = strings.TrimRight(keystorePath, "/\\")
	if filepath.Base(keystorePath) == keyUID {
		return filepath.Dir(filepath.Dir(keystorePath))
	}
	return filepath.Dir(keystorePath)
}

// prepareKeysForTransfer re-encrypts loaded keystore files in memory from the profile's DEK to the transfer password
// when the profile uses the DEK encryption scheme. The local-pairing wire format therefore stays password-encrypted,
// compatible with receivers of any version. No-op for legacy profiles.
func prepareKeysForTransfer(keys map[string][]byte, keystorePath, keyUID, password string) error {
	rootDataDir := rootDataDirFromKeystorePath(keystorePath, keyUID)
	if !envelope.Exists(rootDataDir, keyUID) {
		return nil
	}

	dek, _, err := envelope.Unwrap(rootDataDir, keyUID, password)
	if err != nil {
		return err
	}

	for name, rawKey := range keys {
		reEncrypted, err := keystore.ReEncryptRawKey(rawKey, dek, password)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt keystore file %s for transfer: %w", name, err)
		}
		keys[name] = reEncrypted
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

		// skip filesystem metadata files (e.g. .DS_Store created by macOS Finder) and other hidden files
		if strings.HasPrefix(dirEntry.Name(), ".") {
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

func ValidateStruct(s interface{}) error {
	v, err := newValidate()
	if err != nil {
		return err
	}

	return v.Struct(s)
}

func validateAndVerifyPassword(s interface{}, senderConfig *SenderConfig) error {
	err := ValidateStruct(s)
	if err != nil {
		return err
	}

	keys := make(map[string][]byte)
	err = loadKeys(keys, senderConfig.KeystorePath)
	if err != nil {
		return err
	}

	secret := senderConfig.Password
	rootDataDir := rootDataDirFromKeystorePath(senderConfig.KeystorePath, senderConfig.KeyUID)
	if envelope.Exists(rootDataDir, senderConfig.KeyUID) {
		// The KEK (client-hashed password) is required here, credentials like DEK or its client hash should not be used here.
		secret, _, err = envelope.Unwrap(rootDataDir, senderConfig.KeyUID, senderConfig.Password)
		if err != nil {
			return err
		}
	}

	return validateKeys(keys, secret)
}

func validateReceiverConfig(s interface{}, receiverConfig *ReceiverConfig) error {
	err := ValidateStruct(s)
	if err != nil {
		return err
	}

	return receiverConfig.CreateAccount.Validate(&requests.CreateAccountValidation{
		AllowEmptyDisplayName:        true,
		AllowEmptyCustomizationColor: true,
		AllowEmptyPassword:           true,
	})
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

func validateKeystoreFilesConfig(backend *backend.StatusBackend, conf interface{}) error {
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
		if ok, _ := accountService.VerifyPassword(password); !ok {
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
