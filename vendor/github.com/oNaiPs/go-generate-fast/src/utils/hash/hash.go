package hash

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/blake2b"
)

func HashString(message string) (string, error) {
	h, err := blake2b.New256(nil)
	if err != nil {
		return "", err
	}

	h.Write([]byte(message))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func HashFile(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h, err := blake2b.New256(nil)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
