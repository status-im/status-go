package types

type EncryptionSubscriptions struct {
	SharedSecrets      []*SharedSecret
	SendContactCode    <-chan struct{}
	NewHashRatchetKeys <-chan []*HashRatchetInfo
	Quit               chan struct{}
}
