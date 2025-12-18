package logutils

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNamespaceFilteringCore(t *testing.T) {
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	buffer := bytes.NewBuffer(nil)

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(buffer),
		level,
	)
	filteringCore := newNamespaceFilteringCore(core)
	logger := zap.New(filteringCore)

	err := filteringCore.Rebuild("namespaceA:error,namespaceA.namespaceB:debug")
	require.NoError(t, err)

	logger.Info("one")                                                                // OK
	logger.Debug("two")                                                               // not OK
	logger.Named("unregistered").Info("three")                                        // OK
	logger.Named("unregistered").Debug("four")                                        // not OK
	logger.Named("namespaceA").Error("five")                                          // OK
	logger.Named("namespaceA").Info("six")                                            // not OK
	logger.Named("namespaceA").Named("unregistered").Error("seven")                   // OK
	logger.Named("namespaceA").Named("unregistered").Info("eight")                    // not OK
	logger.Named("namespaceA").Named("namespaceB").Debug("nine")                      // OK
	logger.Named("namespaceA").Named("namespaceB").Named("unregistered").Debug("ten") // OK

	require.Contains(t, buffer.String(), "one", "three", "five", "seven", "nine", "ten")
	require.NotContains(t, buffer.String(), "two", "four", "six", "eight")

	err = filteringCore.Rebuild("") // Remove filtering
	require.NoError(t, err)
	buffer.Reset()

	logger.Info("one")                                          // OK
	logger.Named("namespaceA").Named("namespaceB").Debug("two") // not OK

	require.Contains(t, buffer.String(), "one")
	require.NotContains(t, buffer.String(), "two")
}

func generateNamespaces(n, depth int, level string) (namespaces string, deepestNamespace string) {
	namespace := ""
	for i := 0; i < n; i++ {
		namespace = strconv.Itoa(i)
		for d := 0; d < depth; d++ {
			namespace += "." + strconv.Itoa(d)
			namespaces += namespace + ":" + level + ","
		}
	}
	namespaces = strings.TrimSuffix(namespaces, ",")
	deepestNamespace = namespace

	return namespaces, deepestNamespace
}

func benchmarkNamespaces(b *testing.B, core zapcore.Core, setupFilter func(namespaces string) error) {
	// Creates complex namespaces filter:
	// "0.0:info,0.0.1:info,0.0.1.2:info,0.0.1.2.3:info,1.0:info,1.0.1:info,1.0.1.2:info,1.0.1.2.3:info,2.0:info,2.0.1:info,2.0.1.2:info,2.0.1.2.3:info"
	const n = 3
	const depth = 4
	namespaces, deepestNamespace := generateNamespaces(n, depth, zapcore.LevelOf(core).String())

	err := setupFilter(namespaces)
	if err != nil {
		b.Fatal(err)
	}

	rootLogger := zap.New(core)
	deepestLogger := rootLogger.Named(deepestNamespace)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for level := zap.DebugLevel; level <= zap.ErrorLevel; level++ {
			rootLogger.Check(level, "Benchmark message").Write(zap.Int("i", i))
			deepestLogger.Check(level, "Benchmark message").Write(zap.Int("i", i))
		}
	}
}

func BenchmarkNamespacesFilteringCore(b *testing.B) {
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(bytes.NewBuffer(nil))),
		zap.NewAtomicLevelAt(zap.InfoLevel),
	)
	filteringCore := newNamespaceFilteringCore(core)

	benchmarkNamespaces(b, filteringCore, filteringCore.Rebuild)
}

func BenchmarkNamespacesZapCore(b *testing.B) {
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(bytes.NewBuffer(nil))),
		zap.NewAtomicLevelAt(zap.InfoLevel),
	)

	benchmarkNamespaces(b, core, func(string) error { return nil })
}
