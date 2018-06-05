package fetch

//go:generate go-bindata -pkg fetch -o dist_fetch.go ./dist-fetch/

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/robertkrimen/otto"

	"github.com/status-im/status-go/jail/internal/loop"
	"github.com/status-im/status-go/jail/internal/promise"
	"github.com/status-im/status-go/jail/internal/vm"
)

func mustValue(v otto.Value, err error) otto.Value {
	if err != nil {
		panic(err)
	}

	return v
}

type fetchTask struct {
	id           int64
	jsReq, jsRes *otto.Object
	cb           otto.Value
	err          error
	status       int
	statusText   string
	headers      map[string][]string
	body         []byte
}

func (t *fetchTask) SetID(id int64) { t.id = id }
func (t *fetchTask) GetID() int64   { return t.id }

func (t *fetchTask) Execute(vm *vm.VM, l *loop.Loop) error {
	var arguments []interface{}

	if t.err != nil {
		e, err := vm.Call(`new Error`, nil, t.err.Error())
		if err != nil {
			return err
		}

		arguments = append(arguments, e)
	}

	// We're locking on VM here because underlying otto's VM
	// is not concurrently safe, and this function indirectly
	// access vm's functions in cb.Call/h.Set.
	vm.Lock()
	defer vm.Unlock()

	err := t.jsRes.Set("status", t.status)
	if err != nil {
		return err
	}

	err = t.jsRes.Set("statusText", t.statusText)
	if err != nil {
		return err
	}

	h := mustValue(t.jsRes.Get("headers")).Object()
	for k, vs := range t.headers {
		for _, v := range vs {
			if _, err = h.Call("append", k, v); err != nil {
				return err
			}
		}
	}
	err = t.jsRes.Set("_body", string(t.body))
	if err != nil {
		return err
	}

	_, err = t.cb.Call(otto.NullValue(), arguments...)
	return err
}

func (t *fetchTask) Cancel() {
}

// Define fetch
func Define(vm *vm.VM, l *loop.Loop) error {
	return DefineWithHandler(vm, l, nil)
}

//DefineWithHandler fetch with handler
func DefineWithHandler(vm *vm.VM, l *loop.Loop, h http.Handler) error {
	if err := promise.Define(vm, l); err != nil {
		return err
	}

	jsData := MustAsset("dist-fetch/bundle.js")
	smData := MustAsset("dist-fetch/bundle.js.map")

	s, err := vm.CompileWithSourceMap("fetch-bundle.js", jsData, smData)
	if err != nil {
		return err
	}

	_, err = vm.Run(s)
	if err != nil {
		return err
	}

	err = vm.Set("__private__fetch_execute", func(c otto.FunctionCall) otto.Value {
		jsReq := c.Argument(0).Object()
		jsRes := c.Argument(1).Object()
		cb := c.Argument(2)

		method := mustValue(jsReq.Get("method")).String()
		urlStr := mustValue(jsReq.Get("url")).String()
		jsBody := mustValue(jsReq.Get("body"))
		var body io.Reader
		if jsBody.IsString() {
			body = strings.NewReader(jsBody.String())
		}

		t := &fetchTask{
			jsReq: jsReq,
			jsRes: jsRes,
			cb:    cb,
		}

		// If err is non-nil, then the loop is closed
		// and we shouldn't do anymore with it.
		if err := l.Add(t); err != nil {
			return otto.UndefinedValue()
		}

		go func() {
			defer l.Ready(t) // nolint: errcheck

			req, rqErr := http.NewRequest(method, urlStr, body)
			if rqErr != nil {
				t.err = rqErr
				return
			}

			if h != nil && urlStr[0] == '/' {
				res := httptest.NewRecorder()

				h.ServeHTTP(res, req)

				t.status = res.Code
				t.statusText = http.StatusText(res.Code)
				t.headers = res.Header()
				t.body = res.Body.Bytes()
			} else {
				res, e := http.DefaultClient.Do(req)
				if e != nil {
					t.err = e
					return
				}

				d, e := ioutil.ReadAll(res.Body)
				if e != nil {
					t.err = e
					return
				}

				t.status = res.StatusCode
				t.statusText = res.Status
				t.headers = res.Header
				t.body = d
			}
		}()

		return otto.UndefinedValue()
	})

	return err
}
