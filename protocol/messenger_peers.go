package protocol

func (m *Messenger) AddStorePeer(address string) (string, error) {
	return m.transport.AddStorePeer(address)
}

func (m *Messenger) AddRelayPeer(address string) (string, error) {
	return m.transport.AddStorePeer(address)
}

func (m *Messenger) DialPeer(address string) error {
	return m.transport.DialPeer(address)
}

func (m *Messenger) DialPeerByID(peerID string) error {
	return m.transport.DialPeerByID(peerID)
}

func (m *Messenger) DropPeer(peerID string) error {
	return m.transport.DropPeer(peerID)
}

func (m *Messenger) Peers() map[string][]string {
	return m.transport.Peers()
}
