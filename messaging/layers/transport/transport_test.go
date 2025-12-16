package transport

import (
	"testing"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/messaging/layers/transport/migrations"
	"github.com/status-im/status-go/t/helpers"
)

func TestNewTransport(t *testing.T) {
	db, err := helpers.SetupTestMemorySQLDB(helpers.NewTestDBInitializer([]*bindata.AssetSource{
		{
			Names:     migrations.AssetNames(),
			AssetFunc: migrations.Asset,
		},
	}))
	require.NoError(t, err)

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	_, err = NewTransport(nil, nil, NewSQLiteKeysPersistence(db), NewSQLiteProcessedMessageIDsCachePersistence(db), nil, logger)
	require.NoError(t, err)
}
