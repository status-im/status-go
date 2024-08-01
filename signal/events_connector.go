package signal

const (
	EventConnectorSendRequestAccounts   = "connector.sendRequestAccounts"
	EventConnectorSendTransaction       = "connector.sendTransaction"
	EventConnectorDAppPermissionGranted = "connector.dAppPermissionGranted"
	EventConnectorDAppPermissionRevoked = "connector.dAppPermissionRevoked"
)

type ConnectorDApp struct {
	URL     string `json:"url"`
	Name    string `json:"name"`
	IconURL string `json:"iconUrl"`
}

// ConnectorSendRequestAccountsSignal is triggered when a request for accounts is sent.
type ConnectorSendRequestAccountsSignal struct {
	ConnectorDApp
	RequestID string `json:"requestId"`
}

// ConnectorSendTransactionSignal is triggered when a transaction is requested to be sent.
type ConnectorSendTransactionSignal struct {
	ConnectorDApp
	RequestID string `json:"requestId"`
	ChainID   uint64 `json:"chainId"`
	TxArgs    string `json:"txArgs"`
}

func SendConnectorSendRequestAccounts(dApp ConnectorDApp, requestID string) {
	send(EventConnectorSendRequestAccounts, ConnectorSendRequestAccountsSignal{
		ConnectorDApp: dApp,
		RequestID:     requestID,
	})
}

func SendConnectorSendTransaction(dApp ConnectorDApp, chainID uint64, txArgs string, requestID string) {
	send(EventConnectorSendTransaction, ConnectorSendTransactionSignal{
		ConnectorDApp: dApp,
		RequestID:     requestID,
		ChainID:       chainID,
		TxArgs:        txArgs,
	})
}

func SendConnectorDAppPermissionGranted(dApp ConnectorDApp) {
	send(EventConnectorDAppPermissionGranted, dApp)
}

func SendConnectorDAppPermissionRevoked(dApp ConnectorDApp) {
	send(EventConnectorDAppPermissionRevoked, dApp)
}
