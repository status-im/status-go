package fetch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/robertkrimen/otto"

	"github.com/status-im/status-go/geth/jail/ottoext/loop"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func TestFetch(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})
	s := httptest.NewServer(m)
	defer s.Close()

	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`fetch('` + s.URL + `').then(function(r) {
    return r.text();
  }).then(function(d) {
    if (d.indexOf('hellox') === -1) {
      throw new Error('what');
    }
  });`))
}

func TestFetchCallback(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})
	s := httptest.NewServer(m)
	defer s.Close()

	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	ch := make(chan bool, 1)

	if err := vm.Set("__capture", func(s string) {
		defer func() { ch <- true }()

		if !strings.Contains(s, "hello") {
			panic(fmt.Errorf("expected to find `hello' in response"))
		}
	}); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`fetch('` + s.URL + `').then(function(r) {
    return r.text();
  }).then(__capture)`))

	<-ch
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

	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	ch := make(chan bool, 1)

	if err := vm.Set("__capture", func(s string) {
		defer func() { ch <- true }()

		if s != `{"header-one":["1"],"header-two":["2a","2b"]}` {
			panic(fmt.Errorf("expected headers to contain 1, 2a, and 2b"))
		}
	}); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`fetch('` + s.URL + `').then(function(r) {
    return __capture(JSON.stringify({
      'header-one': r.headers.getAll('header-one'),
      'header-two': r.headers.getAll('header-two'),
    }));
  })`))

	<-ch
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

	vm := otto.New()
	l := loop.New(vm)

	if err := Define(vm, l); err != nil {
		panic(err)
	}

	ch := make(chan bool, 1)

	if err := vm.Set("__capture", func(s string) {
		defer func() { ch <- true }()

		if s != `[1,2,3]` {
			panic(fmt.Errorf("expected data to be json, and for that json to be parsed"))
		}
	}); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`fetch('` + s.URL + `').then(function(r) { return r.json(); }).then(function(d) {
    return setTimeout(__capture, 4, JSON.stringify(d));
  })`))

	<-ch
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

	vm := otto.New()
	l := loop.New(vm)

	if err := DefineWithHandler(vm, l, m); err != nil {
		panic(err)
	}

	ch := make(chan bool, 1)

	if err := vm.Set("__capture", func(s string) {
		defer func() { ch <- true }()

		if s != `[1,2,3]` {
			panic(fmt.Errorf("expected data to be json, and for that json to be parsed"))
		}
	}); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`fetch('/').then(function(r) { return r.json(); }).then(function(d) {
    return setTimeout(__capture, 4, JSON.stringify(d));
  })`))

	<-ch
}

func TestFetchWithHandlerParallel(t *testing.T) {
	m := http.NewServeMux()
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	vm := otto.New()
	l := loop.New(vm)

	if err := DefineWithHandler(vm, l, m); err != nil {
		panic(err)
	}

	ch := make(chan bool, 1)

	if err := vm.Set("__capture", func(c otto.FunctionCall) otto.Value {
		defer func() { ch <- true }()

		return otto.UndefinedValue()
	}); err != nil {
		panic(err)
	}

	must(l.EvalAndRun(`Promise.all([1,2,3,4,5].map(function(i) { return fetch('/' + i); })).then(__capture)`))

	<-ch
}
