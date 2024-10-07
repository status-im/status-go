package main

import (
	"os"
	"path/filepath"
	"strings"

	"time"
	"context"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger with colors
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	loggerConfig.Development = false
	loggerConfig.DisableStacktrace = true
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}

	logger = logger.Named("main")

	//handler := log.StreamHandler(os.Stdout, log.TerminalFormat(true))
	//log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, handler))

	if len(os.Args) < 2 {
		logger.Error("Usage: go run main.go <directory>")
		return
	}

	dir := os.Args[1]
	logger.Info("starting analysis...", zap.String("directory", dir))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	gopls := NewGoplsClient(ctx, logger)
	definition := func(filePath string, lineNumber int, charPosition int) (string, int, error) {
		if goplsHTTP {
			return gopls.definitionTCP(filePath, lineNumber, charPosition)
		} else {
			return definitionCLI(filePath, lineNumber, charPosition, logger)
		}
	}

	// Step 1: Scan all files and look for `go` calls
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("failed to walk path", zap.String("path", dir), zap.Error(err))
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(path, dir+"/vendor") {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		logger.Info("scanning Go file", zap.String("file", path))
		//content, err := os.ReadFile(path)
		//if err != nil {
		//	return err
		//}
		//gopls.DidOpen(ctx, path, string(content), logger)

		checkFileForGoroutines(path, definition, logger)
		//gopls.DidClose(ctx, path)

		return nil
	})

	if err != nil {
		logger.Error("error during file walk", zap.Error(err))
	}

	logger.Info("analysis complete")
}
