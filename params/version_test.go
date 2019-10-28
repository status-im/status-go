package params_test

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"testing"
)

func TestVersionFormat(t *testing.T) {
	data, err := ioutil.ReadFile(filepath.Join("..", "VERSION"))
	if err != nil {
		t.Error("unable to open VERSION file")
	}
	matched, _ := regexp.Match(`^\d+\.\d+\.\d+(-[.\w]+)?\n?$`, data)
	if !matched {
		t.Error("version in incorrect format")
	}
}
