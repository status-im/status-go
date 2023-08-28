package keystore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

const RLN_CREDENTIALS_FILENAME = "rlnKeystore.json"
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
	IdentityCredential *rln.IdentityCredential `json:"identityCredential"`
	MembershipGroups   []MembershipGroup       `json:"membershipGroups"`
}

type AppInfo struct {
	Application   string `json:"application"`
	AppIdentifier string `json:"appIdentifier"`
	Version       string `json:"version"`
}

type AppKeystore struct {
	Application   string                  `json:"application"`
	AppIdentifier string                  `json:"appIdentifier"`
	Credentials   []AppKeystoreCredential `json:"credentials"`
	Version       string                  `json:"version"`
}

type AppKeystoreCredential struct {
	Crypto keystore.CryptoJSON `json:"crypto"`
}

const DefaultSeparator = "\n"

func (m MembershipCredentials) Equals(other MembershipCredentials) bool {
	if !rln.IdentityCredentialEquals(*m.IdentityCredential, *other.IdentityCredential) {
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
		credentialsBytes, err := keystore.DecryptDataV3(credential.Crypto, password)
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

// AddMembershipCredentials inserts a membership credential to the keystore matching the application, appIdentifier and version filters.
func AddMembershipCredentials(path string, newIdentityCredential *rln.IdentityCredential, newMembershipGroup MembershipGroup, password string, appInfo AppInfo, separator string) (membershipGroupIndex uint, err error) {
	k, err := LoadAppKeystore(path, appInfo, DefaultSeparator)
	if err != nil {
		return 0, err
	}

	// A flag to tell us if the keystore contains a credential associated to the input identity credential, i.e. membershipCredential
	found := false
	for i, existingCredentials := range k.Credentials {
		credentialsBytes, err := keystore.DecryptDataV3(existingCredentials.Crypto, password)
		if err != nil {
			continue
		}

		var credentials MembershipCredentials
		err = json.Unmarshal(credentialsBytes, &credentials)
		if err != nil {
			continue
		}

		if rln.IdentityCredentialEquals(*credentials.IdentityCredential, *newIdentityCredential) {
			// idCredential is present in keystore. We add the input credential membership group to the one contained in the decrypted keystore credential (we deduplicate groups using sets)
			allMembershipsMap := make(map[MembershipGroup]struct{})
			for _, m := range credentials.MembershipGroups {
				allMembershipsMap[m] = struct{}{}
			}
			allMembershipsMap[newMembershipGroup] = struct{}{}

			// We sort membership groups, otherwise we will not have deterministic results in tests
			var allMemberships []MembershipGroup
			for k := range allMembershipsMap {
				allMemberships = append(allMemberships, k)
			}
			sort.Slice(allMemberships, func(i, j int) bool {
				return allMemberships[i].MembershipContract.Address < allMemberships[j].MembershipContract.Address
			})

			// we define the updated credential with the updated membership sets
			updatedCredential := MembershipCredentials{
				IdentityCredential: newIdentityCredential,
				MembershipGroups:   allMemberships,
			}

			// we re-encrypt creating a new keyfile
			b, err := json.Marshal(updatedCredential)
			if err != nil {
				return 0, err
			}

			encryptedCredentials, err := keystore.EncryptDataV3(b, []byte(password), keystore.StandardScryptN, keystore.StandardScryptP)
			if err != nil {
				return 0, err
			}

			// we update the original credential field in keystoreCredentials
			k.Credentials[i] = AppKeystoreCredential{Crypto: encryptedCredentials}

			found = true

			// We setup the return values
			membershipGroupIndex = uint(len(allMemberships))
			for mIdx, mg := range updatedCredential.MembershipGroups {
				if mg.MembershipContract.Equals(newMembershipGroup.MembershipContract) {
					membershipGroupIndex = uint(mIdx)
					break
				}
			}

			// We stop decrypting other credentials in the keystore
			break
		}
	}

	if !found { // Not found
		newCredential := MembershipCredentials{
			IdentityCredential: newIdentityCredential,
			MembershipGroups:   []MembershipGroup{newMembershipGroup},
		}

		b, err := json.Marshal(newCredential)
		if err != nil {
			return 0, err
		}

		encryptedCredentials, err := keystore.EncryptDataV3(b, []byte(password), keystore.StandardScryptN, keystore.StandardScryptP)
		if err != nil {
			return 0, err
		}

		k.Credentials = append(k.Credentials, AppKeystoreCredential{Crypto: encryptedCredentials})

		membershipGroupIndex = uint(len(newCredential.MembershipGroups) - 1)
	}

	return membershipGroupIndex, save(k, path, separator)
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
