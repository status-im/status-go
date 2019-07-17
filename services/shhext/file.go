package shhext

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// copyFile implementation is borrowed from https://go-review.googlesource.com/c/go/+/1591
// which didn't make into ioutil package.
// A slight modification is that the file permissions are copied from the source file.
// Another modification is the order of parameters which was reversed.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	inPerm, err := in.Stat()
	if err != nil {
		return err
	}

	tmp, err := ioutil.TempFile(filepath.Dir(dst), "")
	if err != nil {
		return err
	}

	_, err = io.Copy(tmp, in)
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}

	if err = tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}

	if err = os.Chmod(tmp.Name(), inPerm.Mode()); err != nil {
		os.Remove(tmp.Name())
		return err
	}

	return os.Rename(tmp.Name(), dst)
}
