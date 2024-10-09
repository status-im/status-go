package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"path"
)

func main() {
	// Initialize logger with colors
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	loggerConfig.Development = false
	loggerConfig.DisableStacktrace = true
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}

	logger = logger.Named("main")

	if len(os.Args) < 2 {
		logger.Error("Usage: go run main.go <directory>")
		return
	}

	dir := os.Args[1]
	logger.Info("starting analysis...", zap.String("directory", dir))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	gopls := NewGoplsClient(ctx, logger)
	parser := NewParser(logger, gopls)

	// Step 1: Scan all files and look for `go` calls
	err = filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("failed to walk path", zap.String("path", dir), zap.Error(err))
			return err
		}
		if info.IsDir() {
			return nil
		}
		vendorPath := path.Join(dir, "vendor")
		if strings.HasPrefix(filePath, vendorPath) {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		_, err = parser.Run(filePath)
		if err != nil {
			return err
		}

		//checkFileForGoroutines(path, definition, logger)

		return nil
	})

	if err != nil {
		logger.Error("error during file walk", zap.Error(err))
	}

	logger.Info("analysis complete")
}
