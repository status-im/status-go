package reliability

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/sds-go-bindings/sds"

	"github.com/status-im/status-go/internal/testutils"
)

func TestSdsFileDescriptors(t *testing.T) {
	t.Skip("test demonstrates issue #7216")

	if !isLsofAvailable() {
		t.Skip("lsof command not available")
	}

	t.Logf("process id: %d", os.Getpid())

	a := logFileDescriptors(t, "before sds setup")

	logger := testutils.MustCreateTestLogger()
	reliabilityManager, err := sds.NewReliabilityManager(logger)
	require.NoError(t, err)
	_ = logFileDescriptors(t, "sds setup complete")

	t.Cleanup(func() {
		err = reliabilityManager.Cleanup()
		assert.NoError(t, err)
		b := logFileDescriptors(t, "sds cleanup complete")

		assert.Equalf(t, a, b, "nubmer of opened file descriptors after sds cleauup is not the same as before sds setup")
	})
}

func TestMultipleSdsFileDescriptors(t *testing.T) {
	if !isLsofAvailable() {
		t.Skip("lsof command not available")
	}

	t.Logf("process id: %d", os.Getpid())

	_ = logFileDescriptors(t, "before sds setup")

	logger := testutils.MustCreateTestLogger()
	rm1, err := sds.NewReliabilityManager(logger)
	require.NoError(t, err)
	_ = logFileDescriptors(t, "sds 1 setup complete")

	err = rm1.Cleanup()
	require.NoError(t, err)
	b := logFileDescriptors(t, "sds 1 cleanup complete")

	// Create one more Reliability Manager
	rm2, err := sds.NewReliabilityManager(logger)
	require.NoError(t, err)
	_ = logFileDescriptors(t, "sds 2 setup complete")

	// Cleanup the second Reliability Manager
	err = rm2.Cleanup()
	require.NoError(t, err)
	c := logFileDescriptors(t, "sds 2 cleanup complete")

	// Expect
	require.Equalf(t, b, c, "sds file descriptors should be reused")
}

func isLsofAvailable() bool {
	_, err := exec.LookPath("lsof")
	return err == nil
}

func logFileDescriptors(t *testing.T, label string) int {
	pid := os.Getpid()
	// pid comes from os.Getpid, safe for exec. Suppress gosec false-positive.
	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid)) //nolint:gosec

	output, err := cmd.Output()
	assert.NoError(t, err)

	message := string(output)
	count := len(strings.Split(message, "\n")) - 1

	t.Logf("--- %s --- file descriptors: %d", label, count)
	t.Logf("%s", message)

	return count
}
