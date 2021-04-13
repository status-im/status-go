package anonmetrics

import (
	"crypto/ecdsa"

	"go.uber.org/zap"

	"github.com/status-im/status-go/appmetrics"
)

type ClientConfig struct {
	ShouldSend bool
	SendAddress *ecdsa.PublicKey
}

type Client struct {
	Config *ClientConfig
	DB *appmetrics.Database
	Identity *ecdsa.PrivateKey
	Logger *zap.Logger
}

// TODO implement start functionality

// TODO implement stop functionality