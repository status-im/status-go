package pairing

type ForSenders interface {
	_forSenders()
}

type ForReceivers interface {
	_forReceivers()
}

type ForServers interface {
	_forServers()
}

type ForClients interface {
	_forClients()
}

func (sc *SenderConfig) _forSenders()     {}
func (rc *ReceiverConfig) _forReceivers() {}
func (sc *ServerConfig) _forServers()     {}
func (sc *ClientConfig) _forClients()     {}

type PayloadManagerer interface {
	// Mount Loads the payload into the PayloadManager's state
	Mount() error

	// Receive stores data from an inbound source into the PayloadManager's state
	Receive(data []byte) error
}

type PayloadRepository interface {
	PayloadLoader
	StoreToSource() error
}

type PayloadLocker interface {
	// LockPayload prevents future excess to outbound safe and received data
	LockPayload()
}

type PayloadResetter interface {
	// ResetPayload resets all payloads the PayloadManager has in its state
	ResetPayload()
}

type Encryptor interface {
	// EncryptPlain encrypts the given plaintext using internal key(s)
	EncryptPlain(plaintext []byte) ([]byte, error)
}
