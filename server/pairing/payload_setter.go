package pairing

type PayloadSetter interface {
	PayloadLocker
	PayloadResetter
	Encryptor

	// Receive accepts data from an inbound source into the PayloadSetter's state
	Receive(data []byte) error

	// Received returns a decrypted and parsed payload from an inbound source
	Received() []byte
}
