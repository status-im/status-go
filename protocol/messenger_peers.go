package protocol

func (m *Messenger) AddStorePeer(address string) error {
	return m.transport.AddStorePeer(address)
}

func (m *Messenger) AddRelayPeer(address string) error {
	return m.transport.AddStorePeer(address)
}

func (m *Messenger) DropPeer(peerID string) error {
	return m.transport.DropPeer(peerID)
}
