package server

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync"
	"testing"

	"go.uber.org/zap"
)

// Regression: StatusNode.Stop calls SetDataProviders(nil, nil, nil) while the
// media server is still serving, so handlers read the provider fields while
// they are being written. Run with -race.
func TestSetDataProvidersRacesHandlers(t *testing.T) {
	s := &MediaServer{
		Server: NewServer(zap.NewNop(), &Config{
			AddrPort: netip.MustParseAddrPort("127.0.0.1:0"),
		}),
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Reader: the same two-step read handleIPFS does (nil check, then use).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/ipfs?hash=x", nil)
			s.handleIPFS(rec, req)
		}
	}()

	// Writer: teardown clearing the providers underneath it.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			s.SetDataProviders(nil, nil, nil)
		}
		close(stop)
	}()

	wg.Wait()
}
