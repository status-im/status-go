package main

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/status-im/status-go/cmd/lint-panics/analyzer"
	"github.com/status-im/status-go/cmd/lint-panics/utils"
)

/*
	Run with `-root=<directory>` to specify the root directory to run gopls. Defaults to the current working directory.
	Set `-skip=<directory>` to skip errors in certain directories. If relative, it is relative to the root directory.

	If provided, `-root` and `-skip` arguments MUST go first, before any other args.
*/

func main() {
	logger := utils.BuildLogger(zap.ErrorLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	a, err := analyzer.New(ctx, logger)
	if err != nil {
		logger.Error("failed to create analyzer", zap.Error(err))
		os.Exit(1)
	}

	singlechecker.Main(a)
}
