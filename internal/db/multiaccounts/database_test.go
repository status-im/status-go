package multiaccounts

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/dbsetup"
	"github.com/status-im/status-go/internal/db/multiaccounts/common"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/testutils/fake"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	require.NoError(t, err)
	db, err := InitializeDB(tmpfile.Name())
	require.NoError(t, err)
	return db, func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestAccounts(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	expected := Account{Name: "string", KeyUID: "string", CustomizationColor: common.CustomizationColorBlue, ColorHash: ColorHash{{4, 3}, {4, 0}, {4, 3}, {4, 0}}, ColorID: 10, KDFIterations: dbsetup.ReducedKDFIterationsNumber, Timestamp: 1712856359}
	require.NoError(t, db.SaveAccount(expected))
	accounts, err := db.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, expected, accounts[0])
}

func TestAccountsUpdate(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	expected := Account{
		KeyUID:             "string",
		CustomizationColor: common.CustomizationColorBlue,
		ColorHash:          ColorHash{{4, 3}, {4, 0}, {4, 3}, {4, 0}},
		ColorID:            10,
		KDFIterations:      dbsetup.ReducedKDFIterationsNumber,
	}
	require.NoError(t, db.SaveAccount(expected))
	expected.Name = "chars"
	expected.CustomizationColor = common.CustomizationColorMagenta
	expected.HasAcceptedTerms = true
	require.NoError(t, db.UpdateAccount(expected))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, expected, rst[0])
}

func TestUpdateHasAcceptedTerms(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	keyUID := "string"
	expected := Account{
		KeyUID:        keyUID,
		KDFIterations: dbsetup.ReducedKDFIterationsNumber,
	}
	require.NoError(t, db.SaveAccount(expected))
	accounts, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, []Account{expected}, accounts)

	// Update from false -> true
	require.NoError(t, db.UpdateHasAcceptedTerms(keyUID, true))
	account, err := db.GetAccount(keyUID)
	require.NoError(t, err)
	expected.HasAcceptedTerms = true
	require.Equal(t, &expected, account)

	// Update from true -> false
	require.NoError(t, db.UpdateHasAcceptedTerms(keyUID, false))
	account, err = db.GetAccount(keyUID)
	require.NoError(t, err)
	expected.HasAcceptedTerms = false
	require.Equal(t, &expected, account)
}

func TestAccountKeycardPairingRoundTrip(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	expected := Account{
		Name:           "string",
		KeyUID:         keyUID,
		KeycardPairing: "cc9d96f9b65b551595f3cf7c531beacda24b4937cece7fef70f5236ee80a0808",
		KDFIterations:  dbsetup.ReducedKDFIterationsNumber,
	}
	require.NoError(t, db.SaveAccount(expected))

	account, err := db.GetAccount(keyUID)
	require.NoError(t, err)
	require.Equal(t, expected.KeycardPairing, account.KeycardPairing,
		"Expected the saved keycardPairing back because keycard login authenticates against the stored pairing key")

	accounts, err := db.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, expected.KeycardPairing, accounts[0].KeycardPairing,
		"Expected GetAccounts to return the same non-empty keycardPairing as GetAccount")
}

func TestUpdateAccountKeycardPairingSetsAndClears(t *testing.T) {
	const bystanderKeyUID = "0x000000000000000000000000000000000000000000000000000000000000beef"
	const bystanderPairing = "bystander-pairing-must-not-change"

	db, stop := setupTestDB(t)
	defer stop()
	require.NoError(t, db.SaveAccount(Account{KeyUID: keyUID, KDFIterations: dbsetup.ReducedKDFIterationsNumber}))
	// a second account with a pairing of its own, so a write that loses its
	// WHERE clause shows up as a changed bystander rather than passing silently
	require.NoError(t, db.SaveAccount(Account{KeyUID: bystanderKeyUID, KeycardPairing: bystanderPairing, KDFIterations: dbsetup.ReducedKDFIterationsNumber}))

	require.NoError(t, db.UpdateAccountKeycardPairing(keyUID, "843edb10045d329f4ecfac73fe66f13d"))
	account, err := db.GetAccount(keyUID)
	require.NoError(t, err)
	require.Equal(t, "843edb10045d329f4ecfac73fe66f13d", account.KeycardPairing,
		"Expected UpdateAccountKeycardPairing to persist the pairing key because it is the only writer used when a keycard is paired post-creation")

	bystander, err := db.GetAccount(bystanderKeyUID)
	require.NoError(t, err)
	require.Equal(t, bystanderPairing, bystander.KeycardPairing,
		"Expected the other account's pairing to be untouched because the update must apply to the given keyUID only")

	require.NoError(t, db.UpdateAccountKeycardPairing(keyUID, ""))
	account, err = db.GetAccount(keyUID)
	require.NoError(t, err)
	require.Equal(t, "", account.KeycardPairing,
		"Expected clearing the pairing to persist because unpairing must not leave a stale key behind")

	bystander, err = db.GetAccount(bystanderKeyUID)
	require.NoError(t, err)
	require.Equal(t, bystanderPairing, bystander.KeycardPairing,
		"Expected clearing one account's pairing to leave the other account's pairing in place")
}

func TestDatabase_GetAccountsCount(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	count, err := db.GetAccountsCount()
	require.NoError(t, err)
	require.Equal(t, 0, count)

	account := Account{
		KeyUID:        keyUID,
		KDFIterations: dbsetup.ReducedKDFIterationsNumber,
	}
	require.NoError(t, db.SaveAccount(account))

	count, err = db.GetAccountsCount()
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestLoginUpdate(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	accounts := []Account{{Name: "first", KeyUID: "0x1", KDFIterations: dbsetup.ReducedKDFIterationsNumber}, {Name: "second", KeyUID: "0x2", KDFIterations: dbsetup.ReducedKDFIterationsNumber}}
	for _, acc := range accounts {
		require.NoError(t, db.SaveAccount(acc))
	}
	require.NoError(t, db.UpdateAccountTimestamp(accounts[0].KeyUID, 100))
	require.NoError(t, db.UpdateAccountTimestamp(accounts[1].KeyUID, 10))
	accounts[0].Timestamp = 100
	accounts[1].Timestamp = 10
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Equal(t, accounts, rst)
}

// Profile Image tests

var (
	keyUID  = "0xdeadbeef"
	keyUID2 = "0x1337beef"
)

func seedTestDBWithIdentityImages(t *testing.T, db *Database, keyUID string) {
	iis := fake.IdentityImages()
	require.NoError(t, db.StoreIdentityImages(keyUID, iis, false))
}

func TestDatabase_GetIdentityImages(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	seedTestDBWithIdentityImages(t, db, keyUID)

	expected := `[{"keyUid":"0xdeadbeef","type":"large","width":240,"height":300,"fileSize":1024,"resizeTarget":240,"clock":0},{"keyUid":"0xdeadbeef","type":"thumbnail","width":80,"height":80,"fileSize":256,"resizeTarget":80,"clock":0}]`

	oiis, err := db.GetIdentityImages(keyUID)
	require.NoError(t, err)

	joiis, err := json.Marshal(oiis)
	require.NoError(t, err)
	require.Exactly(t, expected, string(joiis))

	oiis, err = db.GetIdentityImages(keyUID2)
	require.NoError(t, err)

	require.Exactly(t, 0, len(oiis))
}

func TestDatabase_GetIdentityImage(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	seedTestDBWithIdentityImages(t, db, keyUID)

	cs := []struct {
		KeyUID   string
		Name     string
		Expected string
	}{
		{
			keyUID,
			images.SmallDimName,
			`{"keyUid":"0xdeadbeef","type":"thumbnail","width":80,"height":80,"fileSize":256,"resizeTarget":80,"clock":0}`,
		},
		{
			keyUID,
			images.LargeDimName,
			`{"keyUid":"0xdeadbeef","type":"large","width":240,"height":300,"fileSize":1024,"resizeTarget":240,"clock":0}`,
		},
		{
			keyUID2,
			images.LargeDimName,
			"null",
		},
	}

	for _, c := range cs {
		oii, err := db.GetIdentityImage(c.KeyUID, c.Name)
		require.NoError(t, err)

		joii, err := json.Marshal(oii)
		require.NoError(t, err)
		require.Exactly(t, c.Expected, string(joii))
	}
}

func TestDatabase_DeleteIdentityImage(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	seedTestDBWithIdentityImages(t, db, keyUID)

	require.NoError(t, db.DeleteIdentityImage(keyUID))

	oii, err := db.GetIdentityImage(keyUID, images.SmallDimName)
	require.NoError(t, err)
	require.Empty(t, oii)
}

func removeAllWhitespace(s string) string {
	tmp := strings.ReplaceAll(s, " ", "")
	tmp = strings.ReplaceAll(tmp, "\n", "")
	tmp = strings.ReplaceAll(tmp, "\t", "")
	return tmp
}

func TestDatabase_GetAccountsWithIdentityImages(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	testAccs := []Account{
		{Name: "string", KeyUID: keyUID, Identicon: "data", HasAcceptedTerms: true},
		{Name: "string", KeyUID: keyUID2},
		{Name: "string", KeyUID: keyUID2 + "2"},
		{Name: "string", KeyUID: keyUID2 + "3"},
	}

	for _, a := range testAccs {
		require.NoError(t, db.SaveAccount(a), a.KeyUID)
	}

	seedTestDBWithIdentityImages(t, db, keyUID)
	seedTestDBWithIdentityImages(t, db, keyUID2+"3")

	err := db.UpdateAccountTimestamp(keyUID, 100)
	require.NoError(t, err)
	err = db.UpdateAccountTimestamp(keyUID2, 10)
	require.NoError(t, err)

	accs, err := db.GetAccounts()
	require.NoError(t, err)

	accJSON, err := json.Marshal(accs)
	require.NoError(t, err)

	expected := `
[
	{
		"name": "string",
		"timestamp": 100,
		"identicon": "data",
		"colorHash": null,
		"colorId": 0,
		"keycard-pairing": "",
		"key-uid": "0xdeadbeef",
		"images": [
			{
				"keyUid": "0xdeadbeef",
				"type": "large",
				"width": 240,
				"height": 300,
				"fileSize": 1024,
				"resizeTarget": 240,
				"clock": 0
			},
			{
				"keyUid": "0xdeadbeef",
				"type": "thumbnail",
				"width": 80,
				"height": 80,
				"fileSize": 256,
				"resizeTarget": 80,
				"clock": 0
			}
		],
		"kdfIterations": 3200,
		"hasAcceptedTerms": true
	},
	{
		"name": "string",
		"timestamp": 10,
		"identicon": "",
		"colorHash": null,
		"colorId": 0,
		"keycard-pairing": "",
		"key-uid": "0x1337beef",
		"images": null,
		"kdfIterations": 3200,
		"hasAcceptedTerms": false
	},
	{
		"name": "string",
		"timestamp": 0,
		"identicon": "",
		"colorHash": null,
		"colorId": 0,
		"keycard-pairing": "",
		"key-uid": "0x1337beef2",
		"images": null,
		"kdfIterations": 3200,
		"hasAcceptedTerms": false
	},
	{
		"name": "string",
		"timestamp": 0,
		"identicon": "",
		"colorHash": null,
		"colorId": 0,
		"keycard-pairing": "",
		"key-uid": "0x1337beef3",
		"images": [
			{
				"keyUid": "0x1337beef3",
				"type": "large",
				"width": 240,
				"height": 300,
				"fileSize": 1024,
				"resizeTarget": 240,
				"clock": 0
			},
			{
				"keyUid": "0x1337beef3",
				"type": "thumbnail",
				"width": 80,
				"height": 80,
				"fileSize": 256,
				"resizeTarget": 80,
				"clock": 0
			}
		],
		"kdfIterations": 3200,
		"hasAcceptedTerms": false
	}
]
`

	require.Exactly(t, removeAllWhitespace(expected), string(accJSON))
}

func TestDatabase_GetAccount(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	expected := Account{
		Name:             "string",
		KeyUID:           keyUID,
		ColorHash:        ColorHash{{4, 3}, {4, 0}, {4, 3}, {4, 0}},
		ColorID:          10,
		KDFIterations:    dbsetup.ReducedKDFIterationsNumber,
		HasAcceptedTerms: true,
	}
	require.NoError(t, db.SaveAccount(expected))

	account, err := db.GetAccount(expected.KeyUID)
	require.NoError(t, err)
	require.Equal(t, &expected, account)
}

func TestDatabase_SaveAccountWithIdentityImages(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	expected := Account{
		Name:      "string",
		KeyUID:    keyUID,
		ColorHash: ColorHash{{4, 3}, {4, 0}, {4, 3}, {4, 0}},
		ColorID:   10,
		Images:    fake.IdentityImages(),
	}
	require.NoError(t, db.SaveAccount(expected))

	account, err := db.GetAccount(expected.KeyUID)
	require.NoError(t, err)
	require.Exactly(t, expected.ColorHash, account.ColorHash)
	require.Exactly(t, expected.ColorID, account.ColorID)
	require.Exactly(t, expected.Identicon, account.Identicon)
	require.Exactly(t, expected.KeycardPairing, account.KeycardPairing)
	require.Exactly(t, expected.KeyUID, account.KeyUID)
	require.Exactly(t, expected.Name, account.Name)
	require.Exactly(t, expected.Timestamp, account.Timestamp)
	require.Len(t, expected.Images, 2)

	matches := 0
	for _, expImg := range expected.Images {
		for _, accImg := range account.Images {
			if expImg.Name != accImg.Name {
				continue
			}
			matches++

			require.Exactly(t, expImg.Clock, accImg.Clock)
			require.Exactly(t, keyUID, accImg.KeyUID)
			require.Exactly(t, expImg.Name, accImg.Name)
			require.Exactly(t, expImg.ResizeTarget, accImg.ResizeTarget)
			require.Exactly(t, expImg.Payload, accImg.Payload)
			require.Exactly(t, expImg.Height, accImg.Height)
			require.Exactly(t, expImg.Width, accImg.Width)
			require.Exactly(t, expImg.FileSize, accImg.FileSize)
		}
	}
	require.Equal(t, 2, matches)
}
