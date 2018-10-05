package globalplatform

// KeyProvider is a struct that contains encoding and MAC keys used to communicate with smartcards.
type KeyProvider struct {
	enc []byte
	mac []byte
}

// Enc returns the enc key data.
func (k *KeyProvider) Enc() []byte {
	return k.enc
}

// Mac returns the MAC key data.
func (k *KeyProvider) Mac() []byte {
	return k.mac
}

// NewKeyProvider returns a new KeyProvider with the specified ENC and MAC keys.
func NewKeyProvider(enc, mac []byte) *KeyProvider {
	return &KeyProvider{
		enc: enc,
		mac: mac,
	}
}
