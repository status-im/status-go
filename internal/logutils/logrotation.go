package logutils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// processStartTime represents the start time of this process
var processStartTime = time.Now()

// sessionArchiveTimeFormat must equal lumberjack's backupTimeFormat, different naming conventions will be invisible to lumberjack's cleanup.
const sessionArchiveTimeFormat = "2006-01-02T15-04-05.000"

// legacySessionArchiveRegex matches legacy archives whose "Z" suffix lumberjack cannot parse.
var legacySessionArchiveRegex = regexp.MustCompile(`^(.+)-(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2})Z$`)

// FileOptions are all options supported by internal rotation module.
type FileOptions struct {
	// Base name for log file.
	Filename string
	// Size in megabytes.
	MaxSize int
	// Number of rotated log files.
	MaxBackups int
	// If true rotated log files will be gzipped.
	Compress bool
}

// ZapSyncerWithRotation creates a zapcore.WriteSyncer with a configured rotation
func ZapSyncerWithRotation(opts FileOptions) zapcore.WriteSyncer {
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   opts.Filename,
		MaxSize:    opts.MaxSize,
		MaxBackups: opts.MaxBackups,
		Compress:   opts.Compress,
	})
}

// rotateLogFileForNewSession renames an existing log file so the new process writes to a fresh file.
func rotateLogFileForNewSession(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.ModTime().Before(processStartTime) {
		return nil
	}
	ts := info.ModTime().UTC().Format(sessionArchiveTimeFormat)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	archived := base + "-" + ts + ext
	return os.Rename(path, archived)
}

// renameLegacySessionArchives renames legacy session archives created with the "...Z" timestamp suffix to sessionArchiveTimeFormat
// so lumberjack counts them against MaxBackups and eventually prunes them.
func renameLegacySessionArchives(path string) error {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var firstErr error
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ext {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ext)
		m := legacySessionArchiveRegex.FindStringSubmatch(name)
		if m == nil || m[1] != base {
			continue
		}
		newPath := filepath.Join(dir, m[1]+"-"+m[2]+".000"+ext)
		if _, err := os.Stat(newPath); err == nil {
			// Target exists (another legacy archive already claimed this second).
			continue
		}
		if err := os.Rename(filepath.Join(dir, entry.Name()), newPath); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
