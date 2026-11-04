package history

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmitter(t *testing.T) {
	emitter := NewEmitter[int]()

	subscr1 := emitter.Subscribe()
	subscr2 := emitter.Subscribe()

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		emitter.Emit(1)
		emitter.Emit(2)
	}()

	go func() {
		defer wg.Done()
		require.Equal(t, 1, <-subscr1)
		require.Equal(t, 2, <-subscr1)
	}()

	go func() {
		defer wg.Done()
		require.Equal(t, 1, <-subscr2)
		require.Equal(t, 2, <-subscr2)
	}()

	wg.Wait()
}

func TestOneShotEmitter(t *testing.T) {
	emitter := NewOneshotEmitter[struct{}]()

	subscr1 := emitter.Subscribe()
	subscr2 := emitter.Subscribe()

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		emitter.Emit(struct{}{})
	}()

	go func() {
		defer wg.Done()
		for range subscr1 {
		}
	}()

	go func() {
		defer wg.Done()
		for range subscr2 {
		}
	}()

	wg.Wait()
}
