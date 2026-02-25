package signal

import (
	"github.com/status-im/status-go/internal/crypto/types"
)

const (
	EventConnectorSendRequestAccounts   = "connector.sendRequestAccounts"
	EventConnectorSendTransaction       = "connector.sendTransaction"
	EventConnectorSign                  = "connector.sign"
	EventConnectorDAppPermissionGranted = "connector.dAppPermissionGranted"
	EventConnectorDAppPermissionRevoked = "connector.dAppPermissionRevoked"
	EventConnectorDAppChainIdSwitched   = "connector.dAppChainIdSwitched"
	EventConnectorAccountChanged        = "connector.dAppAccountChanged"

	// WalletConnect (via connector)
	EventWCSessionProposal = "connector.wcSessionProposal"
	EventWCSessionRequest  = "connector.wcSessionRequest"
	EventWCSessionDelete   = "connector.wcSessionDelete"
)

type WCSessionProposalSignal struct {
	RequestID string `json:"requestId"`
	URI       string `json:"uri"`
	Proposal  string `json:"proposal"` // JSON-encoded session proposal
}

func SendWCSessionProposal(requestID, uri, proposalJSON string) {
	send(EventWCSessionProposal, WCSessionProposalSignal{
		RequestID: requestID,
		URI:       uri,
		Proposal:  proposalJSON,
	})
}

type WCSessionRequestSignal struct {
	Topic       string `json:"topic"`
	RequestID   int64  `json:"requestId"`
	RequestJSON string `json:"requestJson"`
}

func SendWCSessionRequest(topic string, requestID int64, requestJSON string) {
	send(EventWCSessionRequest, WCSessionRequestSignal{
		Topic:       topic,
		RequestID:   requestID,
		RequestJSON: requestJSON,
	})
}

type WCSessionDeleteSignal struct {
	Topic   string `json:"topic"`
	DAppURL string `json:"dappUrl"`
}

func SendWCSessionDelete(topic, dappURL string) {
	send(EventWCSessionDelete, WCSessionDeleteSignal{
		Topic:   topic,
		DAppURL: dappURL,
	})
}

type ConnectorDApp struct {
	URL      string `json:"url"`
	Name     string `json:"name"`
	IconURL  string `json:"iconUrl"`
	ClientID string `json:"clientId"`
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
	URL      string `json:"url"`
	ChainId  string `json:"chainId"`
	ClientID string `json:"clientId"`
}

type ConnectorAccountChangedSignal struct {
	URL           string        `json:"url"`
	ClientID      string        `json:"clientId"`
	SharedAccount types.Address `json:"sharedAccount"`
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

func SendConnectorAccountChanged(url string, clientID string, account types.Address) {
	send(EventConnectorAccountChanged, ConnectorAccountChangedSignal{
		URL:           url,
		ClientID:      clientID,
		SharedAccount: account,
	})
}
