package utils

import (
	"strconv"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap"
	gotoken "go/token"
)

func PositionURI(pos gotoken.Position) string {
	return URI(pos.Filename, pos.Line)
}

func URI(path string, line int) string {
	return path + ":" + strconv.Itoa(line)
}

func ZapURI(path string, line int) zap.Field {
	return zap.Field{
		Type:   zapcore.StringType,
		Key:    "uri",
		String: URI(path, line),
	}
}
