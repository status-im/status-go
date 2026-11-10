package signal

type SignalType string

const (
	Wallet                           = SignalType("wallet")
	SignTransactions                 = SignalType("wallet.sign.transactions")
	RouterSendingTransactionsStarted = SignalType("wallet.router.sending-transactions-started")
	SignRouterTransactions           = SignalType("wallet.router.sign-transactions")
	RouterTransactionsSent           = SignalType("wallet.router.transactions-sent")
	TransactionStatusChanged         = SignalType("wallet.transaction.status-changed")
	SuggestedRoutes                  = SignalType("wallet.suggested.routes")
	TokenListsUpdated                = SignalType("wallet.token-lists.updated")
)

// SendWalletEvent sends event from services/wallet/events.
func SendWalletEvent(signalType SignalType, event interface{}) {
	CurrentSession().SendWalletEvent(signalType, event)
}

// SendWalletEvent sends in the session s.
func (s Session) SendWalletEvent(signalType SignalType, event interface{}) {
	s.send(string(signalType), event)
}
