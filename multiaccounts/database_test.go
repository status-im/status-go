package multiaccounts

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/status-im/status-go/images"

	"github.com/stretchr/testify/require"
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
	expected := Account{Name: "string", KeyUID: "string"}
	require.NoError(t, db.SaveAccount(expected))
	accounts, err := db.GetAccounts()
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, expected, accounts[0])
}

func TestAccountsUpdate(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	expected := Account{KeyUID: "string"}
	require.NoError(t, db.SaveAccount(expected))
	expected.Name = "chars"
	require.NoError(t, db.UpdateAccount(expected))
	rst, err := db.GetAccounts()
	require.NoError(t, err)
	require.Len(t, rst, 1)
	require.Equal(t, expected, rst[0])
}

func TestLoginUpdate(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	accounts := []Account{{Name: "first", KeyUID: "0x1"}, {Name: "second", KeyUID: "0x2"}}
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

func seedTestDB(t *testing.T, db *Database) {
	iis := images.SampleIdentityImages()
	require.NoError(t, db.StoreIdentityImages(keyUID, iis))
}

func TestDatabase_GetIdentityImages(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()
	seedTestDB(t, db)

	expected := `[{"keyUid":"0xdeadbeef","type":"large","uri":"data:image/png;base64,iVBORw0KGgoAAAANSUg=","width":240,"height":300,"fileSize":1024,"resizeTarget":240},{"keyUid":"0xdeadbeef","type":"thumbnail","uri":"data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=","width":80,"height":80,"fileSize":256,"resizeTarget":80}]`

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
	seedTestDB(t, db)

	cs := []struct {
		KeyUID   string
		Name     string
		Expected string
	}{
		{
			keyUID,
			images.SmallDimName,
			`{"keyUid":"0xdeadbeef","type":"thumbnail","uri":"data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=","width":80,"height":80,"fileSize":256,"resizeTarget":80}`,
		},
		{
			keyUID,
			images.LargeDimName,
			`{"keyUid":"0xdeadbeef","type":"large","uri":"data:image/png;base64,iVBORw0KGgoAAAANSUg=","width":240,"height":300,"fileSize":1024,"resizeTarget":240}`,
		},
		{
			keyUID2,
			images.LargeDimName,
			`{"keyUid":"","type":"","uri":"","width":0,"height":0,"fileSize":0,"resizeTarget":0}`,
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
	seedTestDB(t, db)

	require.NoError(t, db.DeleteIdentityImage(keyUID))

	oii, err := db.GetIdentityImage(keyUID, images.SmallDimName)
	require.NoError(t, err)
	require.Empty(t, oii)
}
