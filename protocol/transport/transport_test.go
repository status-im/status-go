package transport

import (
	"testing"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/protocol/tt"
)

func TestNewTransport(t *testing.T) {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	err = sqlite.Migrate(db)
	require.NoError(t, err)

	require.NoError(t, err)

	logger := tt.MustCreateTestLogger()
	require.NoError(t, err)
	defer func() { _ = logger.Sync() }()

	_, err = NewTransport(nil, nil, db, "waku_keys", nil, nil, logger)
	require.NoError(t, err)
}
