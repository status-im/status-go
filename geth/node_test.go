package geth_test

import (
	"github.com/status-im/status-go/geth"
	"testing"
)

const (
	testAddress         = "0x89b50b2b26947ccad43accaef76c21d175ad85f4"
	testAddressPassword = "asdf"
	newAccountPassword  = "badpassword"

	whisperMessage1 = "test message 1 (K1 -> K1)"
	whisperMessage2 = "test message 2 (K1 -> '')"
	whisperMessage3 = "test message 3 ('' -> '')"
	whisperMessage4 = "test message 4 ('' -> K1)"
	whisperMessage5 = "test message 5 (K2 -> K1)"
)

func TestNodeSetup(t *testing.T) {
	err := geth.PrepareTestNode()
	if err != nil {
		t.Error(err)
		return
	}
}
