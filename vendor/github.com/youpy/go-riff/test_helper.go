package riff

import (
	"os"
	"path"
	"runtime"
)

type testFile struct {
	Name      string
	ChunkSize int
	FileSize  uint32
	FileType  string
}

func fixture(basename string) string {
	_, filename, _, _ := runtime.Caller(1)

	return path.Join(path.Dir(filename), "files", basename)
}

func fixtureFile(basename string) (file *os.File, err error) {
	file, err = os.Open(fixture(basename))

	return
}
