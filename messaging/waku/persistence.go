package wakuv2

import "crypto/ecdsa"

type ProtectedTopicsPersistence interface {
	Insert(pubsubTopic string, privKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) error
	Delete(pubsubTopic string) error
	FetchPrivateKey(topic string) (*ecdsa.PrivateKey, error)
	All() ([]ProtectedTopic, error)
}

type ProtectedTopic struct {
	PubKey *ecdsa.PublicKey
	Topic  string
}
