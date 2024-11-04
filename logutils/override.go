package logutils

import (
	"io"
	"os"

	logging "github.com/ipfs/go-log/v2"

	"go.uber.org/zap/zapcore"
)

type LogSettings struct {
	Enabled         bool   `json:"Enabled"`
	MobileSystem    bool   `json:"MobileSystem"`
	Level           string `json:"Level"`
	File            string `json:"File"`
	MaxSize         int    `json:"MaxSize"`
	MaxBackups      int    `json:"MaxBackups"`
	CompressRotated bool   `json:"CompressRotated"`
	Colorized       bool   `json:"Colorized"` // FIXME: doesn't take effect
}

func OverrideRootLoggerWithConfig(settings LogSettings) error {
	return overrideCoreWithConfig(ZapLogger().Core().(*Core), settings)
}

func overrideCoreWithConfig(core *Core, settings LogSettings) error {
	if !settings.Enabled {
		core.UpdateSyncer(zapcore.AddSync(io.Discard))
		return nil
	}

	if settings.Level == "" {
		settings.Level = "info"
	}
	level, err := lvlFromString(settings.Level)
	if err != nil {
		return err
	}
	core.SetLevel(level)

	if settings.MobileSystem {
		core.UpdateSyncer(zapcore.Lock(os.Stdout))
		return nil
	}

	if settings.File != "" {
		if settings.MaxBackups == 0 {
			// Setting MaxBackups to 0 causes all log files to be kept. Even setting MaxAge to > 0 doesn't fix it
			// Docs: https://pkg.go.dev/gopkg.in/natefinch/lumberjack.v2@v2.0.0#readme-cleaning-up-old-log-files
			settings.MaxBackups = 1
		}
		core.UpdateSyncer(ZapSyncerWithRotation(FileOptions{
			Filename:   settings.File,
			MaxSize:    settings.MaxSize,
			MaxBackups: settings.MaxBackups,
			Compress:   settings.CompressRotated,
		}))
	} else {
		core.UpdateSyncer(zapcore.Lock(os.Stderr))
	}

	// FIXME: remove go-libp2p logging altogether
	// go-libp2p logger
	{
		lvl, err := logging.LevelFromString(settings.Level)
		if err != nil {
			return err
		}
		logging.SetAllLoggers(lvl)
	}

	return nil
}
