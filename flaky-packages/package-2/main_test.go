package package_2

import (
	"github.com/status-im/status-go/flaky-packages/utils"
	"testing"
	"os"
	"github.com/stretchr/testify/require"
)

func FailIfNoFile(t *testing.T, filename string) {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		require.NoError(t, err)
		return
	}

	err = os.WriteFile(filename, []byte("test"), 0644)
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
