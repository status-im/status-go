package main

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/node"
)

var (
	scryptN = 262144
	scryptP = 1
)

func createAccount(password, keydir string) error {

	var sync *[]node.Service
	w := true
	accman := accounts.NewManager(keydir, scryptN, scryptP, sync)

	account, err := accman.NewAccount(password, w)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("{%x}", account.Address)
	fmt.Println(address)
	return nil

}

func createAndStartNode(datadir string) error {

	currentNode := MakeNode(datadir)
	if currentNode != nil {
		StartNode(currentNode)
		return nil
	}

	return errors.New("Could not create the in-memory node object")

}
