package wav

import (
	"os"
	"path"
	"runtime"
)

func fixture(basename string) string {
	_, filename, _, _ := runtime.Caller(1)

	return path.Join(path.Dir(filename), "files", basename)
}

func fixtureFile(basename string) (file *os.File, err error) {
	file, err = os.Open(fixture(basename))

	return
}
