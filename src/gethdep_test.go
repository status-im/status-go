package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/whisper"
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
	address, pubkey, err := createAccount("badpassword")
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
	}

	// unlock the created account
	err = unlockAccount(address, "badpassword", 3)
	if err != nil {
		fmt.Println(err)
		t.Error("Test failed: could not unlock account")
	}
	time.Sleep(2 * time.Second)

	// test to see if the account was injected in whisper
	whisperInstance := (*accountSync)[0].(*whisper.Whisper)
	identitySucess := whisperInstance.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubkey)))
	if !identitySucess || err != nil {
		t.Error("Test failed: identity not injected into whisper")
	}

	// test to see if we can post with the injected whisper identity
	postArgs := whisper.PostArgs{
		From:    pubkey,
		To:      pubkey,
		TTL:     100,
		Topics:  [][]byte{[]byte("test topic")},
		Payload: "test message",
	}
	whisperAPI := whisper.NewPublicWhisperAPI(whisperInstance)
	postSucess, err := whisperAPI.Post(postArgs)
	if !postSucess || err != nil {
		t.Error("Test failed: Could not post to whisper")
	}

	// clean up
	err = os.RemoveAll(".ethereumtest")
	if err != nil {
		t.Error("Test failed: could not clean up temporary datadir")
	}

}
