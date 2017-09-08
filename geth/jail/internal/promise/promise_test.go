package promise_test

import (
	"testing"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/promise"
	"github.com/status-im/status-go/geth/jail/internal/vm"
)

func TestResolve(t *testing.T) {
	v, l := newVM()

	err := promise.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})
	err = v.Set("__resolve", func(s string) {
		defer func() { ch <- struct{}{} }()

		require.Equal(t, "good", s)
	})
	require.NoError(t, err)

	err = l.Eval(`
		var p = new Promise(function(resolve, reject) {
			setTimeout(function() {
				resolve('good');
			}, 10);
		});

		p.then(function(d) {
			__resolve(d);
		});

		p.catch(function(err) {
			throw err;
		});
	`)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
		return
	}
}

func TestReject(t *testing.T) {
	v, l := newVM()

	err := promise.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})
	err = v.Set("__reject", func(s string) {
		defer func() { ch <- struct{}{} }()

		require.Equal(t, "bad", s)
	})
	require.NoError(t, err)

	err = l.Eval(`
		var p = new Promise(function(resolve, reject) {
			setTimeout(function() {
				reject('bad');
			}, 10);
		});

		p.catch(function(err) {
			__reject(err);
		});
	`)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
		return
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
