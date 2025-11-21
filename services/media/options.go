package media

import (
	"go.uber.org/zap"
)

type Option func(*config)

func WithDisableTLS(disableTLS bool) Option {
	return func(s *config) {
		s.disableTLS = disableTLS
	}
}

func WithServerAddress(address string) Option {
	return func(s *config) {
		s.address = address
	}
}

func WithServerAdvertizeAddress(host string, port int) Option {
	return func(s *config) {
		s.advertizeHost = host
		s.advertizePort = port
	}
}

func WithLogger(logger *zap.Logger) Option {
	return func(s *config) {
		s.logger = logger
	}
}