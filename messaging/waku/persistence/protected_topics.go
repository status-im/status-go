package persistence

import (
	"crypto/ecdsa"
)

type ProtectedTopics interface {
	Insert(pubsubTopic string, privKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) error
	Delete(pubsubTopic string) error
	FetchPrivateKey(topic string) (*ecdsa.PrivateKey, error)
	ProtectedTopics() ([]ProtectedTopic, error)
}

type ProtectedTopic struct {
	PubKey *ecdsa.PublicKey
	Topic  string
}
