package signal

const (
	walletEvent = "wallet"
)

// SendWalletEvent sends event from services/wallet/events.
func SendWalletEvent(event interface{}) {
	send(walletEvent, event)
}
