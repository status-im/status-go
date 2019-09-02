package mailservers

import (
	"context"
	"github.com/status-im/status-go/appdatabase"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func setupTestDB(t *testing.T) (*Database, func()) {
	tmpfile, err := ioutil.TempFile("", "mailservers-service")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "mailservers-tests")
	require.NoError(t, err)
	return NewDB(db), func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestAddAndGetMailserver(t *testing.T) {
	db, close := setupTestDB(t)
	defer close()
	api := &API{db: db}
	testMailserver := Mailserver{
		ID:       "abc",
		Name:     "My Mailserver",
		Address:  "enode://...",
		Password: nil,
		Fleet:    "beta",
	}

	err := api.AddMailserver(context.Background(), testMailserver)
	require.NoError(t, err)

	mailservers, err := api.GetMailservers(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, []Mailserver{testMailserver}, mailservers)
}
