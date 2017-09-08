package timers_test

import (
	"testing"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/geth/jail/ottoext/loop"
	"github.com/status-im/status-go/geth/jail/ottoext/timers"
	"github.com/status-im/status-go/geth/jail/ottoext/vm"
)

func TestSetTimeout(t *testing.T) {
	v, l := newVM()

	err := timers.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})
	err = v.Set("__capture", func() {
		defer func() { ch <- struct{}{} }()
	})
	require.NoError(t, err)

	err = l.Eval(`setTimeout(function(n) {
		if (Date.now() - n < 50) {
			throw new Error('timeout was called too soon');
		}
		__capture();
	}, 50, Date.now());`)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
		return
	}
}

func TestClearTimeout(t *testing.T) {
	v, l := newVM()

	err := timers.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})
	err = v.Set("__shouldNeverRun", func() {
		defer func() { ch <- struct{}{} }()
	})
	require.NoError(t, err)

	err = l.Eval(`clearTimeout(setTimeout(function() {
		__shouldNeverRun();
	}, 50));`)
	require.NoError(t, err)

	select {
	case <-ch:
		require.Fail(t, "should never run")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSetInterval(t *testing.T) {
	v, l := newVM()

	err := timers.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})
	err = v.Set("__done", func() {
		defer func() { ch <- struct{}{} }()
	})
	require.NoError(t, err)

	err = l.Eval(`
		var c = 0;
		var iv = setInterval(function() {
			if (c === 1) {
				clearInterval(iv);
				__done();
			}
			c++;
		}, 50);
	`)
	require.NoError(t, err)

	select {
	case <-ch:
		value, err := v.Get("c")
		require.NoError(t, err)
		n, err := value.ToInteger()
		require.NoError(t, err)
		require.Equal(t, 2, int(n))
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
	}
}

func TestClearIntervalImmediately(t *testing.T) {
	v, l := newVM()

	err := timers.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})
	err = v.Set("__shouldNeverRun", func() {
		defer func() { ch <- struct{}{} }()
	})
	require.NoError(t, err)

	err = l.Eval(`clearInterval(setInterval(function() {
		__shouldNeverRun();
	}, 50));`)
	require.NoError(t, err)

	select {
	case <-ch:
		require.Fail(t, "should never run")
	case <-time.After(100 * time.Millisecond):
	}
}

// newVM creates new VM along with the loop.
//
// Currently all ottoext Define-functions accepts both
// vm and loop as a tuple. It should be
// refactored to accept only loop (which has an access to vm),
// and this function provide easy way
// to reflect this refactor for tests at least.
func newVM() (*vm.VM, *loop.Loop) {
	o := otto.New()
	v := vm.New(o)
	l := loop.New(v)
	go l.Run()
	return v, l
}
