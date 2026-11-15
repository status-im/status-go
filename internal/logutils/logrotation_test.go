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
	require.NoError(t, os.WriteFile(file, []byte("x"), 0600))
	// Stamp the mtime rather than relying on the write landing after processStartTime.
	// That assumption has almost no margin: processStartTime is set when this package is
	// initialised, and the test runs ~0.5ms later (~0.1ms when gotestsum --rerun-fails
	// re-runs it alone). Filesystem mtimes are coarser than time.Now() and can lag it by
	// more than that, so the file would stat as older than the process and be archived.
	active := processStartTime.Add(time.Minute)
	require.NoError(t, os.Chtimes(file, active, active))

	require.NoError(t, rotateLogFileForNewSession(file))

	_, err := os.Stat(file)
	require.NoError(t, err)
}

func TestRotateLogFileForNewSessionPreservesExistingArchives(t *testing.T) {
	for _, tc := range []struct {
		name     string
		archives []string
		next     time.Duration
	}{
		{name: "uncompressed", archives: []string{"2020-01-02T03-04-59.999.log"}, next: time.Millisecond},
		{name: "compressed", archives: []string{"2020-01-02T03-04-59.999.log.gz"}, next: time.Millisecond},
		{
			name: "consecutive collisions across a second boundary",
			archives: []string{
				"2020-01-02T03-04-59.999.log",
				"2020-01-02T03-05-00.000.log.gz",
				"2020-01-02T03-05-00.001.log",
			},
			next: 3 * time.Millisecond,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "test.log")
			old := time.Date(2020, time.January, 2, 3, 4, 59, 999000000, time.UTC)
			for _, archive := range tc.archives {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "test-"+archive), []byte("ARCHIVE_FROM_PREVIOUS_SESSION"), 0600))
			}
			require.NoError(t, os.WriteFile(file, []byte("NEW_ACTIVE_LOG"), 0600))
			require.NoError(t, os.Chtimes(file, old, old))

			require.NoError(t, rotateLogFileForNewSession(file))

			for _, archive := range tc.archives {
				content, err := os.ReadFile(filepath.Join(dir, "test-"+archive))
				require.NoError(t, err)
				require.Equal(t, "ARCHIVE_FROM_PREVIOUS_SESSION", string(content))
			}
			ts := old.Add(tc.next).Format(sessionArchiveTimeFormat)
			content, err := os.ReadFile(filepath.Join(dir, "test-"+ts+".log"))
			require.NoError(t, err)
			require.Equal(t, "NEW_ACTIVE_LOG", string(content))
			_, err = time.Parse(sessionArchiveTimeFormat, ts)
			require.NoError(t, err)
			_, err = os.Stat(file)
			require.True(t, os.IsNotExist(err))
			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Len(t, entries, len(tc.archives)+1)
		})
	}
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
	old := processStartTime.Add(-24 * time.Hour).Truncate(time.Millisecond)
	for range DefaultLogMaxBackups + 5 {
		require.NoError(t, os.WriteFile(file, []byte("old"), 0600))
		require.NoError(t, os.Chtimes(file, old, old))
		require.NoError(t, rotateLogFileForNewSession(file))
	}
	// All sessions must survive timestamp collisions and remain visible to cleanup.
	matches, err := filepath.Glob(filepath.Join(dir, "test-*.log"))
	require.NoError(t, err)
	require.Len(t, matches, DefaultLogMaxBackups+5)

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
		return err == nil && len(matches) == DefaultLogMaxBackups
	}, 5*time.Second, 100*time.Millisecond)
}
