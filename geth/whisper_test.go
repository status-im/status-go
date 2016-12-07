package geth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv2"
	"github.com/status-im/status-go/geth"
)

func TestWhisperMessaging(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}

	whisperService, err := geth.NodeManagerInstance().WhisperService()
	if err != nil {
		t.Errorf("whisper service not running: %v", err)
	}

	whisperAPI := whisper.NewPublicWhisperAPI(whisperService)

	// prepare message
	postArgs := whisper.PostArgs{
		From:    "",
		To:      "",
		TTL:     10,
		Topics:  [][]byte{[]byte("test topic")},
		Payload: "test message",
	}

	// create an accounts
	address1, pubKey1, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
		return
	}
	t.Logf("Account created: {address: %s, key: %s}", address1, pubKey1)

	address2, pubKey2, _, err := geth.CreateAccount(newAccountPassword)
	if err != nil {
		fmt.Println(err.Error())
		t.Error("Test failed: could not create account")
		return
	}
	t.Logf("Account created: {address: %s, key: %s}", address2, pubKey2)

	// start watchers
	var receivedMessages = map[string]bool{
		whisperMessage1: false,
		whisperMessage2: false,
		whisperMessage3: false,
		whisperMessage4: false,
		whisperMessage5: false,
	}
	whisperService.Watch(whisper.Filter{
		//From: crypto.ToECDSAPub(common.FromHex(pubKey1)),
		//To:   crypto.ToECDSAPub(common.FromHex(pubKey2)),
		Fn: func(msg *whisper.Message) {
			//t.Logf("Whisper message received: %s", msg.Payload)
			receivedMessages[string(msg.Payload)] = true
		},
	})

	// inject key of newly created account into Whisper, as identity
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1))) {
		t.Error("identity already present in whisper")
	}
	err = geth.SelectAccount(address1, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}
	identitySucceess := whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey1)))
	if !identitySucceess || err != nil {
		t.Errorf("identity not injected into whisper: %v", err)
	}
	if whisperService.HasIdentity(crypto.ToECDSAPub(common.FromHex(pubKey2))) { // ensure that second id is not injected
		t.Error("identity already present in whisper")
	}

	// double selecting (shouldn't be a problem)
	err = geth.SelectAccount(address1, newAccountPassword)
	if err != nil {
		t.Errorf("Test failed: could not select account: %v", err)
		return
	}

	// TEST 0: From != nil && To != nil: encrypted signed message (but we cannot decrypt it - so watchers will not report this)
	postArgs.From = pubKey1
	postArgs.To = pubKey2 // owner of that public key will be able to decrypt it
	postSuccess, err := whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// TEST 1: From != nil && To != nil: encrypted signed message (to self)
	postArgs.From = pubKey1
	postArgs.To = pubKey1
	postArgs.Payload = whisperMessage1
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// send from account that is not in Whisper identity list
	postArgs.From = pubKey2
	postSuccess, err = whisperAPI.Post(postArgs)
	if err == nil || err.Error() != fmt.Sprintf("unknown identity to send from: %s", pubKey2) {
		t.Error("expected error not voiced: we are sending from non-injected whisper identity")
	}

	// TEST 2: From != nil && To == nil: signed broadcast (known sender)
	postArgs.From = pubKey1
	postArgs.To = ""
	postArgs.Payload = whisperMessage2
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// TEST 3: From == nil && To == nil: anonymous broadcast
	postArgs.From = ""
	postArgs.To = ""
	postArgs.Payload = whisperMessage3
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// TEST 4: From == nil && To != nil: encrypted anonymous message
	postArgs.From = ""
	postArgs.To = pubKey1
	postArgs.Payload = whisperMessage4
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	// TEST 5: From != nil && To != nil: encrypted and signed response
	postArgs.From = ""
	postArgs.To = pubKey1
	postArgs.Payload = whisperMessage5
	postSuccess, err = whisperAPI.Post(postArgs)
	if !postSuccess || err != nil {
		t.Errorf("could not post to whisper: %v", err)
	}

	time.Sleep(2 * time.Second) // allow whisper to poll
	for message, status := range receivedMessages {
		if !status {
			t.Errorf("Expected message not received: %s", message)
		}
	}

}
