package backend

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/internal/signal"
	"github.com/status-im/status-go/internal/testutils"
)

// TestLogoutSignalEpoch checks that only the client-requested logout starts a new
// signal epoch, that the old session's node.stopped keeps the old epoch, and that
// the internal logout used by re-encryption and pairing keeps it unchanged.
func TestLogoutSignalEpoch(t *testing.T) {
	var mu sync.Mutex
	epochsByType := map[string][]uint64{}
	signal.SetHandler(func(data []byte) {
		var envelope signal.Envelope
		if json.Unmarshal(data, &envelope) != nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		epochsByType[envelope.Type] = append(epochsByType[envelope.Type], envelope.Epoch)
	})
	defer signal.ResetHandler()

	tmpdir := t.TempDir()
	b := NewStatusBackend(testutils.MustCreateTestLogger())
	b.UpdateRootDataDir(tmpdir)
	initializeApplicationWithMediaServerPortHelper(t, b, reserveLocalhostPort(t))

	login := func() {
		_, err := b.CreateAccountAndLogin(&requests.CreateAccount{
			DisplayName:        "signal-epoch-test",
			CustomizationColor: "#ffffff",
			Password:           testPassword,
			RootDataDir:        tmpdir,
			LogFilePath:        tmpdir + "/log",
			KdfIterations:      1,
		})
		require.NoError(t, err)
	}

	login()
	before := signal.Epoch()

	require.NoError(t, b.Logout())
	require.Equal(t, before, signal.Epoch(), "internal logout must keep the epoch")

	login()
	mu.Lock()
	epochsByType = map[string][]uint64{}
	mu.Unlock()

	advanced, err := b.LogoutAndAdvanceSignalEpoch()
	require.NoError(t, err)
	require.Equal(t, before+1, advanced)
	require.Equal(t, advanced, signal.Epoch())

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []uint64{before}, epochsByType[signal.EventNodeStopped],
		"node.stopped belongs to the session that logged out")
}
