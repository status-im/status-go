package keygen

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v2"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

// Command generates a key file used to generate the node's peerID, encrypted with an optional password
var Command = cli.Command{
	Name:  "generate-key",
	Usage: "Generate private key file at path specified in --key-file with the password defined by --key-password",
	Action: func(cCtx *cli.Context) error {
		if err := generateKeyFile(Options.KeyFile, []byte(Options.KeyPasswd), Options.Overwrite); err != nil {
			utils.Logger().Fatal("could not write keyfile", zap.Error(err))
		}
		return nil
	},
	Flags: []cli.Flag{
		KeyFile,
		KeyPassword,
		Overwrite,
	},
}

func checkForFileExistence(path string, overwrite bool) error {
	_, err := os.Stat(path)

	if err == nil && !overwrite {
		return fmt.Errorf("%s already exists. Use --overwrite to overwrite the file", path)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func generatePrivateKey() ([]byte, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return key.D.Bytes(), nil
}

func writeKeyFile(path string, key []byte, passwd []byte) error {
	encryptedK, err := keystore.EncryptDataV3(key, passwd, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		return err
	}

	output, err := json.Marshal(encryptedK)
	if err != nil {
		return err
	}

	return os.WriteFile(path, output, 0600)
}

func generateKeyFile(path string, passwd []byte, overwrite bool) error {
	if err := checkForFileExistence(path, overwrite); err != nil {
		return err
	}

	key, err := generatePrivateKey()
	if err != nil {
		return err
	}

	return writeKeyFile(path, key, passwd)
}
