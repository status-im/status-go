package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/signal"
	persistence "github.com/status-im/status-go/services/connector/database"
)

func TestDisconnectWCSessionDeletesDApp(t *testing.T) {
	db, close := createWalletDB(t)
	t.Cleanup(close)

	sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
	wcDAppData := signal.ConnectorDApp{
		URL:      "https://wc-test-dapp.com",
		Name:     "WC Test DApp",
		IconURL:  "https://wc-test-icon.com",
		ClientID: persistence.WCClientID,
	}

	// Persist WalletConnect DApp
	err := PersistDAppData(db, wcDAppData, sharedAccount, 0x1)
	assert.NoError(t, err)

	// Insert a WC session
	topic := "test-topic-123"
	err = persistence.UpsertWCSession(db, topic, `{"session":"data"}`, 9999999999, "pairing1", wcDAppData.URL, "", 100)
	assert.NoError(t, err)

	// Verify DApp and session exist
	dApp, err := persistence.SelectDApp(db, wcDAppData.URL, wcDAppData.ClientID)
	assert.NoError(t, err)
	assert.NotNil(t, dApp)

	session, err := persistence.SelectWCSession(db, topic)
	assert.NoError(t, err)
	assert.NotNil(t, session)

	// Disconnect the WC session
	disconnector := NewWCSessionDisconnector(db, nil)
	err = disconnector.DisconnectSession(context.Background(), topic)
	assert.NoError(t, err)

	// Verify session is deleted
	session, err = persistence.SelectWCSession(db, topic)
	assert.NoError(t, err)
	assert.Nil(t, session)

	// Verify DApp is also deleted
	dApp, err = persistence.SelectDApp(db, wcDAppData.URL, wcDAppData.ClientID)
	assert.NoError(t, err)
	assert.Nil(t, dApp)
}

func TestDisconnectWCSessionIdempotent(t *testing.T) {
	db, close := createWalletDB(t)
	t.Cleanup(close)

	// Try to disconnect a non-existent session
	disconnector := NewWCSessionDisconnector(db, nil)
	err := disconnector.DisconnectSession(context.Background(), "non-existent-topic")

	// Should not error - idempotent operation
	assert.NoError(t, err)
}

func TestDisconnectWCSessionWithMultipleSessions(t *testing.T) {
	db, close := createWalletDB(t)
	t.Cleanup(close)

	sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
	wcDAppData := signal.ConnectorDApp{
		URL:      "https://wc-test-dapp.com",
		Name:     "WC Test DApp",
		IconURL:  "https://wc-test-icon.com",
		ClientID: persistence.WCClientID,
	}

	// Persist WalletConnect DApp
	err := PersistDAppData(db, wcDAppData, sharedAccount, 0x1)
	assert.NoError(t, err)

	// Insert 2 WC sessions for the same DApp
	err = persistence.UpsertWCSession(db, "topic1", `{"session":"data1"}`, 9999999999, "pairing1", wcDAppData.URL, "", 100)
	assert.NoError(t, err)
	err = persistence.UpsertWCSession(db, "topic2", `{"session":"data2"}`, 9999999999, "pairing2", wcDAppData.URL, "", 200)
	assert.NoError(t, err)

	// Disconnect first session
	disconnector := NewWCSessionDisconnector(db, nil)
	err = disconnector.DisconnectSession(context.Background(), "topic1")
	assert.NoError(t, err)

	// Verify first session is deleted
	session1, err := persistence.SelectWCSession(db, "topic1")
	assert.NoError(t, err)
	assert.Nil(t, session1)

	// Verify second session still exists
	session2, err := persistence.SelectWCSession(db, "topic2")
	assert.NoError(t, err)
	assert.NotNil(t, session2)

	// DApp entry should still exist (other session remains)
	dApp, err := persistence.SelectDApp(db, wcDAppData.URL, wcDAppData.ClientID)
	assert.NoError(t, err)
	assert.NotNil(t, dApp, "DApp entry should be preserved when other sessions exist")

	// Now disconnect the second session
	err = disconnector.DisconnectSession(context.Background(), "topic2")
	assert.NoError(t, err)

	// Verify second session is deleted
	session2, err = persistence.SelectWCSession(db, "topic2")
	assert.NoError(t, err)
	assert.Nil(t, session2)

	// DApp entry should now be deleted (no sessions remain)
	dApp, err = persistence.SelectDApp(db, wcDAppData.URL, wcDAppData.ClientID)
	assert.NoError(t, err)
	assert.Nil(t, dApp, "DApp entry should be deleted when last session is removed")
}

func TestDisconnectWCSessionHandlesNonWCDApp(t *testing.T) {
	db, close := createWalletDB(t)
	t.Cleanup(close)

	// Create a non-WC DApp (browser connector)
	sharedAccount := types.BytesToAddress(types.FromHex("0x6d0aa2a774b74bb1d36f97700315adf962c69fcg"))
	bcDAppData := signal.ConnectorDApp{
		URL:      "https://bc-test-dapp.com",
		Name:     "BC Test DApp",
		IconURL:  "https://bc-test-icon.com",
		ClientID: "", // Browser connector has empty client ID
	}

	err := PersistDAppData(db, bcDAppData, sharedAccount, 0x1)
	assert.NoError(t, err)

	// Try to disconnect with a bogus topic (no session exists)
	disconnector := NewWCSessionDisconnector(db, nil)
	err = disconnector.DisconnectSession(context.Background(), "non-existent-topic")

	// Should not error
	assert.NoError(t, err)

	// Browser connector DApp should still exist (wasn't touched)
	dApp, err := persistence.SelectDApp(db, bcDAppData.URL, bcDAppData.ClientID)
	assert.NoError(t, err)
	assert.NotNil(t, dApp)
}
