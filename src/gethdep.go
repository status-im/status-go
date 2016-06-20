package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli"
)

var (
	scryptN = 262144
	scryptP = 1
)

func createAccount(c *cli.Context) error {

	var sync *[]node.Service
	w := true
	accman := accounts.NewManager(c.String("keydir"), scryptN, scryptP, sync)

	account, err := accman.NewAccount(c.String("password"), w)
	if err != nil {
		return err
	}

	address := fmt.Sprintf("{%x}", account.Address)
	fmt.Println(address)
	return nil

}
