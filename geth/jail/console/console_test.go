package console_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/jail/console"
	"github.com/status-im/status-go/geth/node"
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
	vm := otto.New()
	require := s.Require()

	require.NotNil(vm)
	require.IsType(&otto.Otto{}, vm)

	s.vm = vm
}

// TestConsoleLog will validate the operations of the console.log extension
// for the otto vm.
func (s *ConsoleTestSuite) TestConsoleLog() {
	require := s.Require()
	written := "Bob Marley"

	var customWriter bytes.Buffer

	// register console handler.
	err := s.vm.Set("console", map[string]interface{}{
		"log": console.Extension(&customWriter, console.Log),
	})
	require.NoError(err)

	_, err = s.vm.Run(fmt.Sprintf(`console.log(%q);`, written))
	require.NoError(err)

	require.NotEmpty(&customWriter)
	require.Equal(written, strings.TrimPrefix(customWriter.String(), "console.log: "))
}

// TestObjectLogging will validate the operations of the console.log extension
// when capturing objects declared from javascript.
func (s *ConsoleTestSuite) TestObjectLogging() {
	require := s.Require()

	node.SetDefaultNodeNotificationHandler(func(event string) {

		var eventReceived struct {
			Type  string `json:"type"`
			Event []struct {
				Age  int    `json:"age"`
				Name string `json:"name"`
			} `json:"event"`
		}

		err := json.Unmarshal([]byte(event), &eventReceived)
		require.NoError(err)

		require.Equal(eventReceived.Type, "vm.console.log")
		require.NotEmpty(eventReceived.Event)

		objectReceived := eventReceived.Event[0]
		require.Equal(objectReceived.Age, 24)
		require.Equal(objectReceived.Name, "bob")
	})

	var customWriter bytes.Buffer

	// register console handler.
	err := s.vm.Set("console", map[string]interface{}{
		"log": console.Extension(&customWriter, console.Log),
	})
	require.NoError(err)

	_, err = s.vm.Run(`
		var person = {name:"bob", age:24}
		console.log(person);
	`)
	require.NoError(err)
	require.NotEmpty(&customWriter)
}
