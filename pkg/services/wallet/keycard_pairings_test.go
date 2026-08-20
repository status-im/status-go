package wallet

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	"github.com/status-im/status-go/internal/db/walletdatabase"
	"github.com/status-im/status-go/internal/rpc"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/pubsub"
)

func TestKeycardPairingsFile(t *testing.T) {
	appDB, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)

	accountsDb, err := accounts.NewDB(appDB)
	require.NoError(t, err)

	db, err := testutils.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	require.NoError(t, err)

	accountsPublisher := pubsub.NewPublisher()

	rpcClient, err := rpc.NewClient(rpc.ClientConfig{
		DB:                db,
		AccountsPublisher: accountsPublisher,
	})
	require.NoError(t, err)

	service, err := NewService(db, accountsDb, appDB, rpcClient, accountsPublisher, nil, nil, &params.NodeConfig{}, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	data, err := service.KeycardPairings().GetPairingsJSONFileContent()
	require.NoError(t, err)
	require.Equal(t, 0, len(data))

	pairingsFile, err := ioutil.TempFile("", "keycard-pairings.json")
	require.NoError(t, err)
	defer pairingsFile.Close()

	service.KeycardPairings().SetKeycardPairingsFile(pairingsFile.Name())

	dataToStore := []byte(`
	{"2b907a26ee4319ab50d7eda44b525f6a":{"key":"cc9d96f9b65b551595f3cf7c531beacda24b4937cece7fef70f5236ee80a0808","index":0},
	"4abcc337a3dfc7e89785c427ef32983b":{"key":"3543288f50b2c0bbb2745ffd7107bc3acd105197b97384342fe864e7391a7af7","index":3},
	"4b2e0fe09f997d7ce20320c971ad54df":{"key":"843edb10045d329f4ecfac73fe66f13deb7b2b685dd54a4b2d2d700d19062391","index":0},
	"7ce8e7456eb9025a97f3579490246cae":{"key":"b12a89ca66288f4239a2b58c2bb533df2694b613eb73fc55b72391497627766f","index":1}}
	`)

	err = service.KeycardPairings().SetPairingsJSONFileContent(dataToStore)
	require.NoError(t, err)

	data, err = service.KeycardPairings().GetPairingsJSONFileContent()
	require.NoError(t, err)
	require.Equal(t, len(dataToStore), len(data))
}

func TestKeycardPairingsSetContentCreatesNestedDirAndRoundTrips(t *testing.T) {
	kp := NewKeycardPairings()
	kp.SetKeycardPairingsFile(filepath.Join(t.TempDir(), "keycard", "pairings.json"))

	content := []byte(`{"2b907a26ee4319ab50d7eda44b525f6a":{"key":"cc9d96f9b65b551595f3cf7c531beacda24b4937cece7fef70f5236ee80a0808","index":0},"4abcc337a3dfc7e89785c427ef32983b":{"key":"3543288f50b2c0bbb2745ffd7107bc3acd105197b97384342fe864e7391a7af7","index":3}}`)
	require.NoError(t, kp.SetPairingsJSONFileContent(content),
		"Expected the write to succeed on a fresh install because SetPairingsJSONFileContent must create the missing parent directory")

	pairings, err := kp.GetPairings()
	require.NoError(t, err,
		"Expected GetPairings to read back what was just written because desktop keycard login depends on this read after sync")
	require.Equal(t, map[string]KeycardPairing{
		"2b907a26ee4319ab50d7eda44b525f6a": {Key: "cc9d96f9b65b551595f3cf7c531beacda24b4937cece7fef70f5236ee80a0808", Index: 0},
		"4abcc337a3dfc7e89785c427ef32983b": {Key: "3543288f50b2c0bbb2745ffd7107bc3acd105197b97384342fe864e7391a7af7", Index: 3},
	}, pairings,
		"Expected the full pairing map back (keys, pairing keys, and indexes) because a length-only comparison would miss corrupted values")
}

func TestKeycardPairingsGetPairingsMissingFileReturnsErrNotExist(t *testing.T) {
	kp := NewKeycardPairings()
	kp.SetKeycardPairingsFile(filepath.Join(t.TempDir(), "pairings.json"))

	pairings, err := kp.GetPairings()
	require.ErrorIs(t, err, os.ErrNotExist,
		"Expected os.ErrNotExist for a missing pairings file because prepareForKeycard distinguishes not-synced-yet from a broken file")
	require.Nil(t, pairings, "Expected no pairings map when the file does not exist")
}

func TestKeycardPairingsSetEmptyContentIsNoOp(t *testing.T) {
	kp := NewKeycardPairings()
	path := filepath.Join(t.TempDir(), "pairings.json")
	kp.SetKeycardPairingsFile(path)

	require.NoError(t, kp.SetPairingsJSONFileContent(nil))
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist,
		"Expected no file to be created for empty content because SetPairingsJSONFileContent treats empty input as nothing-to-write")
}

func TestKeycardPairingsGetPairingsMalformedJSONReturnsError(t *testing.T) {
	kp := NewKeycardPairings()
	path := filepath.Join(t.TempDir(), "pairings.json")
	kp.SetKeycardPairingsFile(path)

	require.NoError(t, kp.SetPairingsJSONFileContent([]byte(`{"broken":`)))
	pairings, err := kp.GetPairings()
	require.Error(t, err,
		"Expected a JSON error for a corrupt pairings file because silently returning no pairings would make keycard login fail with a misleading not-found")
	require.NotErrorIs(t, err, os.ErrNotExist,
		"Expected a parse error, not ErrNotExist, because the file exists but is corrupt")
	require.Nil(t, pairings, "Expected no partial map from a corrupt file")
}
