package protocol

import (
	"context"
	"github.com/status-im/status-go/protocol/requests"
)

func (m *Messenger) addWalletConnectSession(peerId string, connectorInfo string) (*MessengerResponse, error) {
	err := m.persistence.InsertWalletConnectSession(peerId, connectorInfo)
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

func (m *Messenger) AddWalletConnectSession(ctx context.Context, request *requests.StoreWalletConnectSession) (*MessengerResponse, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	return m.addWalletConnectSession(request.PeerId, request.ConnectorInfo)
}

func (m *Messenger) GetWalletConnectSession(ctx context.Context, request *requests.StoreWalletConnectSession) (Session, error) {

	seshObject := Session{
		PeerId:        "",
		ConnectorInfo: "",
	}

	err := request.Validate()
	if err != nil {
		return seshObject, err
	}
	return m.getWalletConnectSession()
}
