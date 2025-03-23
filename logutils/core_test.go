package logutils

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestCore(t *testing.T) {
	level := zap.NewAtomicLevelAt(zap.DebugLevel)

	buffer1 := bytes.NewBuffer(nil)
	buffer2 := bytes.NewBuffer(nil)

	core := NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(buffer1),
		level,
	)

	parent := zap.New(core)
	child := parent.Named("child")
	childWithContext := child.With(zap.String("key1", "value1"))
	childWithMoreContext := childWithContext.With(zap.String("key2", "value2"))
	grandChild := childWithMoreContext.Named("grandChild")

	print := func() {
		parent.Debug("Status")
		child.Debug("Super")
		childWithContext.Debug("App")
		childWithMoreContext.Debug("The")
		grandChild.Debug("Best")
	}

	// Ensure that the first buffer has the console encoder output
	print()
	buffer1Lines := strings.Split(buffer1.String(), "\n")
	require.Len(t, buffer1Lines, 5+1)
	require.Regexp(t, `\s+child\s+`, buffer1Lines[1])
	require.Regexp(t, `\s+child\.grandChild\s+`, buffer1Lines[4])

	// Ensure syncer was updated
	core.UpdateSyncer(zapcore.AddSync(buffer2))
	print()
	require.Equal(t, buffer1, buffer2)
}

func benchmarkCore(b *testing.B, core zapcore.Core) {
	logger := zap.New(core)

	messageQueue := make(chan int, b.N)
	for i := 0; i < b.N; i++ {
		messageQueue <- i
	}
	close(messageQueue)

	b.ResetTimer()

	wg := sync.WaitGroup{}
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range messageQueue {
				logger.Debug("Benchmark message", zap.Int("i", i))
			}
		}()
	}

	wg.Wait()
	err := logger.Sync()
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkCustomCore(b *testing.B) {
	benchmarkCore(b, NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(bytes.NewBuffer(nil))),
		zap.NewAtomicLevelAt(zap.DebugLevel)),
	)
}

func BenchmarkZapCore(b *testing.B) {
	benchmarkCore(b, zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(bytes.NewBuffer(nil))),
		zap.NewAtomicLevelAt(zap.DebugLevel)),
	)
}
