package pkg

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/tools/go/packages"
)

func LoadPackages(pwd string, patterns []string, tags []string) *packages.Package {
	cfg := &packages.Config{
		Mode:       packages.NeedCompiledGoFiles,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(tags, " "))},
		Dir:        pwd,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		zap.S().Fatal(err)
	}
	if len(pkgs) != 1 {
		zap.S().Fatalf("error: %d packages found", len(pkgs))
	}

	return pkgs[0]
}
