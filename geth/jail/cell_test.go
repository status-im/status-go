package jail_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/static"
	"github.com/stretchr/testify/suite"
)

const (
	testChatID = "testChat"
)

var (
	baseStatusJSCode = string(static.MustAsset("testdata/jail/status.js"))
)

type CellTestSuite struct {
	suite.Suite
	jail *jail.Jail
}

func (s *CellTestSuite) SetupTest() {
	s.jail = jail.New(nil)
	s.NotNil(s.jail)
}

func (s *CellTestSuite) TestJailTimeout() {
	require := s.Require()

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)
	defer cell.Stop()

	// Attempt to run a timeout string against a Cell.
	_, err = cell.Run(`
		var timerCounts = 0;
 		setTimeout(function(n){
 			if (Date.now() - n < 50) {
 				throw new Error("Timed out");
 			}

			timerCounts++;
 		}, 50, Date.now());
 	`)
	require.NoError(err)

	// wait at least 10x longer to decrease probability
	// of false negatives as we using real clock here
	time.Sleep(300 * time.Millisecond)

	value, err := cell.Get("timerCounts")
	require.NoError(err)
	require.True(value.IsNumber())
	require.Equal("1", value.String())
}

func (s *CellTestSuite) TestJailLoopInCall() {
	require := s.Require()

	// load Status JS and add test command to it
	s.jail.BaseJS(baseStatusJSCode)
	s.jail.Parse(testChatID, ``)

	cell, err := s.jail.Cell(testChatID)
	require.NoError(err)
	require.NotNil(cell)
	defer cell.Stop()

	items := make(chan string)

	err = cell.Set("__captureResponse", func(val string) otto.Value {
		go func() { items <- val }()
		return otto.UndefinedValue()
	})
	require.NoError(err)

	_, err = cell.Run(`
		function callRunner(namespace){
			console.log("Initiating callRunner for: ", namespace)
			return setTimeout(function(){
				__captureResponse(namespace);
			}, 1000);
		}
	`)
	require.NoError(err)

	_, err = cell.Call("callRunner", nil, "softball")
	require.NoError(err)

	select {
	case received := <-items:
		require.Equal(received, "softball")
	case <-time.After(5 * time.Second):
		require.Fail("Failed to received event response")
	}
}

// TestJailLoopRace tests multiple setTimeout callbacks,
// supposed to be run with '-race' flag.
func (s *CellTestSuite) TestJailLoopRace() {
	require := s.Require()

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)
	defer cell.Stop()

	items := make(chan struct{})

	err = cell.Set("__captureResponse", func() otto.Value {
		go func() { items <- struct{}{} }()
		return otto.UndefinedValue()
	})
	require.NoError(err)

	_, err = cell.Run(`
		function callRunner(){
			return setTimeout(function(){
				__captureResponse();
			}, 1000);
		}
	`)
	require.NoError(err)

	for i := 0; i < 100; i++ {
		_, err = cell.Call("callRunner", nil)
		require.NoError(err)
	}

	for i := 0; i < 100; i++ {
		select {
		case <-items:
		case <-time.After(5 * time.Second):
			require.Fail("test timed out")
		}
	}
}

func (s *CellTestSuite) TestJailFetchPromise() {
	body := `{"key": "value"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer server.Close()

	require := s.Require()

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)
	defer cell.Stop()

	dataCh := make(chan otto.Value, 1)
	errCh := make(chan otto.Value, 1)

	err = cell.Set("__captureSuccess", func(res otto.Value) { dataCh <- res })
	require.NoError(err)
	err = cell.Set("__captureError", func(res otto.Value) { errCh <- res })
	require.NoError(err)

	// run JS code for fetching valid URL
	_, err = cell.Run(`fetch('` + server.URL + `').then(function(r) {
		return r.text()
	}).then(function(data) {
		__captureSuccess(data)
	}).catch(function (e) {
		__captureError(e)
	})`)
	require.NoError(err)

	select {
	case data := <-dataCh:
		require.True(data.IsString())
		require.Equal(body, data.String())
	case err := <-errCh:
		require.Fail("request failed", err)
	case <-time.After(1 * time.Second):
		require.Fail("test timed out")
	}
}

func (s *CellTestSuite) TestJailFetchCatch() {
	require := s.Require()

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)
	defer cell.Stop()

	dataCh := make(chan otto.Value, 1)
	errCh := make(chan otto.Value, 1)

	err = cell.Set("__captureSuccess", func(res otto.Value) { dataCh <- res })
	require.NoError(err)
	err = cell.Set("__captureError", func(res otto.Value) { errCh <- res })
	require.NoError(err)

	// run JS code for fetching invalid URL
	_, err = cell.Run(`fetch('http://👽/nonexistent').then(function(r) {
		return r.text()
	}).then(function(data) {
		__captureSuccess(data)
	}).catch(function (e) {
		__captureError(e)
	})`)
	require.NoError(err)

	select {
	case data := <-dataCh:
		require.Fail("request should have failed, but returned", data)
	case e := <-errCh:
		require.True(e.IsObject())
		name, err := e.Object().Get("name")
		require.NoError(err)
		require.Equal("Error", name.String())
		_, err = e.Object().Get("message")
		require.NoError(err)
	case <-time.After(1 * time.Second):
		require.Fail("test timed out")
	}
}

// TestJailFetchRace tests multiple fetch callbacks,
// supposed to be run with '-race' flag.
func (s *CellTestSuite) TestJailFetchRace() {
	body := `{"key": "value"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer server.Close()
	require := s.Require()

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)
	defer cell.Stop()

	dataCh := make(chan otto.Value, 1)
	errCh := make(chan otto.Value, 1)

	err = cell.Set("__captureSuccess", func(res otto.Value) { dataCh <- res })
	require.NoError(err)
	err = cell.Set("__captureError", func(res otto.Value) { errCh <- res })
	require.NoError(err)

	// run JS code for fetching valid URL
	_, err = cell.Run(`fetch('` + server.URL + `').then(function(r) {
		return r.text()
	}).then(function(data) {
		__captureSuccess(data)
	}).catch(function (e) {
		__captureError(e)
	})`)
	require.NoError(err)

	// run JS code for fetching invalid URL
	_, err = cell.Run(`fetch('http://👽/nonexistent').then(function(r) {
		return r.text()
	}).then(function(data) {
		__captureSuccess(data)
	}).catch(function (e) {
		__captureError(e)
	})`)
	require.NoError(err)

	for i := 0; i < 2; i++ {
		select {
		case data := <-dataCh:
			require.True(data.IsString())
			require.Equal(body, data.String())
		case e := <-errCh:
			require.True(e.IsObject())
			name, err := e.Object().Get("name")
			require.NoError(err)
			require.Equal("Error", name.String())
			_, err = e.Object().Get("message")
			require.NoError(err)
		case <-time.After(1 * time.Second):
			require.Fail("test timed out")
			return
		}
	}
}

// TestJailLoopCancel tests that cell.Stop() really cancels event
// loop and pending tasks.
func (s *CellTestSuite) TestJailLoopCancel() {
	require := s.Require()

	// load Status JS and add test command to it
	s.jail.BaseJS(baseStatusJSCode)
	s.jail.Parse(testChatID, ``)

	cell, err := s.jail.Cell(testChatID)
	require.NoError(err)
	require.NotNil(cell)

	var count int
	err = cell.Set("__captureResponse", func(val string) otto.Value {
		count++
		return otto.UndefinedValue()
	})
	require.NoError(err)

	_, err = cell.Run(`
		function callRunner(val, delay){
			return setTimeout(function(){
				__captureResponse(val);
			}, delay);
		}
	`)
	require.NoError(err)

	// Run 5 timeout tasks to be executed in: 1, 2, 3, 4 and 5 secs
	for i := 1; i <= 5; i++ {
		_, err = cell.Call("callRunner", nil, "value", i*1000)
		require.NoError(err)
	}

	// Wait 1.5 second (so only one task executed) so far
	// and stop the cell (event loop should die)
	time.Sleep(1500 * time.Millisecond)
	cell.Stop()

	// check that only 1 task has increased counter
	require.Equal(1, count)

	// wait 2 seconds more (so at least two more tasks would
	// have been executed if event loop is still running)
	<-time.After(2 * time.Second)

	// check that counter hasn't increased
	require.Equal(1, count)
}

// TestJailCellStopSetIntervalBeforeStop checks that after Cell will be stopped,
// setInterval command (which was appended before)
// will not execute anything
func (s *CellTestSuite) TestJailCellStopSetIntervalBeforeStop() {
	require := s.Require()

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)

	_, err = cell.Run(`
		var counter = 1;
 		setInterval(function(){
 			counter++;
 		}, 10);
 	`)
	require.NoError(err)

	cell.Stop()

	val1, err := cell.Get("counter")
	require.NoError(err)
	require.True(val1.IsNumber())
	valInt1, err := val1.ToInteger()
	require.NoError(err)
	time.Sleep(20 * time.Millisecond)
	val2, err := cell.Get("counter")
	require.NoError(err)
	require.True(val2.IsNumber())
	valInt2, err := val2.ToInteger()
	require.Equal(valInt1, valInt2,
		"Unexpected behavior: Cell stopped but setInterval function was executed again")
}

// TestJailCellStopSetIntervalAfterStop checks that after Cell will be stopped,
// commands will not executing
func (s *CellTestSuite) TestJailCellStopSetIntervalAfterStop() {
	var (
		require        = s.Require()
		getCalledTimes = func(cell *common.JailCell) int64 {
			ottoValue, err := (*cell).Get("calledTimes")
			require.NoError(err)
			require.True(ottoValue.IsNumber())
			val, err := ottoValue.ToInteger()
			require.NoError(err)
			return val
		}
	)

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)

	_, err = cell.Run(`
		var counter = 1;
 		setInterval(function(){
 			counter++;
 		}, 10);
 	`)

	select {
	case <-cell.Stop():
		break
	case <-time.After(time.Second):
		require.Fail("Cell Stop timeout")
	}

	firstValue := getCalledTimes(&cell)

	time.Sleep(40 * time.Millisecond)

	//VM is not removed and we still can get values, but loop stopped and setInterval will not execute
	secondValue := getCalledTimes(&cell)
	require.Equal(firstValue, secondValue, "Expected stopped setInterval function")
}

// TestJailCellStopTwice stops cell twice
// Expected correct answer
func (s *CellTestSuite) TestJailCellStopTwice() {
	require := s.Require()

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)

	select {
	case <-cell.Stop():
		break
	case <-time.After(100 * time.Millisecond):
		require.Fail("Expected less than 100ms duration for cell stopping")
	}

	select {
	case <-cell.Stop():
		break
	case <-time.After(10 * time.Millisecond):
		require.Fail("Expected less than 10ms duration for trying to stop stopped cell")
	}
}
