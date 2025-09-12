package golist

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

type PackageError struct {
	ImportStack []string // shortest path from package named on command line to this one
	Pos         string   // position of error (if present, file:line:col)
	Err         string   // the error itself
}

type Package struct {
	Dir     string   // directory containing package sources
	GoFiles []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	// Error information
	Incomplete bool          // this package or a dependency has an error
	Error      *PackageError // error loading package
}

type PkgError struct {
	Error   *string
	Package string
}

// TODO see alternative
// https://github.com/uber-go/mock/blob/fcaca4af4e64b707bdb0773ec92441c524bce3d0/mockgen/mockgen.go#L836

func ModulesAndErrors(args []string) []PkgError {
	if len(args) == 0 {
		args = []string{"./..."}
	}

	cmd := exec.Command("go", append([]string{
		"list", "-e", "-json=GoFiles,Dir,Incomplete,Error,DepsErrors",
	}, args...)...)

	zap.S().Debug("Running command: ", strings.Join(cmd.Args, " "))

	// Get the output of the command
	output, err := cmd.Output()
	if err != nil {
		zap.S().Error("Error executing command: ", err)
		strError := err.Error()
		return []PkgError{{Error: &strError}}
	}

	zap.S().Debug("go list output:\n", string(output))

	dec := json.NewDecoder(strings.NewReader(string(output)))

	pkgErrors := []PkgError{}

	for {
		var pkg Package

		err := dec.Decode(&pkg)
		if err == io.EOF {
			// all done
			break
		}
		if err != nil {
			zap.S().Error("Error parsing JSON: ", err)
			strError := err.Error()
			return []PkgError{{Error: &strError}}
		}

		if pkg.Error != nil {
			strError := fmt.Sprintf("%s %s", pkg.Error.Pos, pkg.Error.Err)
			pkgErrors = append(pkgErrors, PkgError{Error: &strError})
			continue
		}

		for _, file := range pkg.GoFiles {
			fileLocation := fmt.Sprintf("%s/%s", pkg.Dir, file)
			pkgErrors = append(pkgErrors, PkgError{Package: fileLocation})
		}
	}

	return pkgErrors
}
