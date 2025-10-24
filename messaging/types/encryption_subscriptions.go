package types

type EncryptionSubscriptions struct {
	SendContactCode    <-chan struct{}
	NewHashRatchetKeys <-chan []*HashRatchetInfo
	Quit               chan struct{}
}
