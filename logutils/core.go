package logutils

import (
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// encoderWrapper holds any zapcore.Encoder and ensures a consistent type for atomic.Value
type encoderWrapper struct {
	zapcore.Encoder
}

// writeSyncerWrapper holds any zapcore.WriteSyncer and ensures a consistent type for atomic.Value
type writeSyncerWrapper struct {
	zapcore.WriteSyncer
}

// Core wraps a zapcore.Core that can update its syncer and encoder at runtime
type Core struct {
	encoder atomic.Value  // encoderWrapper
	syncer  *atomic.Value // writeSyncerWrapper
	level   zap.AtomicLevel

	next       *Core
	nextFields []zapcore.Field
}

var (
	_ zapcore.Core = (*Core)(nil)
)

func NewCore(encoder zapcore.Encoder, syncer zapcore.WriteSyncer, atomicLevel zap.AtomicLevel) *Core {
	core := &Core{
		syncer: &atomic.Value{},
		level:  atomicLevel,
	}
	core.encoder.Store(encoderWrapper{Encoder: encoder})
	core.syncer.Store(writeSyncerWrapper{WriteSyncer: syncer})
	return core
}

func (core *Core) getEncoder() zapcore.Encoder {
	return core.encoder.Load().(zapcore.Encoder)
}

func (core *Core) getSyncer() zapcore.WriteSyncer {
	return core.syncer.Load().(zapcore.WriteSyncer)
}

func (core *Core) Enabled(lvl zapcore.Level) bool {
	return core.level.Enabled(lvl)
}

func (core *Core) Level() zapcore.Level {
	return core.level.Level()
}

func (core *Core) SetLevel(lvl zapcore.Level) {
	core.level.SetLevel(lvl)
}

func (core *Core) With(fields []zapcore.Field) zapcore.Core {
	clonedEncoder := encoderWrapper{Encoder: core.getEncoder().Clone()}
	for i := range fields {
		fields[i].AddTo(clonedEncoder)
	}

	clone := *core
	clone.encoder.Store(clonedEncoder)

	core.next = &clone
	core.nextFields = fields

	return &clone
}

func (core *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.Enabled(ent.Level) {
		return ce.AddCore(ent, core)
	}
	return ce
}

func (core *Core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := core.getEncoder().EncodeEntry(ent, fields)
	if err != nil {
		return err
	}
	_, err = core.getSyncer().Write(buf.Bytes())
	buf.Free()
	if err != nil {
		return err
	}

	if ent.Level > zapcore.ErrorLevel {
		// Since we may be crashing the program, sync the output.
		_ = core.Sync()
	}

	return err
}

func (core *Core) Sync() error {
	return core.getSyncer().Sync()
}

func (core *Core) UpdateSyncer(newSyncer zapcore.WriteSyncer) {
	core.syncer.Store(writeSyncerWrapper{WriteSyncer: newSyncer})
}

func (core *Core) UpdateEncoder(newEncoder zapcore.Encoder) {
	core.encoder.Store(encoderWrapper{Encoder: newEncoder})

	// Update next Cores with newEncoder
	current := core
	for current.next != nil {
		clonedEncoder := encoderWrapper{Encoder: core.getEncoder().Clone()}
		for i := range core.nextFields {
			current.nextFields[i].AddTo(clonedEncoder)
		}
		current.next.encoder.Store(clonedEncoder)
		current = current.next
	}
}
