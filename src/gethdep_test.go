package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestAccountBindings makes sure we can create an account and subsequently
// unlock that account
func TestAccountBindings(t *testing.T) {

	// start geth node and wait for it to initialize
	go createAndStartNode(".ethereumtest")
	time.Sleep(5 * time.Second)
	if currentNode == nil {
		t.Error("Test failed: could not start geth node")
	}

	// create an account
	address, _, err := createAccount("badpassword")
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
	}

	// unlock the created account
	err = unlockAccount(address, "badpassword", 10)
	if err != nil {
		fmt.Println(err)
		t.Error("Test failed: could not unlock account")
	}

	// clean up
	err = os.RemoveAll(".ethereumtest")
	if err != nil {
		t.Error("Test failed: could not clean up temporary datadir")
	}

}
