package fs

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			zap.S().Errorf("Error checking path %s: %s\n", path, err)
			return false
		}
	}
	return fileInfo.IsDir()
}

func FindExecutablePath(executable string) (string, error) {
	if executable != "" && !strings.Contains(executable, string(os.PathSeparator)) {
		// Prefer to resolve the binary from GOROOT/bin, and for consistency
		// prefer to resolve any other commands there too.
		gorootBinPath, err := exec.LookPath(filepath.Join(runtime.GOROOT(), "bin", executable))
		if err == nil {
			return gorootBinPath, nil
		}
	}

	// resolve executable and use absolute path
	return exec.LookPath(executable)
}
