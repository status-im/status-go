package debug_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/status-im/status-go/cmd/statusd/debug"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/assert"
)

// TestStartStopNode tests starting and stopping a node remotely.
func TestStartStopNode(t *testing.T) {
	assert := assert.New(t)
	configJSON, cleanup, err := mkConfigJSON("status-start-stop-node")
	assert.NoError(err)
	defer cleanup()

	startDebugging(assert)

	conn := connectDebug(assert)

	commandLine := fmt.Sprintf("StartNode(%q)", configJSON)
	replies := sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal("[0] <nil>", replies[0])

	commandLine = "StopNode()"
	replies = sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal("[0] <nil>", replies[0])
}

// TestCreateAccount tests creating an account on the server.
func TestCreateAccount(t *testing.T) {
	assert := assert.New(t)
	configJSON, cleanup, err := mkConfigJSON("status-create-account")
	assert.NoError(err)
	defer cleanup()

	startDebugging(assert)

	conn := connectDebug(assert)

	commandLine := fmt.Sprintf("StartNode(%q)", configJSON)
	replies := sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal("[0] <nil>", replies[0])

	commandLine = fmt.Sprintf("CreateAccount(%q)", "password")
	replies = sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 4)
	assert.NotEqual("[0] <nil>", replies[0])
	assert.NotEqual("[1] <nil>", replies[1])
	assert.NotEqual("[2] <nil>", replies[2])
	assert.Equal("[3] <nil>", replies[3])

	commandLine = "StopNode()"
	replies = sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal("[0] <nil>", replies[0])
}

// TestSelectAccountLogout tests selecting an account on the server
// and logging out afterwards.
func TestSelectAccountLogout(t *testing.T) {
	assert := assert.New(t)
	configJSON, cleanup, err := mkConfigJSON("status-create-account")
	assert.NoError(err)
	defer cleanup()

	startDebugging(assert)

	conn := connectDebug(assert)

	commandLine := fmt.Sprintf("StartNode(%q)", configJSON)
	replies := sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal("[0] <nil>", replies[0])

	commandLine = fmt.Sprintf("CreateAccount(%q)", "password")
	replies = sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 4)
	assert.NotEqual("[0] <nil>", replies[0])
	assert.NotEqual("[1] <nil>", replies[1])
	assert.NotEqual("[2] <nil>", replies[2])
	assert.Equal("[3] <nil>", replies[3])

	address := replies[0][4:]

	commandLine = fmt.Sprintf("SelectAccount(%q, %q)", address, "password")
	replies = sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal("[0] <nil>", replies[0])

	commandLine = "Logout()"
	replies = sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal("[0] <nil>", replies[0])

	commandLine = "StopNode()"
	replies = sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal("[0] <nil>", replies[0])
}

//-----
// HELPERS
//-----

var (
	mu sync.Mutex
	d  *debug.Debug
)

// startDebugging lazily creates or reuses a debug instance.
func startDebugging(assert *assert.Assertions) {
	mu.Lock()
	defer mu.Unlock()
	if d == nil {
		var err error
		api := api.NewStatusAPI()
		d, err = debug.New(api)
		assert.NoError(err)
	}
}

// mkConfigJSON creates a configuration matching to
// a temporary directory and a cleanup for that directory.
func mkConfigJSON(name string) (string, func(), error) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), name)
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		os.RemoveAll(tmpDir) //nolint: errcheck
	}
	configJSON := `{
		"NetworkId": ` + strconv.Itoa(params.RopstenNetworkID) + `,
		"DataDir": "` + tmpDir + `",
		"LogLevel": "INFO",
		"RPCEnabled": true
	}`
	return configJSON, cleanup, nil
}

// connectDebug connects to the debug instance.
func connectDebug(assert *assert.Assertions) net.Conn {
	conn, err := net.Dial("tcp", ":51515")
	assert.NoError(err)
	return conn
}

// sendCommandLine sends a command line via the passed connection.
func sendCommandLine(assert *assert.Assertions, conn net.Conn, commandLine string) []string {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	_, err := writer.WriteString(commandLine + "\n")
	assert.NoError(err)
	err = writer.Flush()
	assert.NoError(err)
	countStr, err := reader.ReadString('\n')
	assert.NoError(err)
	count, err := strconv.Atoi(strings.TrimSuffix(countStr, "\n"))
	assert.NoError(err)
	replies := make([]string, count)
	for i := 0; i < count; i++ {
		reply, err := reader.ReadString('\n')
		assert.NoError(err)
		replies[i] = strings.TrimSuffix(reply, "\n")
	}
	return replies
}
