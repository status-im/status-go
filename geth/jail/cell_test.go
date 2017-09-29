package jail_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/robertkrimen/otto"
)

func (s *JailTestSuite) TestJailTimeoutFailure() {
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
 		}, 30, Date.now());
 	`)
	require.NoError(err)

	// wait at least 10x longer to decrease probability
	// of false negatives as we using real clock here
	time.Sleep(300 * time.Millisecond)

	value, err := cell.Get("timerCounts")
	require.NoError(err)
	require.True(value.IsNumber())
	require.Equal("0", value.String())
}

func (s *JailTestSuite) TestJailTimeout() {
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

func (s *JailTestSuite) TestJailLoopInCall() {
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
func (s *JailTestSuite) TestJailLoopRace() {
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

func (s *JailTestSuite) TestJailFetchPromise() {
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

func (s *JailTestSuite) TestJailFetchCatch() {
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
func (s *JailTestSuite) TestJailFetchRace() {
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
