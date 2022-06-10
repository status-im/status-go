package server

import (
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/status-im/status-go/protocol/common"
)

type Payload struct {
	plain     []byte
	encrypted []byte
}

type PayloadManager struct {
	aesKey   []byte
	toSend   *Payload
	received *Payload
}

func NewPayloadManager(pk *ecdsa.PrivateKey) (*PayloadManager, error) {
	ek, err := makeEncryptionKey(pk)
	if err != nil {
		return nil, err
	}

	return &PayloadManager{ek, new(Payload), new(Payload)}, nil
}

func (pm *PayloadManager) Mount(data []byte) error {
	ep, err := common.Encrypt(data, pm.aesKey, rand.Reader)
	if err != nil {
		return err
	}

	pm.toSend.plain = data
	pm.toSend.encrypted = ep
	return nil
}

func (pm *PayloadManager) Receive(data []byte) error {
	pd, err := common.Decrypt(data, pm.aesKey)
	if err != nil {
		return err
	}

	pm.received.encrypted = data
	pm.received.plain = pd
	return nil
}

func (pm *PayloadManager) ToSend() []byte {
	return pm.toSend.encrypted
}

func (pm *PayloadManager) Received() []byte {
	return pm.received.plain
}

func (pm *PayloadManager) ResetPayload() {
	pm.toSend = new(Payload)
	pm.received = new(Payload)
}
