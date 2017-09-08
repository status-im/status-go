package fetch_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robertkrimen/otto"

	"github.com/status-im/status-go/geth/jail/ottoext/fetch"
	"github.com/status-im/status-go/geth/jail/ottoext/loop"
	"github.com/status-im/status-go/geth/jail/ottoext/vm"
	"github.com/stretchr/testify/require"
)

func TestFetch(t *testing.T) {
	ch := make(chan struct{})
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
		ch <- struct{}{}
	})
	s := httptest.NewServer(m)
	defer s.Close()

	v, l := newVM()
	err := fetch.Define(v, l)
	require.NoError(t, err)

	err = l.Eval(`fetch('` + s.URL + `').then(function(r) {
		    return r.text();
		  }).then(function(d) {
		    if (d.indexOf('hellox') === -1) {
		      throw new Error('what');
		    }
		  });`)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
		return
	}
}

func TestFetchCallback(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})
	s := httptest.NewServer(m)
	defer s.Close()

	v, l := newVM()
	err := fetch.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})

	err = v.Set("__capture", func(s string) {
		defer func() { ch <- struct{}{} }()

		require.Contains(t, s, "hello")
	})
	require.NoError(t, err)

	err = l.Eval(`fetch('` + s.URL + `').then(function(r) {
		return r.text();
	}).then(__capture)`)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
		return
	}
}

func TestFetchHeaders(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("header-one", "1")
		w.Header().Add("header-two", "2a")
		w.Header().Add("header-two", "2b")

		w.Write([]byte("hello"))
	})
	s := httptest.NewServer(m)
	defer s.Close()

	v, l := newVM()
	err := fetch.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})

	err = v.Set("__capture", func(s string) {
		defer func() { ch <- struct{}{} }()

		require.Equal(t, s, `{"header-one":["1"],"header-two":["2a","2b"]}`)
	})
	require.NoError(t, err)

	err = l.Eval(`fetch('` + s.URL + `').then(function(r) {
    return __capture(JSON.stringify({
      'header-one': r.headers.getAll('header-one'),
      'header-two': r.headers.getAll('header-two'),
    }));
  })`)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
		return
	}
}

func TestFetchJSON(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// these spaces are here so we can disambiguate between this and the
		// re-encoded data the javascript below spits out
		w.Write([]byte("[ 1 , 2 , 3 ]"))
	})
	s := httptest.NewServer(m)
	defer s.Close()

	v, l := newVM()
	err := fetch.Define(v, l)
	require.NoError(t, err)

	ch := make(chan struct{})

	err = v.Set("__capture", func(s string) {
		defer func() { ch <- struct{}{} }()

		require.Equal(t, s, `[1,2,3]`)
	})
	require.NoError(t, err)

	err = l.Eval(`fetch('` + s.URL + `').then(function(r) { return r.json(); }).then(function(d) {
    return setTimeout(__capture, 4, JSON.stringify(d));
  })`)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
		return
	}
}

func TestFetchJSONRepeated(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	for i := 0; i < 100; i++ {
		TestFetchJSON(t)
	}
}

func TestFetchWithHandler(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// these spaces are here so we can disambiguate between this and the
		// re-encoded data the javascript below spits out
		w.Write([]byte("[ 1 , 2 , 3 ]"))
	})

	v, l := newVM()
	err := fetch.DefineWithHandler(v, l, m)
	require.NoError(t, err)

	ch := make(chan struct{})

	err = v.Set("__capture", func(s string) {
		defer func() { ch <- struct{}{} }()

		require.Equal(t, s, `[1,2,3]`)
	})
	require.NoError(t, err)

	err = l.Eval(`fetch('/').then(function(r) { return r.json(); }).then(function(d) {
    return setTimeout(__capture, 4, JSON.stringify(d));
  })`)
	require.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		require.Fail(t, "test timed out")
		return
	}
}

func TestFetchWithHandlerParallel(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	v, l := newVM()
	err := fetch.DefineWithHandler(v, l, m)
	require.NoError(t, err)

	ch := make(chan struct{})

	err = v.Set("__capture", func(c otto.FunctionCall) otto.Value {
		ch <- struct{}{}

		return otto.UndefinedValue()
	})
	require.NoError(t, err)

	err = l.Eval(`Promise.all([1,2,3,4,5].map(function(i) { return fetch('/' + i).then(__capture); }))`)
	require.NoError(t, err)

	timerCh := time.After(1 * time.Second)
	var count int
loop:
	for i := 0; i < 5; i++ {
		select {
		case <-ch:
			count++
		case <-timerCh:
			break loop
		}
	}
	require.Equal(t, 5, count)
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
