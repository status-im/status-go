package main

import (
	"strconv"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap"
)

// atoi is a helper to safely convert a string to an int
func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func uri(path string, line int) string {
	return path + ":" + strconv.Itoa(line)
}

func ZapURI(path string, line int) zap.Field {
	return zap.Field{
		Type:   zapcore.StringType,
		Key:    "uri",
		String: uri(path, line),
	}
}
