package requests

import "errors"

type StoreWalletConnectSession struct {
	PeerID        string `json:"peerId"`
	ConnectorInfo string `json:"connectorInfo"`
}

var ErrEmptyPeerID = errors.New("store-wallet-connect-session: empty peerId")
var ErrEmptyConnectorInfo = errors.New("store-wallet-connect-session: empty connectorInfo")

func (a *StoreWalletConnectSession) Validate() error {
	if len(a.PeerID) == 0 {
		return ErrEmptyPeerID
	}

	if len(a.ConnectorInfo) == 0 {
		return ErrEmptyConnectorInfo
	}

	return nil
}
