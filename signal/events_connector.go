package signal

import (
	"encoding/hex"
)

const (
	EventConnectorSendTransaction = "connector.sendTransaction"
)

// ConnectorSendTransactionSignal is triggered when a transaction is requested to be sent.
type ConnectorSendTransactionSignal struct {
	DAppUrl string `json:"dAppUrl"`
	ChainID uint64 `json:"chainID"`
	TxArgs  string `json:"txArgs"`
}

func SendConnectorSendTransaction(dAppUrl string, chainID uint64, txArgs string) {
	send(EventConnectorSendTransaction, ConnectorSendTransactionSignal{
		DAppUrl: dAppUrl,
		ChainID: chainID,
		TxArgs:  hex.EncodeToString([]byte(txArgs)),
	})
}
