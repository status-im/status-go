package temp

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func resolveExecutablePath(executable string) (string, error) {
	// Support `go tool <exe>` by resolving the real tool path via `go tool -n`
	const goToolPrefix = "go tool "
	if strings.HasPrefix(executable, goToolPrefix) {
		tool := strings.TrimPrefix(executable, goToolPrefix)
		out, err := exec.Command("go", "tool", "-n", tool).Output()
		if err != nil {
			return "", fmt.Errorf("cannot resolve %q via 'go tool -n': %w", executable, err)
		}

		return strings.TrimSpace(string(out)), nil
	}

	// Use the executable path as-is by default
	return "", errors.New("nope")
}

func getExecutableDetails(ExecutablePath string) (string, error) {
	ExecutablePath, err := resolveExecutablePath(ExecutablePath)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(ExecutablePath)
	if err != nil {
		return "", err
	}

	execInfo := fmt.Sprint(
		ExecutablePath,
		fmt.Sprintf("%019d", info.Size()),
		info.ModTime().Format(time.RFC3339))

	zap.S().Debugf("Exec info %s", execInfo)

	return execInfo, nil
}

func TestExecutableFileInfoGoTool(t *testing.T) {
	info, err := getExecutableDetails("go tool compile")
	assert.NoError(t, err)
	assert.NotEmpty(t, info)
}
