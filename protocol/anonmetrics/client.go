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

type FibonacciIntervalIncrementer struct {
	Last    int
	Current int
}

func (f *FibonacciIntervalIncrementer) Next() int {
	out := f.Last + f.Current

	f.Last = f.Current
	f.Current = out

	return out
}
