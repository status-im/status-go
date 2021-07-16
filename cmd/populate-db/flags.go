package main

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// configFlags represents an array of JSON configuration files passed to a command line utility
type configFlags []string

func (f *configFlags) String() string {
	return strings.Join(*f, ", ")
}

func (f *configFlags) Set(value string) error {
	if !path.IsAbs(value) {
		// Convert to absolute path
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		value = path.Join(cwd, value)
	}

	// Check that the file exists
	stat, err := os.Stat(value)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("path does not represent a file: %s", value)
	}
	*f = append(*f, value)
	return nil
}
