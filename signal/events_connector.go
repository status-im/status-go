package signal

import (
	"github.com/status-im/status-go/eth-node/types"
)

const (
	EventConnectorSendRequestAccounts   = "connector.sendRequestAccounts"
	EventConnectorSendTransaction       = "connector.sendTransaction"
	EventConnectorSign                  = "connector.sign"
	EventConnectorDAppPermissionGranted = "connector.dAppPermissionGranted"
	EventConnectorDAppPermissionRevoked = "connector.dAppPermissionRevoked"
	EventConnectorDAppChainIdSwitched   = "connector.dAppChainIdSwitched"
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

type ConnectorSendDappPermissionGrantedSignal struct {
	ConnectorDApp
	Chains        []uint64      `json:"chains"`
	SharedAccount types.Address `json:"sharedAccount"`
}

type ConnectorSignSignal struct {
	ConnectorDApp
	RequestID string `json:"requestId"`
	Challenge string `json:"challenge"`
	Address   string `json:"address"`
	Method    string `json:"method"`
}

type ConnectorDAppChainIdSwitchedSignal struct {
	URL     string `json:"url"`
	ChainId string `json:"chainId"`
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

func SendConnectorSign(dApp ConnectorDApp, requestID, challenge, address string, method string) {
	send(EventConnectorSign, ConnectorSignSignal{
		ConnectorDApp: dApp,
		RequestID:     requestID,
		Challenge:     challenge,
		Address:       address,
		Method:        method,
	})
}

func SendConnectorDAppPermissionGranted(dApp ConnectorDApp, account types.Address, chains []uint64) {
	send(EventConnectorDAppPermissionGranted, ConnectorSendDappPermissionGrantedSignal{
		ConnectorDApp: dApp,
		Chains:        chains,
		SharedAccount: account,
	})
}

func SendConnectorDAppPermissionRevoked(dApp ConnectorDApp) {
	send(EventConnectorDAppPermissionRevoked, dApp)
}

func SendConnectorDAppChainIdSwitched(payload ConnectorDAppChainIdSwitchedSignal) {
	send(EventConnectorDAppChainIdSwitched, payload)
}
