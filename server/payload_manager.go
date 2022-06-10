package server

type PayloadManager struct {
	toSend   []byte
	received []byte
}

func (pm *PayloadManager) Mount(data []byte) {
	pm.toSend = data
}

func (pm *PayloadManager) Receive(data []byte) {
	pm.received = data
}

func (pm *PayloadManager) ToSend() []byte {
	return pm.toSend
}

func (pm *PayloadManager) Received() []byte {
	return pm.received
}
