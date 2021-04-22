package transport

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/status-im/status-go/protocol/sqlite"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/tt"
)

func TestNewTransport(t *testing.T) {
	dbPath, err := ioutil.TempFile("", "transport.sql")
	require.NoError(t, err)
	defer os.Remove(dbPath.Name())
	db, err := sqlite.Open(dbPath.Name(), "some-key")
	require.NoError(t, err)

	logger := tt.MustCreateTestLogger()
	require.NoError(t, err)
	defer func() { _ = logger.Sync() }()

	_, err = NewTransport(nil, nil, db, nil, nil, logger)
	require.NoError(t, err)
}
