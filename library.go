package main

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/node"
)

/*
#include <stdio.h>
int doNewAccount();
*/
import "C"

var (
	scryptN = 262144
	scryptP = 1
)

func main() {
	Example()
}

//export NewAccount
func NewAccount(p, k *C.char) *C.char {

	password := C.GoString(p)
	keydir := C.GoString(k)

	var sync *[]node.Service
	w := true
	accman := accounts.NewManager(keydir, scryptN, scryptP, sync)

	account, err := accman.NewAccount(password, w)
	if err != nil {
		log.Fatal(err)
	}

	address := fmt.Sprintf("{%x}", account.Address)
	return C.CString(address)

}

func Example() {
	C.doNewAccount()
}
