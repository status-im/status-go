package persistence

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/testutils"
)

var testDApp = DApp{
	Name:          "Test DApp",
	URL:           "https://test-dapp-url.com",
	IconURL:       "https://test-dapp-icon-url.com",
	ClientID:      "",
	SharedAccount: types.HexToAddress("0x1234567890"),
	ChainID:       0x1,
}

func setupTestDB(t *testing.T) (db *sql.DB, close func()) {
	db, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
	}
}

func setupTestDAppWithPermission(t *testing.T, db *sql.DB, dApp *DApp, parentCapability string, caveats []Caveat, createdAt int64, expectedCount int) []Permission {
	err := UpsertDApp(db, dApp)
	require.NoError(t, err)

	err = InsertPermission(db, dApp.URL, dApp.ClientID, parentCapability, caveats, createdAt)
	require.NoError(t, err)

	permissions, err := SelectPermissions(db, dApp.URL, dApp.ClientID)
	require.NoError(t, err)
	require.Len(t, permissions, expectedCount)
	return permissions
}

func TestInsertAndSelectDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	dAppBack, err := SelectDApp(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &testDApp, dAppBack)
}

func TestInsertAndUpdateDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	updatedDApp := DApp{
		Name:     "Updated Test DApp",
		URL:      testDApp.URL,
		IconURL:  "https://updated-test-dapp-icon-url.com",
		ClientID: testDApp.ClientID,
	}

	err = UpsertDApp(db, &updatedDApp)
	require.NoError(t, err)

	dAppBack, err := SelectDApp(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &updatedDApp, dAppBack)
	require.NotEqual(t, &testDApp, dAppBack)
}

func TestInsertAndRemoveDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	dAppBack, err := SelectDApp(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &testDApp, dAppBack)

	err = DeleteDApp(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)

	dAppBack, err = SelectDApp(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Empty(t, dAppBack)
}

func TestSelectAllDApps(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	dApps, err := SelectAllDApps(db)
	require.NoError(t, err)
	require.Len(t, dApps, 1)
	require.Equal(t, testDApp, dApps[0])
}

func TestMultipleClientsWithSameURL(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// Create two dApps with same URL but different clientIds
	dApp1 := DApp{
		Name:          "DApp Client 1",
		URL:           "https://same-url.com",
		IconURL:       "https://same-icon.com",
		ClientID:      "client1",
		SharedAccount: types.HexToAddress("0x1111111111"),
		ChainID:       0x1,
	}

	dApp2 := DApp{
		Name:          "DApp Client 2",
		URL:           "https://same-url.com", // Same URL
		IconURL:       "https://same-icon.com",
		ClientID:      "client2",
		SharedAccount: types.HexToAddress("0x2222222222"),
		ChainID:       0x89,
	}

	err := UpsertDApp(db, &dApp1)
	require.NoError(t, err)
	err = UpsertDApp(db, &dApp2)
	require.NoError(t, err)

	retrievedDApp1, err := SelectDApp(db, dApp1.URL, dApp1.ClientID)
	require.NoError(t, err)
	require.Equal(t, &dApp1, retrievedDApp1)

	retrievedDApp2, err := SelectDApp(db, dApp2.URL, dApp2.ClientID)
	require.NoError(t, err)
	require.Equal(t, &dApp2, retrievedDApp2)

	allDApps, err := SelectAllDApps(db)
	require.NoError(t, err)
	require.Len(t, allDApps, 2)

	foundDApp1 := false
	foundDApp2 := false
	for _, dApp := range allDApps {
		if dApp.ClientID == "client1" {
			require.Equal(t, dApp1, dApp)
			foundDApp1 = true
		}
		if dApp.ClientID == "client2" {
			require.Equal(t, dApp2, dApp)
			foundDApp2 = true
		}
	}
	require.True(t, foundDApp1, "client1 dApp not found")
	require.True(t, foundDApp2, "client2 dApp not found")
}

func TestDeleteSpecificClient(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// Create two dApps with same URL but different clientIds
	dApp1 := DApp{
		Name:          "DApp Client 1",
		URL:           "https://test-delete.com",
		IconURL:       "https://test-icon.com",
		ClientID:      "client1",
		SharedAccount: types.HexToAddress("0x1111111111"),
		ChainID:       0x1,
	}

	dApp2 := DApp{
		Name:          "DApp Client 2",
		URL:           "https://test-delete.com",
		IconURL:       "https://test-icon.com",
		ClientID:      "client2",
		SharedAccount: types.HexToAddress("0x2222222222"),
		ChainID:       0x89,
	}

	err := UpsertDApp(db, &dApp1)
	require.NoError(t, err)
	err = UpsertDApp(db, &dApp2)
	require.NoError(t, err)

	// Delete only client1
	err = DeleteDApp(db, dApp1.URL, dApp1.ClientID)
	require.NoError(t, err)

	deletedDApp1, err := SelectDApp(db, dApp1.URL, dApp1.ClientID)
	require.NoError(t, err)
	require.Nil(t, deletedDApp1)

	// Verify client2 still exists
	stillExistsDApp2, err := SelectDApp(db, dApp2.URL, dApp2.ClientID)
	require.NoError(t, err)
	require.Equal(t, &dApp2, stillExistsDApp2)
}

func TestBackwardCompatibilityEmptyClientID(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	oldDApp := DApp{
		Name:          "Old Client DApp",
		URL:           "https://old-client.com",
		IconURL:       "https://old-icon.com",
		ClientID:      "",
		SharedAccount: types.HexToAddress("0x0000000000"),
		ChainID:       0x1,
	}

	newDApp := DApp{
		Name:          "New Client DApp",
		URL:           "https://old-client.com", // Same URL
		IconURL:       "https://old-icon.com",
		ClientID:      "newclient123",
		SharedAccount: types.HexToAddress("0x3333333333"),
		ChainID:       0x89,
	}

	err := UpsertDApp(db, &oldDApp)
	require.NoError(t, err)
	err = UpsertDApp(db, &newDApp)
	require.NoError(t, err)

	retrievedOldDApp, err := SelectDApp(db, oldDApp.URL, "")
	require.NoError(t, err)
	require.Equal(t, &oldDApp, retrievedOldDApp)

	retrievedNewDApp, err := SelectDApp(db, newDApp.URL, newDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &newDApp, retrievedNewDApp)

	// Delete old client (empty clientId)
	err = DeleteDApp(db, oldDApp.URL, "")
	require.NoError(t, err)

	deletedOldDApp, err := SelectDApp(db, oldDApp.URL, "")
	require.NoError(t, err)
	require.Nil(t, deletedOldDApp)

	// Verify new client still exists
	stillExistsNewDApp, err := SelectDApp(db, newDApp.URL, newDApp.ClientID)
	require.NoError(t, err)
	require.Equal(t, &newDApp, stillExistsNewDApp)
}

// EIP-2255 Permissions Tests
func TestInsertAndSelectPermission(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	caveats := []Caveat{
		{Type: "requiredMethods", Value: []interface{}{"signTypedData_v3"}},
	}
	createdAt := int64(1234567890)

	permissions := setupTestDAppWithPermission(t, db, &testDApp, "eth_accounts", caveats, createdAt, 1)

	perm := permissions[0]
	require.Equal(t, testDApp.URL, perm.Invoker)
	require.Equal(t, "eth_accounts", perm.ParentCapability)
	require.Len(t, perm.Caveats, 1)
	require.Equal(t, "requiredMethods", perm.Caveats[0].Type)
}

func TestSelectPermissionsEmpty(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	permissions, err := SelectPermissions(db, "https://nonexistent.com", "")
	require.NoError(t, err)
	require.Empty(t, permissions)
}

func TestMultiplePermissionsForSameDApp(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	caveats1 := []Caveat{{Type: "caveat1", Value: "value1"}}
	caveats2 := []Caveat{{Type: "caveat2", Value: "value2"}}

	err = InsertPermission(db, testDApp.URL, testDApp.ClientID, "eth_accounts", caveats1, 111)
	require.NoError(t, err)

	err = InsertPermission(db, testDApp.URL, testDApp.ClientID, "personal_sign", caveats2, 222)
	require.NoError(t, err)

	permissions, err := SelectPermissions(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Len(t, permissions, 2)

	// Verify both permissions
	foundEthAccounts := false
	foundPersonalSign := false
	for _, perm := range permissions {
		if perm.ParentCapability == "eth_accounts" {
			foundEthAccounts = true
			require.Equal(t, "caveat1", perm.Caveats[0].Type)
		}
		if perm.ParentCapability == "personal_sign" {
			foundPersonalSign = true
			require.Equal(t, "caveat2", perm.Caveats[0].Type)
		}
	}
	require.True(t, foundEthAccounts)
	require.True(t, foundPersonalSign)
}

func TestDeletePermissions(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	caveats := []Caveat{{Type: "test", Value: "value"}}
	setupTestDAppWithPermission(t, db, &testDApp, "eth_accounts", caveats, 123, 1)

	err := DeletePermissions(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)

	// Verify permissions deleted
	permissions, err := SelectPermissions(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Empty(t, permissions)
}

func TestCascadeDeletePermissionsWhenDAppDeleted(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	caveats := []Caveat{{Type: "test", Value: "value"}}
	setupTestDAppWithPermission(t, db, &testDApp, "eth_accounts", caveats, 123, 1)

	// Delete dApp (should cascade to permissions)
	err := DeleteDApp(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)

	// Verify permissions automatically deleted by CASCADE
	permissions, err := SelectPermissions(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Empty(t, permissions)
}

func TestPermissionWithEmptyCaveats(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	emptyCaveats := []Caveat{}
	permissions := setupTestDAppWithPermission(t, db, &testDApp, "eth_accounts", emptyCaveats, 123, 1)
	require.Empty(t, permissions[0].Caveats)
}

func TestInsertDuplicatePermission(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	err := UpsertDApp(db, &testDApp)
	require.NoError(t, err)

	caveats := []Caveat{{Type: "test", Value: "value"}}

	err = InsertPermission(db, testDApp.URL, testDApp.ClientID, "eth_accounts", caveats, 111)
	require.NoError(t, err)

	err = InsertPermission(db, testDApp.URL, testDApp.ClientID, "eth_accounts", caveats, 222)
	require.NoError(t, err)

	err = InsertPermission(db, testDApp.URL, testDApp.ClientID, "eth_accounts", caveats, 333)
	require.NoError(t, err)

	// Should only have ONE permission, not duplicates
	permissions, err := SelectPermissions(db, testDApp.URL, testDApp.ClientID)
	require.NoError(t, err)
	require.Len(t, permissions, 1, "Should not create duplicate permissions")
	require.Equal(t, "eth_accounts", permissions[0].ParentCapability)
}

func TestURLNormalizationTrailingSlash(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// dApp with trailing slash
	dAppWithSlash := DApp{
		Name:          "Test DApp",
		URL:           "https://test-dapp.com/",
		IconURL:       "https://icon.com",
		ClientID:      "client1",
		SharedAccount: types.HexToAddress("0x1234567890"),
		ChainID:       0x1,
	}

	err := UpsertDApp(db, &dAppWithSlash)
	require.NoError(t, err)

	dAppBack, err := SelectDApp(db, "https://test-dapp.com", "client1")
	require.NoError(t, err)
	require.NotNil(t, dAppBack)
	require.Equal(t, "Test DApp", dAppBack.Name)
}

func TestPermissionsWithMultipleClients(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// 2 dApps with different clients
	dApp1 := DApp{
		Name:          "DApp Client 1",
		URL:           "https://test.com",
		IconURL:       "https://icon.com",
		ClientID:      "client1",
		SharedAccount: types.HexToAddress("0x1111111111"),
		ChainID:       0x1,
	}

	dApp2 := DApp{
		Name:          "DApp Client 2",
		URL:           "https://test.com",
		IconURL:       "https://icon.com",
		ClientID:      "client2",
		SharedAccount: types.HexToAddress("0x2222222222"),
		ChainID:       0x1,
	}

	err := UpsertDApp(db, &dApp1)
	require.NoError(t, err)
	err = UpsertDApp(db, &dApp2)
	require.NoError(t, err)

	// Add permissions for both clients
	caveats1 := []Caveat{{Type: "test1", Value: "value1"}}
	caveats2 := []Caveat{{Type: "test2", Value: "value2"}}

	err = InsertPermission(db, dApp1.URL, dApp1.ClientID, "eth_accounts", caveats1, 111)
	require.NoError(t, err)
	err = InsertPermission(db, dApp2.URL, dApp2.ClientID, "personal_sign", caveats2, 222)
	require.NoError(t, err)

	// Verify each client has only their permissions
	perms1, err := SelectPermissions(db, dApp1.URL, dApp1.ClientID)
	require.NoError(t, err)
	require.Len(t, perms1, 1)
	require.Equal(t, "eth_accounts", perms1[0].ParentCapability)

	perms2, err := SelectPermissions(db, dApp2.URL, dApp2.ClientID)
	require.NoError(t, err)
	require.Len(t, perms2, 1)
	require.Equal(t, "personal_sign", perms2[0].ParentCapability)
}

func TestSelectPermissionsReturnsNilForNonExistent(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	permissions, err := SelectPermissions(db, "https://nonexistent.com", "client123")
	require.NoError(t, err)
	require.Nil(t, permissions)
	require.Len(t, permissions, 0)
}

func TestSelectWCSessionsByDAppURL(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	dappURL := "https://test-dapp.com"
	wcDApp := DApp{URL: dappURL, Name: "Test", IconURL: "", ClientID: WCClientID, SharedAccount: types.HexToAddress("0x0"), ChainID: 0x1}
	require.NoError(t, UpsertDApp(db, &wcDApp))

	err := UpsertWCSession(db, "topic1", `{"session":"data1"}`, 9999999999, "pairing1", dappURL, "symkey1", 100)
	require.NoError(t, err)

	err = UpsertWCSession(db, "topic2", `{"session":"data2"}`, 9999999999, "pairing2", dappURL, "symkey2", 200)
	require.NoError(t, err)

	err = UpsertWCSession(db, "topic3", `{"session":"data3"}`, 9999999999, "pairing3", dappURL, "symkey3", 300)
	require.NoError(t, err)

	otherDApp := DApp{URL: "https://other-dapp.com", Name: "Other", IconURL: "", ClientID: WCClientID, SharedAccount: types.HexToAddress("0x0"), ChainID: 0x1}
	require.NoError(t, UpsertDApp(db, &otherDApp))
	err = UpsertWCSession(db, "topic4", `{"session":"data4"}`, 9999999999, "pairing4", "https://other-dapp.com", "symkey4", 400)
	require.NoError(t, err)

	// Test: Select sessions by DApp URL
	sessions, err := SelectWCSessionsByDAppURL(db, dappURL)
	require.NoError(t, err)
	require.Len(t, sessions, 3, "Should return all 3 sessions for the DApp")

	// Verify topics
	topics := make(map[string]bool)
	for _, s := range sessions {
		topics[s.Topic] = true
		require.Equal(t, dappURL, s.DAppURL)
		require.Equal(t, WCClientID, s.ClientID)
	}
	require.True(t, topics["topic1"])
	require.True(t, topics["topic2"])
	require.True(t, topics["topic3"])
	require.False(t, topics["topic4"])
}

func TestSelectWCSessionsByDAppURLNormalization(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	wcDApp := DApp{URL: "https://test-dapp.com", Name: "Test", IconURL: "", ClientID: WCClientID, SharedAccount: types.HexToAddress("0x0"), ChainID: 0x1}
	require.NoError(t, UpsertDApp(db, &wcDApp))

	err := UpsertWCSession(db, "topic1", `{"session":"data1"}`, 9999999999, "pairing1", "https://test-dapp.com/", "symkey", 100)
	require.NoError(t, err)

	// Query without trailing slash (should normalize and find it)
	sessions, err := SelectWCSessionsByDAppURL(db, "https://test-dapp.com")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, "topic1", sessions[0].Topic)
}

func TestSelectWCSessionsByDAppURLReturnsEmpty(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	sessions, err := SelectWCSessionsByDAppURL(db, "https://nonexistent-dapp.com")
	require.NoError(t, err)
	require.Len(t, sessions, 0)
}

func TestUpsertWCSessionWithSymKey(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	dappURL := "https://test-dapp.com"
	wcDApp := DApp{URL: dappURL, Name: "Test", IconURL: "", ClientID: WCClientID, SharedAccount: types.HexToAddress("0x0"), ChainID: 0x1}
	require.NoError(t, UpsertDApp(db, &wcDApp))

	topic := "test-topic"
	sessionJSON := `{"session":"data"}`
	expiry := int64(9999999999)
	pairingTopic := "pairing-topic"
	symKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	createdAt := int64(100)

	// Insert session with sym_key
	err := UpsertWCSession(db, topic, sessionJSON, expiry, pairingTopic, dappURL, symKey, createdAt)
	require.NoError(t, err)

	// Select and verify sym_key is stored
	session, err := SelectWCSession(db, topic)
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, topic, session.Topic)
	require.Equal(t, sessionJSON, session.SessionJSON)
	require.Equal(t, expiry, session.Expiry)
	require.Equal(t, pairingTopic, session.PairingTopic)
	require.Equal(t, dappURL, session.DAppURL)
	require.Equal(t, WCClientID, session.ClientID)
	require.Equal(t, symKey, session.SymKey)
}

func TestDeleteWCSession(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	dappURL := "https://test-dapp.com"
	wcDApp := DApp{URL: dappURL, Name: "Test", IconURL: "", ClientID: WCClientID, SharedAccount: types.HexToAddress("0x0"), ChainID: 0x1}
	require.NoError(t, UpsertDApp(db, &wcDApp))

	topic := "test-topic-delete"
	require.NoError(t, UpsertWCSession(db, topic, `{"session":"data"}`, 9999999999, "pairing1", dappURL, "symkey", 100))

	session, err := SelectWCSession(db, topic)
	require.NoError(t, err)
	require.NotNil(t, session)

	err = DeleteWCSession(db, topic)
	require.NoError(t, err)

	session, err = SelectWCSession(db, topic)
	require.NoError(t, err)
	require.Nil(t, session)
}

// TestDeleteEphemeralDApps verifies that DeleteEphemeralDApps removes only
// rows whose clientID ends with the "#ephemeral" suffix, cascades to their
// permissions via the FK, and leaves all other rows (normal sessions,
// WalletConnect, longer suffixes) untouched.
func TestDeleteEphemeralDApps(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	normal := DApp{
		URL: "https://dapp.com", Name: "Normal", IconURL: "",
		ClientID: "status-desktop/dapp-browser", SharedAccount: types.HexToAddress("0x1111"), ChainID: 0x1,
	}
	ephemeral := DApp{
		URL: "https://dapp.com", Name: "Ephemeral", IconURL: "",
		ClientID: "status-desktop/dapp-browser" + EphemeralClientIDSuffix, SharedAccount: types.HexToAddress("0x2222"), ChainID: 0x1,
	}
	require.True(t, IsEphemeralClientID(ephemeral.ClientID))
	notQuiteEphemeral := DApp{
		URL: "https://almost.com", Name: "Almost", IconURL: "",
		ClientID: "status-desktop/dapp-browser#ephemeral-extra", SharedAccount: types.HexToAddress("0x5555"), ChainID: 0x1,
	}
	wcDApp := DApp{
		URL: "https://wc-dapp.com", Name: "WC", IconURL: "",
		ClientID: WCClientID, SharedAccount: types.HexToAddress("0x3333"), ChainID: 0x1,
	}
	require.NoError(t, UpsertDApp(db, &normal))
	require.NoError(t, UpsertDApp(db, &ephemeral))
	require.NoError(t, UpsertDApp(db, &notQuiteEphemeral))
	require.NoError(t, UpsertDApp(db, &wcDApp))
	require.NoError(t, InsertPermission(db, ephemeral.URL, ephemeral.ClientID, "eth_accounts", []Caveat{}, 1))

	require.NoError(t, DeleteEphemeralDApps(db))

	// Ephemeral row gone.
	got, err := SelectDApp(db, ephemeral.URL, ephemeral.ClientID)
	require.NoError(t, err)
	require.Nil(t, got)

	// Permissions cascaded.
	perms, err := SelectPermissions(db, ephemeral.URL, ephemeral.ClientID)
	require.NoError(t, err)
	require.Empty(t, perms)

	// Normal row intact.
	got, err = SelectDApp(db, normal.URL, normal.ClientID)
	require.NoError(t, err)
	require.NotNil(t, got)

	// WalletConnect row intact.
	got, err = SelectDApp(db, wcDApp.URL, wcDApp.ClientID)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Longer suffix than "#ephemeral" must remain.
	got, err = SelectDApp(db, notQuiteEphemeral.URL, notQuiteEphemeral.ClientID)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestDeleteWCSessionIdempotent(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	// Deleting non-existent session should not error
	err := DeleteWCSession(db, "non-existent-topic")
	require.NoError(t, err)
}

func TestSelectWCSessionReturnsNilForNonExistent(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	session, err := SelectWCSession(db, "non-existent-topic")
	require.NoError(t, err)
	require.Nil(t, session)
}

func TestSelectActiveWCSessionsWithSymKey(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()

	dappURL := "https://test-dapp.com"
	wcDApp := DApp{URL: dappURL, Name: "Test", IconURL: "", ClientID: WCClientID, SharedAccount: types.HexToAddress("0x0"), ChainID: 0x1}
	require.NoError(t, UpsertDApp(db, &wcDApp))

	symKey1 := "key1111111111111111111111111111111111111111111111111111111111111"
	symKey2 := "key2222222222222222222222222222222222222222222222222222222222222"

	// Insert active sessions
	err := UpsertWCSession(db, "topic1", `{"session":"data1"}`, 9999999999, "pairing1", dappURL, symKey1, 100)
	require.NoError(t, err)

	err = UpsertWCSession(db, "topic2", `{"session":"data2"}`, 9999999999, "pairing2", dappURL, symKey2, 200)
	require.NoError(t, err)

	// Insert expired session (should not be returned)
	err = UpsertWCSession(db, "topic3", `{"session":"data3"}`, 100, "pairing3", dappURL, "oldkey", 300)
	require.NoError(t, err)

	// Select active sessions
	validAt := int64(1000)
	sessions, err := SelectActiveWCSessions(db, validAt)
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	// Verify sym_keys are included
	symKeys := make(map[string]string)
	for _, s := range sessions {
		symKeys[s.Topic] = s.SymKey
	}
	require.Equal(t, symKey1, symKeys["topic1"])
	require.Equal(t, symKey2, symKeys["topic2"])
}
