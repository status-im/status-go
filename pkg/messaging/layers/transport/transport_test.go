package transport

import (
	"sync"
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

	tr, err := NewTransport(nil, nil, NewSQLiteKeysPersistence(db), NewSQLiteProcessedMessageIDsCachePersistence(db), nil, logger)
	require.NoError(t, err)
	// Stop the transport to prevent cleanFiltersLoop from leaking into later tests.
	t.Cleanup(func() { _ = tr.Stop() })
}

func TestCleanFiltersLoopPausesAndResumesByLifecycle(t *testing.T) {
	originalInterval := cleanFiltersLoopInterval
	cleanFiltersLoopInterval = 20 * time.Millisecond
	t.Cleanup(func() { cleanFiltersLoopInterval = originalInterval })

	logger := testutils.MustCreateTestLogger()
	defer func() { _ = logger.Sync() }()

	quit := make(chan struct{})
	var quitOnce sync.Once
	shutdown := func() {
		quitOnce.Do(func() { close(quit) })
	}
	defer shutdown()

	var cleanCalls atomic.Int32
	tr := &Transport{
		logger: logger,
		quit:   quit,
		cleanFiltersFn: func() error {
			cleanCalls.Add(1)
			return nil
		},
	}
	tr.MarkPaused()

	tr.cleanFiltersLoop()

	// While paused the ticker must not run: Never fails if cleanCalls becomes non-zero during the window.
	pausedSoak := 3 * cleanFiltersLoopInterval
	require.Never(t, func() bool { return cleanCalls.Load() > 0 }, pausedSoak, 5*time.Millisecond,
		"ticks must not run while paused")

	tr.MarkResumed()
	require.Eventually(t, func() bool {
		return cleanCalls.Load() > 0
	}, time.Second, 20*time.Millisecond, "expected at least one tick after resume")

	shutdown()
}
