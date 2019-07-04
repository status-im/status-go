package keychain

// Keychain interface that must be implemented by ios/android backend.
type Keychain interface {
	SecurityLevel() int
	CreateKey(auth string) error
	DeleteKey(auth string) error
	Encrypt(auth string, dec []byte) (enc []byte, err error)
	Decrypt(auth string, enc []byte) (dec []byte, err error)
}
