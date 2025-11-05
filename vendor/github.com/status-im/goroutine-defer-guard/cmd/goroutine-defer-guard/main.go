package main

import (
	"os"

	"go.uber.org/zap"
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/status-im/goroutine-defer-guard/cmd/goroutine-defer-guard/analyzer"
	"github.com/status-im/goroutine-defer-guard/cmd/goroutine-defer-guard/utils"
)

/*
	Set `-skip=<directory>` to skip errors in certain directories.
	If relative, it is relative to the current working directory.
*/

func main() {
	logger := utils.BuildLogger(zap.ErrorLevel)

	a, err := analyzer.New(logger)
	if err != nil {
		logger.Error("failed to create analyzer", zap.Error(err))
		os.Exit(1)
	}

	singlechecker.Main(a)
}
