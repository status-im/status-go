// Imported from github.com/ethereum/go-ethereum/accounts/keystore/passphrase.go
// and github.com/ethereum/go-ethereum/accounts/keystore/presale.go

package geth

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

func reEncryptKey(rawKey []byte, pass string, newPass string) (reEncryptedKey []byte, e error) {
	cryptoJSON, e := RawKeyToCryptoJSON(rawKey)
	if e != nil {
		return reEncryptedKey, fmt.Errorf("convert to crypto json error: %v", e)
	}

	decryptedKey, e := DecryptKey(rawKey, pass)
	if e != nil {
		return reEncryptedKey, fmt.Errorf("decryption error: %v", e)
	}

	if cryptoJSON.KDFParams["n"] == nil || cryptoJSON.KDFParams["p"] == nil {
		return reEncryptedKey, fmt.Errorf("unable to determine `n` or `p`: %v", e)
	}
	n := int(cryptoJSON.KDFParams["n"].(float64))
	p := int(cryptoJSON.KDFParams["p"].(float64))

	gethKey := keystore.Key{
		Id:              decryptedKey.ID,
		Address:         common.Address(decryptedKey.Address),
		PrivateKey:      decryptedKey.PrivateKey,
		ExtendedKey:     decryptedKey.ExtendedKey,
		SubAccountIndex: decryptedKey.SubAccountIndex,
	}

	return keystore.EncryptKey(&gethKey, newPass, n, p)
}

func reEncryptKeyStoreDir(keyDirPath, oldPass, newPass string) error {
	rencryptFileAtPath := func(tempKeyDirPath, path string, fileInfo os.FileInfo) error {
		if fileInfo.IsDir() {
			return nil
		}

		rawKeyFile, e := ioutil.ReadFile(path)
		if e != nil {
			return fmt.Errorf("invalid account key file: %v", e)
		}

		reEncryptedKey, e := reEncryptKey(rawKeyFile, oldPass, newPass)
		if e != nil {
			return fmt.Errorf("unable to re-encrypt key file: %v, path: %s, name: %s", e, path, fileInfo.Name())
		}

		tempWritePath := filepath.Join(tempKeyDirPath, fileInfo.Name())
		e = ioutil.WriteFile(tempWritePath, reEncryptedKey, fileInfo.Mode().Perm())
		if e != nil {
			return fmt.Errorf("unable write key file: %v", e)
		}

		return nil
	}

	keyDirPath = strings.TrimSuffix(keyDirPath, "/")
	keyDirPath = strings.TrimSuffix(keyDirPath, "\\")
	keyParent, keyDirName := filepath.Split(keyDirPath)

	// backupKeyDirName used to store existing keys before final write
	backupKeyDirName := keyDirName + "-backup"
	// tempKeyDirName used to put re-encrypted keys
	tempKeyDirName := keyDirName + "-re-encrypted"
	backupKeyDirPath := filepath.Join(keyParent, backupKeyDirName)
	tempKeyDirPath := filepath.Join(keyParent, tempKeyDirName)

	// create temp key dir
	err := os.MkdirAll(tempKeyDirPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("mkdirall error: %v, tempKeyDirPath: %s", err, tempKeyDirPath)
	}

	err = filepath.Walk(keyDirPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			os.RemoveAll(tempKeyDirPath)
			return fmt.Errorf("walk callback error: %v", err)
		}

		return rencryptFileAtPath(tempKeyDirPath, path, fileInfo)
	})
	if err != nil {
		os.RemoveAll(tempKeyDirPath)
		return fmt.Errorf("walk error: %v", err)
	}

	// move existing keys
	err = os.Rename(keyDirPath, backupKeyDirPath)
	if err != nil {
		os.RemoveAll(tempKeyDirPath)
		return fmt.Errorf("unable to rename keyDirPath to backupKeyDirPath: %v", err)
	}

	// move tempKeyDirPath to keyDirPath
	err = os.Rename(tempKeyDirPath, keyDirPath)
	if err != nil {
		// if this happens, then the app is probably bricked, because the keystore won't exist anymore
		// try to restore from backup
		_ = os.Rename(backupKeyDirPath, keyDirPath)
		return fmt.Errorf("unable to rename tempKeyDirPath to keyDirPath: %v", err)
	}

	// remove temp and backup folders and their contents
	err = os.RemoveAll(tempKeyDirPath)
	if err != nil {
		return fmt.Errorf("unable to delete tempKeyDirPath, manual cleanup required: %v", err)
	}

	err = os.RemoveAll(backupKeyDirPath)
	if err != nil {
		return fmt.Errorf("unable to delete backupKeyDirPath, manual cleanup required: %v", err)
	}

	return nil
}
