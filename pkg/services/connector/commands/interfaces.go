package commands

import (
	"context"

	"github.com/status-im/status-go/pkg/services/connector/walletconnect"
)

//go:generate go tool mockgen -source=interfaces.go -destination=mock/interfaces.go -package=mock_commands

// WCClientGetter returns the live WalletConnect client, or nil if not initialized.
type WCClientGetter func() *walletconnect.Client

// WCSessionDisconnector handles disconnecting WalletConnect sessions
type WCSessionDisconnector interface {
	DisconnectSession(ctx context.Context, topic string) error
}
