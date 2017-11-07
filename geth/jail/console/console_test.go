package console_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/console"
	"github.com/status-im/status-go/geth/signal"
	"github.com/stretchr/testify/suite"
)

// TestConsole validates the behaviour of giving conole extensions.
func TestConsole(t *testing.T) {
	suite.Run(t, new(ConsoleTestSuite))
}

type ConsoleTestSuite struct {
	suite.Suite
	vm *otto.Otto
}

func (s *ConsoleTestSuite) SetupTest() {
	require := s.Require()

	vm := otto.New()
	require.NotNil(vm)
	s.vm = vm
}

// TestConsoleLog will validate the operations of the console.log extension
// for the otto vm.
func (s *ConsoleTestSuite) TestConsoleLog() {
	require := s.Require()
	written := "Bob Marley"

	var customWriter bytes.Buffer

	err := s.vm.Set("console", map[string]interface{}{
		"log": func(fn otto.FunctionCall) otto.Value {
			return console.Write(fn, &customWriter, "vm.console")
		},
	})
	require.NoError(err)

	_, err = s.vm.Run(fmt.Sprintf(`console.log(%q);`, written))
	require.NoError(err)
	require.Equal(written, strings.TrimPrefix(customWriter.String(), "vm.console: "))
}

// TestObjectLogging will validate the operations of the console.log extension
// when capturing objects declared from javascript.
func (s *ConsoleTestSuite) TestObjectLogging() {
	require := s.Require()

	var customWriter bytes.Buffer

	signal.SetDefaultNodeNotificationHandler(func(event string) {
		var eventReceived struct {
			Type  string `json:"type"`
			Event []struct {
				Age  int    `json:"age"`
				Name string `json:"name"`
			} `json:"event"`
		}

		err := json.Unmarshal([]byte(event), &eventReceived)
		require.NoError(err)

		require.Equal(eventReceived.Type, "vm.console")
		require.NotEmpty(eventReceived.Event)

		objectReceived := eventReceived.Event[0]
		require.Equal(objectReceived.Age, 24)
		require.Equal(objectReceived.Name, "bob")
	})

	err := s.vm.Set("console", map[string]interface{}{
		"log": func(fn otto.FunctionCall) otto.Value {
			return console.Write(fn, &customWriter, "vm.console")
		},
	})
	require.NoError(err)

	_, err = s.vm.Run(`
		var person = {name:"bob", age:24}
		console.log(person);
	`)
	require.NoError(err)
	require.NotEmpty(&customWriter)
}
