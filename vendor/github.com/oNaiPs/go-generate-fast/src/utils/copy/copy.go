package copy

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/blake2b"
)

func CopyFile(srcName, destName string) error {
	srcFile, err := os.Open(srcName)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destName)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

func CopyHashFile(srcName, destName string) (string, error) {
	srcFile, err := os.Open(srcName)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destName)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	h, err := blake2b.New256(nil)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(io.MultiWriter(destFile, h), srcFile); err != nil {
		return "", err
	}

	err = destFile.Sync()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
