package transport

import (
	"sync/atomic"
	"testing"
	"time"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/pkg/messaging/layers/transport/migrations"
)

func TestNewTransport(t *testing.T) {
	db, err := testutils.SetupTestMemorySQLDB(testutils.NewTestDBInitializer([]*bindata.AssetSource{
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

func TestCleanFiltersLoopPausesAndResumesByLifecycle(t *testing.T) {
	originalInterval := cleanFiltersLoopInterval
	cleanFiltersLoopInterval = 20 * time.Millisecond
	defer func() { cleanFiltersLoopInterval = originalInterval }()

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	var cleanCalls atomic.Int32
	tr := &Transport{
		logger: logger,
		quit:   make(chan struct{}),
		cleanFiltersFn: func() error {
			cleanCalls.Add(1)
			return nil
		},
	}
	tr.MarkPaused()

	tr.cleanFiltersLoop()
	defer close(tr.quit)

	time.Sleep(100 * time.Millisecond)
	require.Equal(t, int32(0), cleanCalls.Load())

	tr.MarkResumed()
	require.Eventually(t, func() bool {
		return cleanCalls.Load() > 0
	}, time.Second, 20*time.Millisecond)
}
