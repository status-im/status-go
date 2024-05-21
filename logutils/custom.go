package logutils

import (
	"fmt"

	"go.uber.org/zap"
)

func WakuMessageTimestamp(key string, value *int64) zap.Field {
	valueStr := "-"
	if value != nil {
		valueStr = fmt.Sprintf("%d", *value)
	}
	return zap.String(key, valueStr)
}
