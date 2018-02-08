package jail

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/stretchr/testify/suite"
)

func TestCellTestSuite(t *testing.T) {
	suite.Run(t, new(CellTestSuite))
}

type CellTestSuite struct {
	suite.Suite
	cell *Cell
}

func (s *CellTestSuite) SetupTest() {
	cell, err := NewCell("testCell1")
	s.NoError(err)
	s.NotNil(cell)

	s.cell = cell
}

func (s *CellTestSuite) TearDownTest() {
	err := s.cell.Stop()
	s.NoError(err)
}

func (s *CellTestSuite) TestCellRegisteredHandlers() {
	_, err := s.cell.Run(`setTimeout(function(){}, 100)`)
	s.NoError(err)

	_, err = s.cell.Run(`fetch`)
	s.NoError(err)
}

// TestJailLoopRace tests multiple setTimeout callbacks,
// supposed to be run with '-race' flag.
func (s *CellTestSuite) TestCellLoopRace() {
	cell := s.cell
	items := make(chan struct{})

	err := cell.Set("__captureResponse", func() otto.Value {
		items <- struct{}{}
		return otto.UndefinedValue()
	})
	s.NoError(err)

	_, err = cell.Run(`
		function callRunner(){
			return setTimeout(function(){
				__captureResponse();
			}, 200);
		}
	`)
	s.NoError(err)

	for i := 0; i < 100; i++ {
		_, err = cell.Call("callRunner", nil)
		s.NoError(err)
	}

	for i := 0; i < 100; i++ {
		select {
		case <-items:
		case <-time.After(400 * time.Millisecond):
			s.Fail("test timed out")
		}
	}
}

// TestJailFetchRace tests multiple sending multiple HTTP requests simultaneously in one cell, using `fetch`.
// Supposed to be run with '-race' flag.
func (s *CellTestSuite) TestCellFetchRace() {
	// How many request should the test perform ?
	const requestCount = 5

	// Create a test server that simply outputs the "i" parameter passed.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(r.URL.Query()["i"][0])) //nolint: errcheck
	}))

	defer server.Close()

	cell := s.cell
	dataCh := make(chan otto.Value, 1)
	errCh := make(chan otto.Value, 1)

	err := cell.Set("__captureSuccess", func(res otto.Value) { dataCh <- res })
	s.NoError(err)

	err = cell.Set("__captureError", func(res otto.Value) { errCh <- res })
	s.NoError(err)

	fetchCode := `fetch('%s?i=%d').then(function(r) {
 		return r.text()
	}).then(function(data) {
		__captureSuccess(data)
	}).catch(function (e) {
		__captureError(e)
	})`

	for i := 0; i < requestCount; i++ {
		_, err = cell.Run(fmt.Sprintf(fetchCode, server.URL, i))
		s.NoError(err)
	}

	expected := map[string]bool{} // It'll help us verify if every request was successfully completed.
	for i := 0; i < requestCount; i++ {
		select {
		case data := <-dataCh:
			// Mark the request as successful.
			expected[data.String()] = true
		case <-errCh:
			s.Fail("fetch failed to complete the request")
		case <-time.After(5 * time.Second):
			s.Fail("test timed out")
			return
		}
	}

	// Make sure every request was completed successfully.
	for i := 0; i < requestCount; i++ {
		s.Equal(expected[fmt.Sprintf("%d", i)], true)
	}

	// There might be some tasks about to call `ready`,
	// add a little delay before `TearDownTest` closes the loop.
	time.Sleep(100 * time.Millisecond)
}

func (s *CellTestSuite) TestCellFetchErrorRace() {
	cell := s.cell
	dataCh := make(chan otto.Value, 1)
	errCh := make(chan otto.Value, 1)

	err := cell.Set("__captureSuccess", func(res otto.Value) { dataCh <- res })
	s.NoError(err)
	err = cell.Set("__captureError", func(res otto.Value) { errCh <- res })
	s.NoError(err)

	// Find a free port in localhost
	freeportListener, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	defer freeportListener.Close()

	// Send an HTTP request to the free port that we found above
	_, err = cell.Run(fmt.Sprintf(`fetch('%s').then(function(r) {
		return r.text()
	}).then(function(data) {
		__captureSuccess(data)
	}).catch(function (e) {
		__captureError(e)
	})`, freeportListener.Addr().String()))
	s.NoError(err)

	select {
	case _ = <-dataCh:
		s.Fail("fetch didn't return error for nonexistent url")
	case e := <-errCh:
		name, err := e.Object().Get("name")
		s.NoError(err)
		s.Equal("Error", name.String())
		_, err = e.Object().Get("message")
		s.NoError(err)
	case <-time.After(5 * time.Second):
		s.Fail("test timed out")
		return
	}
}

// TestCellLoopCancel tests that cell.Stop() really cancels event
// loop and pending tasks.
func (s *CellTestSuite) TestCellLoopCancel() {
	cell := s.cell

	var err error
	var count int

	err = cell.Set("__captureResponse", func(call otto.FunctionCall) otto.Value {
		count++
		return otto.UndefinedValue()
	})
	s.NoError(err)

	_, err = cell.Run(`
		function callRunner(delay){
			return setTimeout(function(){
				__captureResponse();
			}, delay);
		}
	`)
	s.NoError(err)

	// Run 5 timeout tasks to be executed in: 1, 2, 3, 4 and 5 secs
	for i := 1; i <= 5; i++ {
		_, err = cell.Call("callRunner", nil, i*1000)
		s.NoError(err)
	}

	// Wait 1.5 second (so only one task executed) so far
	// and stop the cell (event loop should die)
	time.Sleep(1500 * time.Millisecond)
	err = cell.Stop()
	s.NoError(err)

	// check that only 1 task has increased counter
	s.Equal(1, count)

	// wait 2 seconds more (so at least two more tasks would
	// have been executed if event loop is still running)
	<-time.After(2 * time.Second)

	// check that counter hasn't increased
	s.Equal(1, count)
}

func (s *CellTestSuite) TestCellCallAsync() {
	// Don't use buffered channel as it's supposed to be an async call.
	datac := make(chan string)

	err := s.cell.Set("testCallAsync", func(call otto.FunctionCall) otto.Value {
		datac <- call.Argument(0).String()
		return otto.UndefinedValue()
	})
	s.NoError(err)

	fn, err := s.cell.Get("testCallAsync")
	s.NoError(err)

	s.cell.CallAsync(fn, "success")
	s.Equal("success", <-datac)
}

func (s *CellTestSuite) TestCellCallStopMultipleTimes() {
	s.NotPanics(func() {
		err := s.cell.Stop()
		s.NoError(err)
		err = s.cell.Stop()
		s.NoError(err)
	})
}
