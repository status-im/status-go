package walletconnect

//go:generate go tool mockgen -source=interfaces.go -destination=relay_mock_test.go -package=walletconnect

// Relay is the interface for the WalletConnect relay transport.
// It is implemented by RelayClient and can be substituted with a mock in tests.
type Relay interface {
	Connect() error
	Close() error
	Subscribe(topic string) (string, error)
	Publish(topic, message string, tag int) error
	FetchMessages(topic string) ([]RelayMessage, bool, error)
	SetMessageHandler(handler MessageHandler)
	SetReconnectedHandler(handler ReconnectedHandler)
}
