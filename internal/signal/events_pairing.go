package signal

const (
	localPairingEvent = "localPairing"
)

// SendLocalPairingEvent sends event from services/pairing/events.
func SendLocalPairingEvent(event interface{}) {
	CurrentSession().SendLocalPairingEvent(event)
}

// SendLocalPairingEvent sends in the session s.
func (s Session) SendLocalPairingEvent(event interface{}) {
	s.send(localPairingEvent, event)
}
