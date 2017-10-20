package jail

import (
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
	s.cell.Stop()
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

// TestJailFetchRace tests multiple fetch callbacks,
// supposed to be run with '-race' flag.
func (s *CellTestSuite) TestCellFetchRace() {
	body := `{"key": "value"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(body)) //nolint: errcheck
	}))
	defer server.Close()

	cell := s.cell
	dataCh := make(chan otto.Value, 1)
	errCh := make(chan otto.Value, 1)

	var err error

	err = cell.Set("__captureSuccess", func(res otto.Value) { dataCh <- res })
	s.NoError(err)
	err = cell.Set("__captureError", func(res otto.Value) { errCh <- res })
	s.NoError(err)

	// run JS code for fetching valid URL
	_, err = cell.Run(`fetch('` + server.URL + `').then(function(r) {
		return r.text()
	}).then(function(data) {
		__captureSuccess(data)
	}).catch(function (e) {
		__captureError(e)
	})`)
	s.NoError(err)

	// run JS code for fetching invalid URL
	_, err = cell.Run(`fetch('http://👽/nonexistent').then(function(r) {
		return r.text()
	}).then(function(data) {
		__captureSuccess(data)
	}).catch(function (e) {
		__captureError(e)
	})`)
	s.NoError(err)

	for i := 0; i < 2; i++ {
		select {
		case data := <-dataCh:
			s.Equal(body, data.String())
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
