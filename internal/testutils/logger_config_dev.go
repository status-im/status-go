//go:build !test_silent

package testutils

import "go.uber.org/zap"

func loggerConfig() zap.Config {
	return zap.NewDevelopmentConfig()
}
