package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClientHandlerTimeout(t *testing.T) {
	clientHandler := NewClientSideHandler()

	backupWalletResponseMaxInterval := WalletResponseMaxInterval
	WalletResponseMaxInterval = 1 * time.Millisecond

	_, _, err := clientHandler.RequestShareAccountForDApp(testDAppData)
	assert.Equal(t, ErrWalletResponseTimeout, err)
	WalletResponseMaxInterval = backupWalletResponseMaxInterval
}

func TestRequestRejectedWhileWaiting(t *testing.T) {
	clientHandler := NewClientSideHandler()

	clientHandler.setRequestRunning()

	_, _, err := clientHandler.RequestShareAccountForDApp(testDAppData)
	assert.Equal(t, ErrAnotherConnectorOperationIsAwaitingFor, err)
}
