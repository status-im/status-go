package logutils

import (
	"github.com/ethereum/go-ethereum/log"
)

// Logger returns the main logger instance used by status-go.
func Logger() log.Logger {
	return log.Root()
}
