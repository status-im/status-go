package protocol

import (
	"context"
)

func (m *Messenger) addWalletConnectSession(peerID string, connectorInfo string, sessionInfo string) (*MessengerResponse, error) {
	err := m.persistence.InsertWalletConnectSession(peerID, connectorInfo, sessionInfo)
	if err != nil {
		return nil, err
	}
	return nil, err
}

func (m *Messenger) getWalletConnectSession() (Session, error) {

	response, err := m.persistence.GetWalletConnectSession()
	if err != nil {
		return response, err
	}
	return response, err
}

func (m *Messenger) destroyWalletConnectSession(peerID string) (Session, error) {

	response, err := m.persistence.DeleteWalletConnectSession(peerID)
	if err != nil {
		return response, err
	}
	return response, err
}

func (m *Messenger) AddWalletConnectSession(ctx context.Context, PeerID string, ConnectorInfo string, SessionInfo string) (*MessengerResponse, error) {
	return m.addWalletConnectSession(PeerID, ConnectorInfo, SessionInfo)
}

func (m *Messenger) GetWalletConnectSession(ctx context.Context) (Session, error) {

	return m.getWalletConnectSession()
}

func (m *Messenger) DestroyWalletConnectSession(ctx context.Context, PeerID string) (Session, error) {

	return m.destroyWalletConnectSession(PeerID)
}
