package pairing

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/signal"
)

// blockingMounter holds the transfer until released, like a slow payload.
type blockingMounter struct {
	release chan struct{}
}

func (m *blockingMounter) Mount() error {
	<-m.release
	return nil
}

func (m *blockingMounter) ToSend() []byte { return []byte("payload") }

func (m *blockingMounter) LockPayload() {}

// A pairing server keeps serving after the call that started it returned. A transfer
// still running when the client logs out must send its events in the session the server
// was started in, not in the one the logout started.
func TestPairingHandlerKeepsItsSessionAcrossLogout(t *testing.T) {
	epochs := make(chan uint64, 8)
	signal.SetHandler(func(data []byte) {
		var e signal.Envelope
		if err := json.Unmarshal(data, &e); err == nil && e.Type == "localPairing" {
			epochs <- e.Epoch
		}
	})
	t.Cleanup(signal.ResetHandler)

	session := signal.CurrentSession()
	started := signal.Epoch()
	mounter := &blockingMounter{release: make(chan struct{})}
	handler := handleSendAccount(zap.NewNop(), session, mounter, func() {})

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	}()

	require.Equal(t, started, <-epochs) // connection success

	signal.AdvanceEpoch() // the logout
	close(mounter.release)
	<-done

	require.Equal(t, started, <-epochs) // transfer success, sent after the logout
}
