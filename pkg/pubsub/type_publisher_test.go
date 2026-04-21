package pubsub

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Mirrors PauseBroadcaster.Subscribe: an initial value is placed on the subscriber channel,
// then TypePublisher.Publish runs. With buffer size 1, Publish uses non-blocking send and
// drops the new value if the buffer is still full — the consumer can then observe a stale
// first read relative to the last Publish. See common.TestPauseBroadcaster_Subscribe_*.
func TestTypePublisher_publishDroppedWhenPerSubscriberBufferFull(t *testing.T) {
	p := NewTypePublisher[bool]()
	ch := p.Subscribe(1)
	select {
	case ch <- false:
	default:
		t.Fatal("initial direct send should succeed on an empty buffer")
	}
	p.Publish(true)

	first, ok := <-ch
	require.True(t, ok)
	require.False(t, first, "first value should still be the buffered false, not the published true")

	select {
	case extra, ok := <-ch:
		if ok {
			t.Fatalf("unexpected second value %v; published true was dropped when buffer was full", extra)
		}
	default:
	}
}
