package jail_test

import (
	"time"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/params"
)

func (s *JailTestSuite) TestJailTimeoutFailure() {
	require := s.Require()

	cell, err := s.jail.NewCell(testChatID)
	require.NoError(err)
	require.NotNil(cell)

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

	s.StartTestNode(params.RopstenNetworkID, true)
	defer s.StopTestNode()

	// load Status JS and add test command to it
	s.jail.BaseJS(baseStatusJSCode)
	s.jail.Parse(testChatID, ``)

	cell, err := s.jail.GetConcreteCell(testChatID)
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

	_, err = cell.Call("callRunner", nil, "softball")
	require.NoError(err)

	select {
	case received := <-items:
		require.Equal(received, "softball")
		break

	case <-time.After(5 * time.Second):
		require.Fail("Failed to received event response")
	}
}
