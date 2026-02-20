package commands

import "context"

//go:generate go tool mockgen -source=interfaces.go -destination=mock/interfaces.go -package=mock_commands

// WCSessionDisconnector handles disconnecting WalletConnect sessions
type WCSessionDisconnector interface {
	DisconnectSession(ctx context.Context, topic string) error
}
