package protocol

import (
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/wallet/walletconnect"
)

type WalletConnectSession struct {
	PeerID   string `json:"peerId"`
	DAppName string `json:"dappName"`
	DAppURL  string `json:"dappURL"`
	Info     string `json:"info"`
}

func (m *Messenger) getWalletConnectSession() ([]WalletConnectSession, error) {
	return m.persistence.GetWalletConnectSession()
}

func (m *Messenger) AddWalletConnectSession(request *requests.AddWalletConnectSession) error {
	if err := request.Validate(); err != nil {
		return err
	}

	session := &WalletConnectSession{
		PeerID:   request.PeerID,
		DAppName: request.DAppName,
		DAppURL:  request.DAppURL,
		Info:     request.Info,
	}

	return m.persistence.InsertWalletConnectSession(session)
}

func (m *Messenger) NewWalletConnectV2SessionCreatedNotification(session walletconnect.Session) error {
	now := m.GetCurrentTimeInMillis()

	notification := &ActivityCenterNotification{
		ID:                         types.FromHex(string(session.Topic) + "_dapp_connected"),
		Type:                       ActivityCenterNotificationTypeDAppConnected,
		DAppURL:                    session.Peer.Metadata.URL,
		DAppName:                   session.Peer.Metadata.Name,
		WalletProviderSessionTopic: string(session.Topic),
		Timestamp:                  now,
		UpdatedAt:                  now,
	}

	if len(session.Peer.Metadata.Icons) > 0 {
		notification.DAppIconURL = session.Peer.Metadata.Icons[0]
	}

	_, err := m.persistence.SaveActivityCenterNotification(notification, true)

	return err
}

func (m *Messenger) GetWalletConnectSession() ([]WalletConnectSession, error) {

	return m.getWalletConnectSession()
}

func (m *Messenger) DestroyWalletConnectSession(peerID string) error {
	return m.persistence.DeleteWalletConnectSession(peerID)
}
