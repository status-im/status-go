package persistence

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/t/helpers"
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
	db, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
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
