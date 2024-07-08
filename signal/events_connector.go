package signal

import (
	"encoding/hex"
)

const (
	EventConnectorSendRequestAccounts = "connector.sendRequestAccounts"
	EventConnectorSendTransaction     = "connector.sendTransaction"
)

// ConnectorSendRequestAccounts is triggered when a request for accounts is sent.
type ConnectorSendRequestAccounts struct {
	DAppUrl     string `json:"dAppUrl"`
	DAppName    string `json:"dAppName"`
	DAppIconUrl string `json:"dAppIconUrl"`
}

// ConnectorSendTransactionSignal is triggered when a transaction is requested to be sent.
type ConnectorSendTransactionSignal struct {
	DAppUrl string `json:"dAppUrl"`
	ChainID uint64 `json:"chainID"`
	TxArgs  string `json:"txArgs"`
}

func SendConnectorSendRequestAccounts(dAppUrl string, dAppName string, dAppIconUrl string) {
	send(EventConnectorSendRequestAccounts, ConnectorSendRequestAccounts{
		DAppUrl:     dAppUrl,
		DAppName:    dAppName,
		DAppIconUrl: dAppIconUrl,
	})
}

func SendConnectorSendTransaction(dAppUrl string, chainID uint64, txArgs string) {
	send(EventConnectorSendTransaction, ConnectorSendTransactionSignal{
		DAppUrl: dAppUrl,
		ChainID: chainID,
		TxArgs:  hex.EncodeToString([]byte(txArgs)),
	})
}
