package requests

import "errors"

type StoreWalletConnectSession struct {
	PeerId    		string         `json:"peerId"`
	ConnectorInfo 	string         `json:"connectorInfo"`
}

var ErrEmptyPeerId = errors.New("store-wallet-connect-session: empty peerId")
var ErrEmptyConnectorInfo = errors.New("store-wallet-connect-session: empty connectorInfo")

func (a *StoreWalletConnectSession) Validate() error {
	if len(a.PeerId) == 0 {
		return ErrEmptyPeerId
	}

	if len(a.ConnectorInfo) == 0 {
		return ErrEmptyConnectorInfo
	}

	return nil
}
