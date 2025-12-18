package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/internal/crypto/types"
	persistence "github.com/status-im/status-go/services/connector/database"
)

func TestFailToChangeAccountWithMissingFields(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	cmd := NewChangeAccountCommand(db)

	args := ChangeAccountArgs{
		URL:      "",
		ClientID: "testClientID",
		Account:  types.HexToAddress("0x1234567890123456789012345678901234567890"),
	}

	err := cmd.Execute(args)
	assert.Equal(t, ErrEmptyRPCParams, err)

	args = ChangeAccountArgs{
		URL:      "http://example.com",
		ClientID: "",
		Account:  types.HexToAddress("0x1234567890123456789012345678901234567890"),
	}
	err = cmd.Execute(args)
	assert.Equal(t, ErrEmptyRPCParams, err)

	// Test zero address
	args = ChangeAccountArgs{
		URL:      "http://example.com",
		ClientID: "testClientID",
		Account:  types.ZeroAddress(),
	}
	err = cmd.Execute(args)
	assert.Equal(t, ErrEmptyRPCParams, err)
}

func TestChangeAccountForUnpermittedDApp(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	cmd := NewChangeAccountCommand(db)

	args := ChangeAccountArgs{
		URL:      "http://nonexistentDAppURL",
		ClientID: "testClientID",
		Account:  types.HexToAddress("0x1234567890123456789012345678901234567890"),
	}

	err := cmd.Execute(args)
	assert.NoError(t, err)
}

func TestChangeAccountForPermittedDApp(t *testing.T) {
	db, cleanup := createWalletDB(t)
	t.Cleanup(cleanup)

	cmd := NewChangeAccountCommand(db)

	sharedAccount := types.HexToAddress("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg")
	clientID := "testClientID"

	dApp := testDAppData
	dApp.ClientID = clientID
	err := PersistDAppData(db, dApp, sharedAccount, 0x123)
	assert.NoError(t, err)

	newAccount := types.HexToAddress("0x2222222222222222222222222222222222222222")
	args := ChangeAccountArgs{
		URL:      dApp.URL,
		ClientID: clientID,
		Account:  newAccount,
	}

	err = cmd.Execute(args)
	assert.NoError(t, err)

	updatedDApp, err := persistence.SelectDApp(db, dApp.URL, clientID)
	assert.NoError(t, err)
	assert.Equal(t, newAccount, updatedDApp.SharedAccount)
}
