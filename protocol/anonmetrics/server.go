package anonmetrics

import "go.uber.org/zap"

type ServerConfig struct {
	Enabled bool
}

type Server struct {
	Config *ServerConfig
	Logger *zap.Logger
}

// TODO implement start functionality

// TODO implement stop functionality
