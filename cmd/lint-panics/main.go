package main

import (
	"os"
	"time"
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gopls2 "github.com/status-im/status-go/cmd/lint-panics/gopls"
	"github.com/status-im/status-go/cmd/lint-panics/processor"
	"golang.org/x/tools/go/analysis/singlechecker"
	"golang.org/x/tools/go/analysis"
	"go/ast"
	"fmt"
	"path/filepath"
	"strings"
)

func main() {
	logger := buildLogger()

	if len(os.Args) == 0 {
		logger.Error("Usage: go run main.go <directory>")
		os.Exit(1)
	}

	// Dirty hack to get the root directory for gopls
	lastArg := os.Args[len(os.Args)-1]
	dir, err := getRootAbsolutePath(lastArg)
	if err != nil {
		logger.Error("failed to get root directory", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("starting analysis...", zap.String("directory", dir))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	gopls := gopls2.NewGoplsClient(ctx, logger, dir)

	analyzer := &analysis.Analyzer{
		Name: "logpanics",
		Doc:  "reports missing defer call to LogOnPanic",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			for _, file := range pass.Files {
				p := processor.NewProcessor(logger, pass, gopls)
				ast.Inspect(file, func(n ast.Node) bool {
					p.ProcessNode(ctx, n)
					return true
				})
			}
			return nil, nil
		},
	}

	singlechecker.Main(analyzer)
}

func buildLogger() *zap.Logger {
	// Initialize logger with colors
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	loggerConfig.Development = false
	loggerConfig.DisableStacktrace = true
	logger, err := loggerConfig.Build()
	if err != nil {
		fmt.Printf("failed to initialize logger: %s", err.Error())
		os.Exit(1)
	}

	return logger.Named("main")
}

func getRootAbsolutePath(path string) (string, error) {
	// Get the absolute path of the current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	if strings.HasSuffix(path, "...") {
		path = strings.TrimSuffix(path, "...")
	}

	// Check if the given path is absolute
	if !filepath.IsAbs(path) {
		// If the path is not absolute, join it with the working directory
		path = filepath.Join(workingDir, path)
	}

	// Convert the path to an absolute path (cleans the result)
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to convert to absolute path: %w", err)
	}

	return absolutePath, nil
}
