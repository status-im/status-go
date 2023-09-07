package keystore

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/waku-org/go-waku/waku/v2/hash"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

// DefaultCredentialsFilename is the suggested default filename for the rln credentials keystore
const DefaultCredentialsFilename = "./rlnKeystore.json"

// DefaultCredentialsPassword is the suggested default password for the rln credentials store
const DefaultCredentialsPassword = "password"

// New creates a new instance of a rln credentials keystore
func New(path string, appInfo AppInfo, logger *zap.Logger) (*AppKeystore, error) {
	logger = logger.Named("rln-keystore")

	if path == "" {
		logger.Warn("keystore: no credentials path set, using default path", zap.String("path", DefaultCredentialsFilename))
		path = DefaultCredentialsFilename
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If no keystore exists at path we create a new empty one with passed keystore parameters
			err = createAppKeystore(path, appInfo, defaultSeparator)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	for _, keystoreBytes := range bytes.Split(src, []byte(defaultSeparator)) {
		if len(keystoreBytes) == 0 {
			continue
		}

		keystore := new(AppKeystore)
		err := json.Unmarshal(keystoreBytes, keystore)
		if err != nil {
			continue
		}

		keystore.logger = logger
		keystore.path = path

		if keystore.Credentials == nil {
			keystore.Credentials = map[Key]appKeystoreCredential{}
		}

		if keystore.AppIdentifier == appInfo.AppIdentifier && keystore.Application == appInfo.Application && keystore.Version == appInfo.Version {
			return keystore, nil
		}
	}

	return nil, errors.New("no keystore found")
}

func getKey(treeIndex rln.MembershipIndex, filterMembershipContract MembershipContractInfo) (Key, error) {
	keyStr := fmt.Sprintf("%s%s%d", filterMembershipContract.ChainID, filterMembershipContract.Address, treeIndex)
	hash := hash.SHA256([]byte(keyStr))
	return Key(strings.ToUpper(hex.EncodeToString(hash))), nil
}

// GetMembershipCredentials decrypts and retrieves membership credentials from the keystore applying filters
func (k *AppKeystore) GetMembershipCredentials(keystorePassword string, treeIndex rln.MembershipIndex, filterMembershipContract MembershipContractInfo) (*MembershipCredentials, error) {
	key, err := getKey(treeIndex, filterMembershipContract)
	if err != nil {
		return nil, err
	}

	credential, ok := k.Credentials[key]
	if !ok {
		return nil, nil
	}

	credentialsBytes, err := keystore.DecryptDataV3(credential.Crypto, keystorePassword)
	if err != nil {
		return nil, err
	}

	credentials := new(MembershipCredentials)
	err = json.Unmarshal(credentialsBytes, credentials)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}

// AddMembershipCredentials inserts a membership credential to the keystore matching the application, appIdentifier and version filters.
func (k *AppKeystore) AddMembershipCredentials(newCredential MembershipCredentials, password string) error {
	credentials, err := k.GetMembershipCredentials(password, newCredential.TreeIndex, newCredential.MembershipContractInfo)
	if err != nil {
		return err
	}

	key, err := getKey(newCredential.TreeIndex, newCredential.MembershipContractInfo)
	if err != nil {
		return err
	}

	if credentials != nil {
		return errors.New("credential already present")
	}

	b, err := json.Marshal(newCredential)
	if err != nil {
		return err
	}

	encryptedCredentials, err := keystore.EncryptDataV3(b, []byte(password), keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		return err
	}

	k.Credentials[key] = appKeystoreCredential{Crypto: encryptedCredentials}

	return save(k, k.path)
}

func createAppKeystore(path string, appInfo AppInfo, separator string) error {
	if separator == "" {
		separator = defaultSeparator
	}

	keystore := AppKeystore{
		Application:   appInfo.Application,
		AppIdentifier: appInfo.AppIdentifier,
		Version:       appInfo.Version,
		Credentials:   make(map[Key]appKeystoreCredential),
	}

	b, err := json.Marshal(keystore)
	if err != nil {
		return err
	}

	b = append(b, []byte(separator)...)

	buffer := new(bytes.Buffer)

	err = json.Compact(buffer, b)
	if err != nil {
		return err
	}

	return os.WriteFile(path, buffer.Bytes(), 0600)
}

// Safely saves a Keystore's JsonNode to disk.
// If exists, the destination file is renamed with extension .bkp; the file is written at its destination and the .bkp file is removed if write is successful, otherwise is restored
func save(keystore *AppKeystore, path string) error {
	// We first backup the current keystore
	_, err := os.Stat(path)
	if err == nil {
		err := os.Rename(path, path+".bkp")
		if err != nil {
			return err
		}
	}

	b, err := json.Marshal(keystore)
	if err != nil {
		return err
	}

	b = append(b, []byte(defaultSeparator)...)

	buffer := new(bytes.Buffer)

	err = json.Compact(buffer, b)
	if err != nil {
		restoreErr := os.Rename(path, path+".bkp")
		if restoreErr != nil {
			return fmt.Errorf("could not restore backup file: %w", restoreErr)
		}
		return err
	}

	err = os.WriteFile(path, buffer.Bytes(), 0600)
	if err != nil {
		restoreErr := os.Rename(path, path+".bkp")
		if restoreErr != nil {
			return fmt.Errorf("could not restore backup file: %w", restoreErr)
		}
		return err
	}

	// The write went fine, so we can remove the backup keystore
	_, err = os.Stat(path + ".bkp")
	if err == nil {
		err := os.Remove(path + ".bkp")
		if err != nil {
			return err
		}
	}

	return nil
}
