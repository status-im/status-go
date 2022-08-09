package requests

import "errors"

type StoreWalletConnectSession struct {
	PeerID        string `json:"peerId"`
	ConnectorInfo string `json:"connectorInfo"`
	SessionInfo   string `json:"sessionInfo"`
}

var ErrEmptyPeerID = errors.New("store-wallet-connect-session: empty peerId")
var ErrEmptyConnectorInfo = errors.New("store-wallet-connect-session: empty connectorInfo")
var ErrEmptySessionInfo = errors.New("store-wallet-connect-session: empty sessionInfo")

func (a *StoreWalletConnectSession) Validate() error {
	if len(a.PeerID) == 0 {
		return ErrEmptyPeerID
	}

	if len(a.ConnectorInfo) == 0 {
		return ErrEmptyConnectorInfo
	}

	if len(a.SessionInfo) == 0 {
		return ErrEmptySessionInfo
	}

	return nil
}
