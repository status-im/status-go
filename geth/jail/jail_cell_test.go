package jail_test

import (
	"time"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/params"
)

func (s *JailTestSuite) TestJailTimeoutFailure() {
	require := s.Require()
	require.NotNil(s.jail)

	newCell, err := s.jail.NewJailCell(testChatID)
	require.NoError(err)
	require.NotNil(newCell)

	// Attempt to run a timeout string against a JailCell.
	_, err = newCell.RunOnLoop(`
		setTimeout(function(n){
			if(Date.now() - n < 50){
				throw new Error("Timedout early");
			}

			return n;
		}, 30, Date.now());
	`)

	require.NotNil(err)
}

func (s *JailTestSuite) TestJailTimeout() {
	require := s.Require()
	require.NotNil(s.jail)

	newCell, err := s.jail.NewJailCell(testChatID)
	require.NoError(err)
	require.NotNil(newCell)

	// Attempt to run a timeout string against a JailCell.
	res, err := newCell.RunOnLoop(`
		setTimeout(function(n){
			if(Date.now() - n < 50){
				throw new Error("Timedout early");
			}

			return n;
		}, 50, Date.now());
	`)

	require.NoError(err)
	require.NotNil(res)
}

// TODO(influx6): JCell.Fetch is not yet being used or needed by
// other API, keep test out for the main time, if method is later required
// add to common.JailCell interface then uncomment test.
// func (s *JailTestSuite) TestJailFetch() {
// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 		w.Write([]byte("Hello World"))
// 	})

// 	server := httptest.NewServer(mux)
// 	defer server.Close()

// 	require := s.Require()
// 	require.NotNil(s.jail)

// 	newCell, err := s.jail.NewJailCell(testChatID)
// 	require.NoError(err)
// 	require.NotNil(newCell)

// 	jcell, ok := newCell.(*jail.JailCell)
// 	require.Equal(ok, true)
// 	require.NotNil(jcell)

// 	wait := make(chan struct{})

// 	// Attempt to run a fetch resource.
// 	_, err = jcell.Fetch(server.URL, func(res otto.Value) {
// 		go func() { wait <- struct{}{} }()
// 	})

// 	require.NoError(err)

// 	<-wait
// }

func (s *JailTestSuite) TestJailLoopInCall() {
	require := s.Require()
	require.NotNil(s.jail)

	s.StartTestNode(params.RopstenNetworkID)
	defer s.StopTestNode()

	// load Status JS and add test command to it
	s.jail.BaseJS(baseStatusJSCode)
	s.jail.Parse(testChatID, ``)

	cell, err := s.jail.GetCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)

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

	// NOTE: Fails to work because it's running intirely

	_, err = cell.CallOnLoop("callRunner", nil, "softball")
	require.NoError(err)

	select {
	case received := <-items:
		require.Equal(received, "softball")
		break

	case <-time.After(5 * time.Second):
		require.Fail("Failed to received event response")
	}
}
