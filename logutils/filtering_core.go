package logutils

import "go.uber.org/zap/zapcore"

type filterFunc func(ent zapcore.Entry) bool

type filteringCore struct {
	parent zapcore.Core

	levelEnabler zapcore.LevelEnabler
	filterFunc   filterFunc
}

var (
	_ zapcore.Core = (*filteringCore)(nil)
)

func newFilteringCore(core zapcore.Core, levelEnabler zapcore.LevelEnabler, filterFunc filterFunc) *filteringCore {
	return &filteringCore{
		parent:       core,
		levelEnabler: levelEnabler,
		filterFunc:   filterFunc,
	}
}

func (core *filteringCore) Enabled(lvl zapcore.Level) bool {
	return core.levelEnabler.Enabled(lvl)
}

func (core *filteringCore) With(fields []zapcore.Field) zapcore.Core {
	return &filteringCore{
		parent:       core.parent.With(fields),
		levelEnabler: core.levelEnabler,
		filterFunc:   core.filterFunc,
	}
}

func (core *filteringCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.filterFunc(ent) {
		return ce.AddCore(ent, core)
	}
	return ce
}

func (core *filteringCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return core.parent.Write(ent, fields)
}

func (core *filteringCore) Sync() error {
	return core.parent.Sync()
}

func (core *filteringCore) Parent() zapcore.Core {
	return core.parent
}
