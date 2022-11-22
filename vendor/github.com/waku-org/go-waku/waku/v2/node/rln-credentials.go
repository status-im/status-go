//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/waku-org/go-zerokit-rln/rln"
	"go.uber.org/zap"
)

const RLN_CREDENTIALS_FILENAME = "rlnCredentials.txt"

func WriteRLNMembershipCredentialsToFile(keyPair *rln.MembershipKeyPair, idx rln.MembershipIndex, contractAddress common.Address, path string, passwd []byte) error {
	if path == "" {
		return nil // we dont want to use a credentials file
	}

	if keyPair == nil {
		return nil // no credentials to store
	}

	credentialsJSON, err := json.Marshal(MembershipCredentials{
		Keypair:  keyPair,
		Index:    idx,
		Contract: contractAddress,
	})

	if err != nil {
		return err
	}

	encryptedCredentials, err := keystore.EncryptDataV3(credentialsJSON, passwd, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		return err
	}

	output, err := json.Marshal(encryptedCredentials)
	if err != nil {
		return err
	}

	path = filepath.Join(path, RLN_CREDENTIALS_FILENAME)

	return ioutil.WriteFile(path, output, 0600)
}

func loadMembershipCredentialsFromFile(credentialsFilePath string, passwd string) (MembershipCredentials, error) {
	src, err := ioutil.ReadFile(credentialsFilePath)
	if err != nil {
		return MembershipCredentials{}, err
	}

	var encryptedK keystore.CryptoJSON
	err = json.Unmarshal(src, &encryptedK)
	if err != nil {
		return MembershipCredentials{}, err
	}

	credentialsBytes, err := keystore.DecryptDataV3(encryptedK, passwd)
	if err != nil {
		return MembershipCredentials{}, err
	}

	var credentials MembershipCredentials
	err = json.Unmarshal(credentialsBytes, &credentials)

	return credentials, err
}

func GetMembershipCredentials(logger *zap.Logger, credentialsPath string, password string, membershipContract common.Address, membershipIndex uint) (credentials MembershipCredentials, err error) {
	if credentialsPath == "" { // Not using a file
		return MembershipCredentials{
			Contract: membershipContract,
		}, nil
	}

	credentialsFilePath := filepath.Join(credentialsPath, RLN_CREDENTIALS_FILENAME)
	if _, err = os.Stat(credentialsFilePath); err == nil {
		if credentials, err := loadMembershipCredentialsFromFile(credentialsFilePath, password); err != nil {
			return MembershipCredentials{}, fmt.Errorf("could not read membership credentials file: %w", err)
		} else {
			logger.Info("loaded rln credentials", zap.String("filepath", credentialsFilePath))
			if (bytes.Equal(credentials.Contract.Bytes(), common.Address{}.Bytes())) {
				credentials.Contract = membershipContract
			}
			if (bytes.Equal(membershipContract.Bytes(), common.Address{}.Bytes())) {
				return MembershipCredentials{}, errors.New("no contract address specified")
			}
			return credentials, nil
		}
	}

	if os.IsNotExist(err) {
		return MembershipCredentials{
			Keypair:  nil,
			Index:    membershipIndex,
			Contract: membershipContract,
		}, nil

	}

	return MembershipCredentials{}, fmt.Errorf("could not read membership credentials file: %w", err)
}
