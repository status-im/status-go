package debug_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
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
	assert.Equal(replies[0], "[0] nil")

	commandLine = "StopNode()"
	replies = sendCommandLine(assert, conn, commandLine)
	assert.Len(replies, 1)
	assert.Equal(replies[0], "[0] nil")
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
	buf := bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)
	_, err := buf.WriteString(commandLine)
	assert.NoError(err)
	countStr, err := buf.ReadString('\n')
	assert.NoError(err)
	count, err := strconv.Atoi(countStr)
	assert.NoError(err)
	replies := make([]string, count)
	for i := 0; i < count; i++ {
		reply, err := buf.ReadString('\n')
		assert.NoError(err)
		replies[i] = reply
	}
	return replies
}
