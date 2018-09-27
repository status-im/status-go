package globalplatform

type KeyProvider struct {
	enc []byte
	mac []byte
}

func (k *KeyProvider) Enc() []byte {
	return k.enc
}

func (k *KeyProvider) Mac() []byte {
	return k.mac
}

func NewKeyProvider(enc, mac []byte) *KeyProvider {
	return &KeyProvider{
		enc: enc,
		mac: mac,
	}
}
