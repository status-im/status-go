package geth_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth"
)

func TestWhisperFilterRace(t *testing.T) {
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

	// account1
	_, accountKey1, err := geth.AddressToDecryptedAccount(testConfig.Account1.Address, testConfig.Account1.Password)
	if err != nil {
		t.Fatal(err)
	}
	accountKey1Hex := common.ToHex(crypto.FromECDSAPub(&accountKey1.PrivateKey.PublicKey))

	whisperService.AddIdentity(accountKey1.PrivateKey)
	if ok, err := whisperAPI.HasIdentity(accountKey1Hex); err != nil || !ok {
		t.Fatalf("identity not injected: %v", accountKey1Hex)
	}

	// account2
	_, accountKey2, err := geth.AddressToDecryptedAccount(testConfig.Account2.Address, testConfig.Account2.Password)
	if err != nil {
		t.Fatal(err)
	}
	accountKey2Hex := common.ToHex(crypto.FromECDSAPub(&accountKey2.PrivateKey.PublicKey))

	whisperService.AddIdentity(accountKey2.PrivateKey)
	if ok, err := whisperAPI.HasIdentity(accountKey2Hex); err != nil || !ok {
		t.Fatalf("identity not injected: %v", accountKey2Hex)
	}

	// race filter addition
	filterAdded := make(chan struct{})
	allFiltersAdded := make(chan struct{})

	go func() {
		counter := 10
		for range filterAdded {
			counter--
			if counter <= 0 {
				break
			}
		}

		close(allFiltersAdded)
	}()

	for i := 0; i < 10; i++ {
		go func() {
			whisperAPI.NewFilter(whisper.WhisperFilterArgs{
				From: accountKey1Hex,
				To:   accountKey2Hex,
				Topics: []whisper.TopicType{
					{0x4e, 0x03, 0x65, 0x7a}, {0x34, 0x60, 0x7c, 0x9b}, {0x21, 0x41, 0x7d, 0xf9},
				},
			})
			filterAdded <- struct{}{}
		}()
	}

	<-allFiltersAdded
}
