package logutils

import (
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/protocol/zaputil"
)

var (
	_zapLogger     *zap.Logger
	_initZapLogger sync.Once
)

func ZapLogger() *zap.Logger {
	_initZapLogger.Do(func() {
		_zapLogger = defaultLogger()

		// forward geth logs to zap logger
		_gethLogger := _zapLogger.Named("geth")
		log.Root().SetHandler(gethAdapter(_gethLogger))
	})
	return _zapLogger
}

func defaultLogger() *zap.Logger {
	core := NewCore(
		defaultEncoder(),
		zapcore.AddSync(io.Discard),
		zap.NewAtomicLevelAt(zap.InfoLevel),
	)
	return zap.New(core, zap.AddCaller())
}

func defaultEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = utcTimeEncoder(encoderConfig.EncodeTime)

	return zaputil.NewConsoleHexEncoder(encoderConfig)
}

func utcTimeEncoder(encoder zapcore.TimeEncoder) zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		encoder(t.UTC(), enc)
	}
}
