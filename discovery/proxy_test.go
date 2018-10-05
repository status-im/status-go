package discovery

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/rendezvous"
	"github.com/stretchr/testify/require"
)

func TestProxyToRendezvous(t *testing.T) {
	var (
		topic    = "test"
		id       = 101
		limited  = 102
		limit    = 1
		reg      = newRegistry()
		original = &fake{id: 110, registry: reg, started: true}
		srv      = makeTestRendezvousServer(t, "/ip4/127.0.0.1/tcp/7788")
		stop     = make(chan struct{})
		feed     = &event.Feed{}
		liveness = 100 * time.Millisecond
		wg       sync.WaitGroup
	)
	client, err := rendezvous.NewEphemeral()
	require.NoError(t, err)
	reg.Add(topic, id)
	reg.Add(topic, limited)
	wg.Add(1)
	events := make(chan proxyEvent, 10)
	sub := feed.Subscribe(events)
	defer sub.Unsubscribe()
	go func() {
		defer wg.Done()
		require.NoError(t, ProxyToRendezvous(original, stop, feed, ProxyOptions{
			Topic:          topic,
			Servers:        []ma.Multiaddr{srv.Addr()},
			Limit:          limit,
			LivenessWindow: liveness,
		}))
	}()
	require.NoError(t, Consistently(func() (bool, error) {
		records, err := client.Discover(context.TODO(), srv.Addr(), topic, 10)
		if err != nil && len(records) < limit {
			return true, nil
		}
		if len(records) > limit {
			return false, fmt.Errorf("more records than expected: %d != %d", len(records), limit)
		}
		var proxied Proxied
		if err := records[0].Load(&proxied); err != nil {
			return false, err
		}
		if proxied[0] != byte(id) {
			return false, fmt.Errorf("returned %v instead of %v", proxied[0], id)
		}
		return true, nil
	}, time.Second, 100*time.Millisecond))
	close(stop)
	wg.Wait()
	eventSlice := []proxyEvent{}
	func() {
		for {
			select {
			case e := <-events:
				eventSlice = append(eventSlice, e)
			default:
				return
			}
		}
	}()
	require.Len(t, eventSlice, 2)
	require.Equal(t, byte(id), eventSlice[0].ID[0])
	require.Equal(t, proxyStart, eventSlice[0].Type)
	require.Equal(t, byte(id), eventSlice[1].ID[0])
	require.Equal(t, proxyStop, eventSlice[1].Type)
	require.True(t, eventSlice[1].Time.Sub(eventSlice[0].Time) > liveness)
}

func Consistently(f func() (bool, error), timeout, period time.Duration) (err error) {
	timer := time.After(timeout)
	ticker := time.Tick(period)
	var cont bool
	for {
		select {
		case <-timer:
			return err
		case <-ticker:
			cont, err = f()
			if cont {
				continue
			}
			if err != nil {
				return err
			}
		}
	}
}
