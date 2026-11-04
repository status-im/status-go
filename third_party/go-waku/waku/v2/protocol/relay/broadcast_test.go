package relay

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestBroadcast(t *testing.T) {
	wg := sync.WaitGroup{}

	b := NewBroadcaster(100)
	require.NoError(t, b.Start(context.Background()))
	defer b.Stop()

	for i := 0; i < 5; i++ {
		wg.Add(1)

		sub := b.RegisterForAll()
		go func() {
			defer wg.Done()
			defer sub.Unsubscribe()
			<-sub.Ch
		}()

	}

	env := protocol.NewEnvelope(&pb.WakuMessage{}, *utils.GetUnixEpoch(), "abc")
	b.Submit(env)

	wg.Wait()
}

func TestBroadcastSpecificTopic(t *testing.T) {
	wg := sync.WaitGroup{}

	b := NewBroadcaster(100)
	require.NoError(t, b.Start(context.Background()))
	defer b.Stop()

	for i := 0; i < 5; i++ {
		wg.Add(1)

		sub := b.Register(protocol.NewContentFilter("abc"))

		go func() {
			defer wg.Done()
			<-sub.Ch
			sub.Unsubscribe()
		}()

	}

	env := protocol.NewEnvelope(&pb.WakuMessage{}, *utils.GetUnixEpoch(), "abc")
	b.Submit(env)

	wg.Wait()
}

// check return from channel after Stop and multiple unregister
func TestBroadcastCleanup(t *testing.T) {
	b := NewBroadcaster(100)
	require.NoError(t, b.Start(context.Background()))
	sub := b.Register(protocol.NewContentFilter("test"))
	b.Stop()
	<-sub.Ch
	sub.Unsubscribe()
	sub.Unsubscribe()
}

func TestBroadcastUnregisterSub(t *testing.T) {
	b := NewBroadcaster(100)
	require.NoError(t, b.Start(context.Background()))
	subForAll := b.RegisterForAll()
	// unregister before submit
	specificSub := b.Register(protocol.NewContentFilter("abc"))
	specificSub.Unsubscribe()
	//
	env := protocol.NewEnvelope(&pb.WakuMessage{}, *utils.GetUnixEpoch(), "abc")
	b.Submit(env)
	// no message on specific sub
	require.Nil(t, <-specificSub.Ch)
	// msg on subForAll
	require.Equal(t, env, <-subForAll.Ch)
	b.Stop() // it automatically unregister/unsubscribe all
	require.Nil(t, <-specificSub.Ch)
}

func TestBroadcastNoOneListening(t *testing.T) {
	b := NewBroadcaster(100)
	require.NoError(t, b.Start(context.Background()))
	_ = b.RegisterForAll() // no one listening on channel
	//
	env := protocol.NewEnvelope(&pb.WakuMessage{}, *utils.GetUnixEpoch(), "abc")
	b.Submit(env)
	b.Submit(env)
	b.Stop()
}
