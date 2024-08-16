package package_2

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/flaky-packages/utils"
)

func FailIfNoFile(t *testing.T, filename string) {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		require.NoError(t, err)
		return
	}

	err = os.WriteFile(filename, []byte("test"), 0600)
	require.NoError(t, err)

	utils.LogFlakiness()
	t.Fatal("file created: ", filename)
}

func TestSleep(t *testing.T) {
	utils.Sleep()
	FailIfNoFile(t, "test-1.txt")
	//FailIfNoFile(t, "test-2.txt")
	utils.Foo()
}
