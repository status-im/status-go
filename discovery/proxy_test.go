package discovery

import (
	"context"
	"sync"
	"testing"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/rendezvous"
	"github.com/stretchr/testify/require"
)

func TestProxyToRendezvous(t *testing.T) {
	var (
		topic    = "test"
		id       = 101
		reg      = newRegistry()
		original = &fake{id: 110, registry: reg, started: true}
		srv      = makeTestRendezvousServer(t, "/ip4/127.0.0.1/tcp/7788")
		stop     = make(chan struct{})
		wg       sync.WaitGroup
	)
	client, err := rendezvous.NewTemporary()
	require.NoError(t, err)
	reg.Add(topic, id)
	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, ProxyToRendezvous(original, []ma.Multiaddr{srv.Addr()}, topic, stop))
	}()
	timer := time.After(3 * time.Second)
	ticker := time.Tick(100 * time.Millisecond)
	for {
		select {
		case <-timer:
			close(stop)
			wg.Wait()
			require.FailNow(t, "failed waiting for record to be proxied")
		case <-ticker:
			records, err := client.Discover(context.TODO(), srv.Addr(), topic, 10)
			if err != nil {
				continue
			}
			if len(records) != 1 {
				continue
			}
			var proxied Proxied
			require.NoError(t, records[0].Load(&proxied))
			require.Equal(t, proxied[0], byte(id))
			close(stop)
			wg.Wait()
			return
		}
	}
}
