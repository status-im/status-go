package fetch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robertkrimen/otto"

	"github.com/status-im/status-go/geth/jail/internal/fetch"
	"github.com/status-im/status-go/geth/jail/internal/loop"
	"github.com/status-im/status-go/geth/jail/internal/vm"
	"github.com/stretchr/testify/suite"
)

func (s *FetchSuite) TestFetch() {
	ch := make(chan struct{})
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
		ch <- struct{}{}
	})

	err := fetch.Define(s.vm, s.loop)
	s.NoError(err)

	err = s.loop.Eval(`fetch('` + s.srv.URL + `').then(function(r) {
		    return r.text();
		  }).then(function(d) {
		    if (d.indexOf('hellox') === -1) {
		      throw new Error('what');
		    }
		  });`)
	s.NoError(err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
	}
}

func (s *FetchSuite) TestFetchCallback() {
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	err := fetch.Define(s.vm, s.loop)
	s.NoError(err)

	ch := make(chan struct{})
	err = s.vm.Set("__capture", func(str string) {
		s.Contains(str, "hello")
		ch <- struct{}{}
	})
	s.NoError(err)

	err = s.loop.Eval(`fetch('` + s.srv.URL + `').then(function(r) {
		return r.text();
	}).then(__capture)`)
	s.NoError(err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
	}
}

func (s *FetchSuite) TestFetchHeaders() {
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("header-one", "1")
		w.Header().Add("header-two", "2a")
		w.Header().Add("header-two", "2b")

		w.Write([]byte("hello"))
	})

	err := fetch.Define(s.vm, s.loop)
	s.NoError(err)

	ch := make(chan struct{})
	err = s.vm.Set("__capture", func(str string) {
		s.Equal(str, `{"header-one":["1"],"header-two":["2a","2b"]}`)
		ch <- struct{}{}
	})
	s.NoError(err)

	err = s.loop.Eval(`fetch('` + s.srv.URL + `').then(function(r) {
    return __capture(JSON.stringify({
      'header-one': r.headers.getAll('header-one'),
      'header-two': r.headers.getAll('header-two'),
    }));
  })`)
	s.NoError(err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
	}
}

func (s *FetchSuite) TestFetchJSON() {
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// these spaces are here so we can disambiguate between this and the
		// re-encoded data the javascript below spits out
		w.Write([]byte("[ 1 , 2 , 3 ]"))
	})

	err := fetch.Define(s.vm, s.loop)
	s.NoError(err)

	ch := make(chan struct{})
	err = s.vm.Set("__capture", func(str string) {
		s.Equal(str, `[1,2,3]`)
		ch <- struct{}{}
	})
	s.NoError(err)

	err = s.loop.Eval(`fetch('` + s.srv.URL + `').then(function(r) { return r.json(); }).then(function(d) {
    return setTimeout(__capture, 4, JSON.stringify(d));
  })`)
	s.NoError(err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
	}
}

func (s *FetchSuite) TestFetchWithHandler() {
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// these spaces are here so we can disambiguate between this and the
		// re-encoded data the javascript below spits out
		w.Write([]byte("[ 1 , 2 , 3 ]"))
	})

	err := fetch.DefineWithHandler(s.vm, s.loop, s.mux)
	s.NoError(err)

	ch := make(chan struct{})
	err = s.vm.Set("__capture", func(str string) {
		s.Equal(str, `[1,2,3]`)
		ch <- struct{}{}
	})
	s.NoError(err)

	err = s.loop.Eval(`fetch('/').then(function(r) { return r.json(); }).then(function(d) {
    return setTimeout(__capture, 4, JSON.stringify(d));
  })`)
	s.NoError(err)

	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		s.Fail("test timed out")
	}
}

func (s *FetchSuite) TestFetchWithHandlerParallel() {
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	err := fetch.DefineWithHandler(s.vm, s.loop, s.mux)
	s.NoError(err)

	ch := make(chan struct{})
	err = s.vm.Set("__capture", func(c otto.FunctionCall) otto.Value {
		ch <- struct{}{}

		return otto.UndefinedValue()
	})
	s.NoError(err)

	err = s.loop.Eval(`Promise.all([1,2,3,4,5].map(function(i) { return fetch('/' + i).then(__capture); }))`)
	s.NoError(err)

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
	s.Equal(5, count)
}

type FetchSuite struct {
	suite.Suite

	mux *http.ServeMux
	srv *httptest.Server

	loop *loop.Loop
	vm   *vm.VM
}

func (s *FetchSuite) SetupTest() {
	s.mux = http.NewServeMux()
	s.srv = httptest.NewServer(s.mux)

	o := otto.New()
	s.vm = vm.New(o)
	s.loop = loop.New(s.vm)

	go s.loop.Run(context.Background())
}

func (s *FetchSuite) TearDownSuite() {
	s.srv.Close()
}

func TestFetchSuite(t *testing.T) {
	suite.Run(t, new(FetchSuite))
}
