package protocol

func (m *Messenger) ImageServerURL() string {
	return m.httpServer.MakeImageServerURL()
}
