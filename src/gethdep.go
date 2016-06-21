package main

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	errextra "github.com/pkg/errors"
)

var (
	scryptN = 262144
	scryptP = 1
)

// createAccount creates an internal geth account
func createAccount(password, keydir string) (string, string, error) {

	var sync *[]node.Service
	w := true
	accman := accounts.NewManager(keydir, scryptN, scryptP, sync)

	account, err := accman.NewAccount(password, w)
	if err != nil {
		return "", "", errextra.Wrap(err, "Account manager could not create the account")
	}

	address := fmt.Sprintf("{%x}", account.Address)
	key, err := crypto.LoadECDSA(account.File)
	if err != nil {
		return address, "", errextra.Wrap(err, "Could not load the key")
	}
	pubKey := string(crypto.FromECDSAPub(&key.PublicKey))

	return address, pubKey, nil

}

// createAndStartNode creates a node entity and starts the
// node running locally
func createAndStartNode(datadir string) error {

	currentNode := MakeNode(datadir)
	if currentNode != nil {
		StartNode(currentNode)
		return nil
	}

	return errors.New("Could not create the in-memory node object")

}
