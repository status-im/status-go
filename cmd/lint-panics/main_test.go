package main

import (
	"testing"
	gopls2 "github.com/status-im/status-go/cmd/lint-panics/gopls"
	"github.com/status-im/status-go/cmd/lint-panics/processor"
	"time"
	"context"
	"go.uber.org/zap"
	"runtime"
	"path"
	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	loggerConfig.Development = false
	loggerConfig.DisableStacktrace = true
	logger := zap.Must(loggerConfig.Build())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_, filePath, _, _ := runtime.Caller(0)
	testDir := path.Join(path.Dir(filePath), "test")
	testFie := path.Join(testDir, "test.go")

	gopls := gopls2.NewGoplsClient(ctx, logger, testDir)
	parser := processor.NewProcessor(logger, gopls)

	result, err := parser.Run(ctx, testFie)
	require.NoError(t, err)

	paths := result.Paths()
	require.Len(t, paths, 5)

	expectedPaths := []string{
		path.Join(testDir, "test.go:39"),
		path.Join(testDir, "test.go:47"),
		path.Join(testDir, "test.go:55"),
		path.Join(testDir, "test.go:63"),
		path.Join(testDir, "test.go:27"),
	}
	require.EqualValues(t, expectedPaths, paths)
}
