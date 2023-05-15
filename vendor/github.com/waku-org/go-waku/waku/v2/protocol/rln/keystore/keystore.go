package keystore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

const RLN_CREDENTIALS_FILENAME = "rlnCredentials.json"
const RLN_CREDENTIALS_PASSWORD = "password"

type MembershipContract struct {
	ChainId string `json:"chainId"`
	Address string `json:"address"`
}

type MembershipGroup struct {
	MembershipContract MembershipContract  `json:"membershipContract"`
	TreeIndex          rln.MembershipIndex `json:"treeIndex"`
}

type MembershipCredentials struct {
	IdentityCredential rln.IdentityCredential `json:"identityCredential"`
	MembershipGroups   []MembershipGroup      `json:"membershipGroups"`
}

type AppInfo struct {
	Application   string `json:"application"`
	AppIdentifier string `json:"appIdentifier"`
	Version       string `json:"version"`
}

type AppKeystore struct {
	Application   string                `json:"application"`
	AppIdentifier string                `json:"appIdentifier"`
	Credentials   []keystore.CryptoJSON `json:"credentials"`
	Version       string                `json:"version"`
}

const DefaultSeparator = "\n"

func (m MembershipCredentials) Equals(other MembershipCredentials) bool {
	if !rln.IdentityCredentialEquals(m.IdentityCredential, other.IdentityCredential) {
		return false
	}

	for _, x := range m.MembershipGroups {
		found := false
		for _, y := range other.MembershipGroups {
			if x.Equals(y) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (m MembershipGroup) Equals(other MembershipGroup) bool {
	return m.MembershipContract.Equals(other.MembershipContract) && m.TreeIndex == other.TreeIndex
}

func (m MembershipContract) Equals(other MembershipContract) bool {
	return m.Address == other.Address && m.ChainId == other.ChainId
}

func CreateAppKeystore(path string, appInfo AppInfo, separator string) error {
	if separator == "" {
		separator = DefaultSeparator
	}

	keystore := AppKeystore{
		Application:   appInfo.Application,
		AppIdentifier: appInfo.AppIdentifier,
		Version:       appInfo.Version,
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

	return ioutil.WriteFile(path, buffer.Bytes(), 0600)
}

func LoadAppKeystore(path string, appInfo AppInfo, separator string) (AppKeystore, error) {
	if separator == "" {
		separator = DefaultSeparator
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// If no keystore exists at path we create a new empty one with passed keystore parameters
			err = CreateAppKeystore(path, appInfo, separator)
			if err != nil {
				return AppKeystore{}, err
			}
		} else {
			return AppKeystore{}, err
		}
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return AppKeystore{}, err
	}

	for _, keystoreBytes := range bytes.Split(src, []byte(separator)) {
		if len(keystoreBytes) == 0 {
			continue
		}

		keystore := AppKeystore{}
		err := json.Unmarshal(keystoreBytes, &keystore)
		if err != nil {
			continue
		}

		if keystore.AppIdentifier == appInfo.AppIdentifier && keystore.Application == appInfo.Application && keystore.Version == appInfo.Version {
			return keystore, nil
		}
	}

	return AppKeystore{}, errors.New("no keystore found")
}

func filterCredential(credential MembershipCredentials, filterIdentityCredentials []MembershipCredentials, filterMembershipContracts []MembershipContract) *MembershipCredentials {
	if len(filterIdentityCredentials) != 0 {
		found := false
		for _, filterCreds := range filterIdentityCredentials {
			if filterCreds.Equals(credential) {
				found = true
			}
		}
		if !found {
			return nil
		}
	}

	if len(filterMembershipContracts) != 0 {
		var membershipGroupsIntersection []MembershipGroup
		for _, filterContract := range filterMembershipContracts {
			for _, credentialGroups := range credential.MembershipGroups {
				if filterContract.Equals(credentialGroups.MembershipContract) {
					membershipGroupsIntersection = append(membershipGroupsIntersection, credentialGroups)
				}
			}
		}
		if len(membershipGroupsIntersection) != 0 {
			// If we have a match on some groups, we return the credential with filtered groups
			return &MembershipCredentials{
				IdentityCredential: credential.IdentityCredential,
				MembershipGroups:   membershipGroupsIntersection,
			}
		} else {
			return nil
		}
	}

	// We hit this return only if
	// - filterIdentityCredentials.len() == 0 and filterMembershipContracts.len() == 0 (no filter)
	// - filterIdentityCredentials.len() != 0 and filterMembershipContracts.len() == 0 (filter only on identity credential)
	// Indeed, filterMembershipContracts.len() != 0 will have its exclusive return based on all values of membershipGroupsIntersection.len()
	return &credential
}

func GetMembershipCredentials(logger *zap.Logger, credentialsPath string, password string, appInfo AppInfo, filterIdentityCredentials []MembershipCredentials, filterMembershipContracts []MembershipContract) ([]MembershipCredentials, error) {
	k, err := LoadAppKeystore(credentialsPath, appInfo, DefaultSeparator)
	if err != nil {
		return nil, err
	}

	var result []MembershipCredentials

	for _, credential := range k.Credentials {
		credentialsBytes, err := keystore.DecryptDataV3(credential, password)
		if err != nil {
			return nil, err
		}

		var credentials MembershipCredentials
		err = json.Unmarshal(credentialsBytes, &credentials)
		if err != nil {
			return nil, err
		}

		filteredCredential := filterCredential(credentials, filterIdentityCredentials, filterMembershipContracts)
		if filteredCredential != nil {
			result = append(result, *filteredCredential)
		}
	}

	return result, nil
}

// Adds a sequence of membership credential to the keystore matching the application, appIdentifier and version filters.
func AddMembershipCredentials(path string, credentials []MembershipCredentials, password string, appInfo AppInfo, separator string) error {
	k, err := LoadAppKeystore(path, appInfo, DefaultSeparator)
	if err != nil {
		return err
	}

	var credentialsToAdd []MembershipCredentials
	for _, newCredential := range credentials {
		// A flag to tell us if the keystore contains a credential associated to the input identity credential, i.e. membershipCredential
		found := -1
		for i, existingCredentials := range k.Credentials {
			credentialsBytes, err := keystore.DecryptDataV3(existingCredentials, password)
			if err != nil {
				continue
			}

			var credentials MembershipCredentials
			err = json.Unmarshal(credentialsBytes, &credentials)
			if err != nil {
				continue
			}

			if rln.IdentityCredentialEquals(credentials.IdentityCredential, newCredential.IdentityCredential) {
				// idCredential is present in keystore. We add the input credential membership group to the one contained in the decrypted keystore credential (we deduplicate groups using sets)
				allMemberships := append(credentials.MembershipGroups, newCredential.MembershipGroups...)

				// we define the updated credential with the updated membership sets
				updatedCredential := MembershipCredentials{
					IdentityCredential: newCredential.IdentityCredential,
					MembershipGroups:   allMemberships,
				}

				// we re-encrypt creating a new keyfile
				b, err := json.Marshal(updatedCredential)
				if err != nil {
					return err
				}

				encryptedCredentials, err := keystore.EncryptDataV3(b, []byte(password), keystore.StandardScryptN, keystore.StandardScryptP)
				if err != nil {
					return err
				}

				// we update the original credential field in keystoreCredentials
				k.Credentials[i] = encryptedCredentials

				found = i

				// We stop decrypting other credentials in the keystore
				break
			}
		}

		if found == -1 {
			credentialsToAdd = append(credentialsToAdd, newCredential)
		}
	}

	for _, c := range credentialsToAdd {
		b, err := json.Marshal(c)
		if err != nil {
			return err
		}

		encryptedCredentials, err := keystore.EncryptDataV3(b, []byte(password), keystore.StandardScryptN, keystore.StandardScryptP)
		if err != nil {
			return err
		}

		k.Credentials = append(k.Credentials, encryptedCredentials)
	}

	return save(k, path, separator)
}

// Safely saves a Keystore's JsonNode to disk.
// If exists, the destination file is renamed with extension .bkp; the file is written at its destination and the .bkp file is removed if write is successful, otherwise is restored
func save(keystore AppKeystore, path string, separator string) error {
	// We first backup the current keystore
	_, err := os.Stat(path)
	if err == nil {
		err := os.Rename(path, path+".bkp")
		if err != nil {
			return err
		}
	}

	if separator == "" {
		separator = DefaultSeparator
	}

	b, err := json.Marshal(keystore)
	if err != nil {
		return err
	}

	b = append(b, []byte(separator)...)

	buffer := new(bytes.Buffer)

	err = json.Compact(buffer, b)
	if err != nil {
		restoreErr := os.Rename(path, path+".bkp")
		if restoreErr != nil {
			return fmt.Errorf("could not restore backup file: %w", restoreErr)
		}
		return err
	}

	err = ioutil.WriteFile(path, buffer.Bytes(), 0600)
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
