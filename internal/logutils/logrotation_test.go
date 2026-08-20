package logutils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestRotateLogFileForNewSessionProducesLumberjackParseableName(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.log")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0600))
	old := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(file, old, old))

	require.NoError(t, rotateLogFileForNewSession(file))

	_, err := os.Stat(file)
	require.True(t, os.IsNotExist(err))

	matches, err := filepath.Glob(filepath.Join(dir, "test-*.log"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	// The archive suffix must parse with lumberjack's backupTimeFormat,
	// otherwise MaxBackups never prunes session archives.
	suffix := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(matches[0]), "test-"), ".log")
	_, err = time.Parse(sessionArchiveTimeFormat, suffix)
	require.NoError(t, err)
}

func TestRotateLogFileForNewSessionKeepsActiveFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.log")
	// Set mtime explicitly after processStartTime so this remains deterministic on
	// filesystems with coarse timestamp resolution.
	require.NoError(t, os.WriteFile(file, []byte("x"), 0600))
	activeTime := processStartTime.Add(2 * time.Second)
	require.NoError(t, os.Chtimes(file, activeTime, activeTime))

	require.NoError(t, rotateLogFileForNewSession(file))

	_, err := os.Stat(file)
	require.NoError(t, err)
}

func TestRenameLegacySessionArchives(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "pre_login.log")
	for _, name := range []string{
		"pre_login-2026-08-14T11-08-48Z.log",    // legacy, to be renamed
		"0x4d..2f40-2026-08-14T13-37-56Z.log",   // legacy, but different base: untouched
		"pre_login-2026-08-14T12-00-00Z.log",    // legacy, collides with existing target
		"pre_login-2026-08-14T12-00-00.000.log", // already in the new layout
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0600))
	}

	require.NoError(t, renameLegacySessionArchives(file))

	for name, expectExists := range map[string]bool{
		"pre_login-2026-08-14T11-08-48Z.log":    false,
		"pre_login-2026-08-14T11-08-48.000.log": true,
		"0x4d..2f40-2026-08-14T13-37-56Z.log":   true,
		"pre_login-2026-08-14T12-00-00Z.log":    true, // left alone due to collision
		"pre_login-2026-08-14T12-00-00.000.log": true,
	} {
		_, err := os.Stat(filepath.Join(dir, name))
		if expectExists {
			require.NoError(t, err, name)
		} else {
			require.True(t, os.IsNotExist(err), name)
		}
	}
}

func TestSessionArchivesPrunedToDefaultMaxBackups(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.log")
	base := time.Now().Add(-24 * time.Hour).UTC()
	for i := range DefaultLogMaxBackups + 5 {
		ts := base.Add(time.Duration(i) * time.Minute).Format(sessionArchiveTimeFormat)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "test-"+ts+".log"), []byte("old"), 0600))
	}

	core := NewCore(
		defaultEncoder(),
		zapcore.AddSync(io.Discard),
		zap.NewAtomicLevelAt(zap.InfoLevel),
	)
	filteringCore := newNamespaceFilteringCore(core)
	// MaxBackups left at 0 => DefaultLogMaxBackups fallback.
	require.NoError(t, overrideCoreWithConfig(filteringCore, LogSettings{
		Enabled: true,
		Level:   "info",
		File:    file,
	}))

	// Lumberjack prunes asynchronously on write.
	zap.New(filteringCore).Info("trigger mill")

	require.Eventually(t, func() bool {
		matches, err := filepath.Glob(filepath.Join(dir, "test-*.log"))
		return err == nil && len(matches) <= DefaultLogMaxBackups
	}, 5*time.Second, 100*time.Millisecond)
}
