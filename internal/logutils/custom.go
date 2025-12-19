package logutils

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

func WakuMessageTimestamp(key string, value *int64) zap.Field {
	valueStr := "-"
	if value != nil {
		valueStr = fmt.Sprintf("%d", *value)
	}
	return zap.String(key, valueStr)
}

func UnixTimeMs(key string, t time.Time) zap.Field {
	return zap.String(key, fmt.Sprintf("%d", t.UnixMilli()))
}

func UnixTimeNano(key string, t time.Time) zap.Field {
	return zap.String(key, fmt.Sprintf("%d", t.UnixNano()))
}
