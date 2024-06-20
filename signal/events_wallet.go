package signal

type SignalType string

const (
	Wallet           = SignalType("wallet")
	SignTransactions = SignalType("wallet.sign.transactions")
)

// SendWalletEvent sends event from services/wallet/events.
func SendWalletEvent(signalType SignalType, event interface{}) {
	send(string(signalType), event)
}
