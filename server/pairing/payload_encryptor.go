package pairing

import (
	"crypto/rand"
	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/common"
)

// EncryptionPayload represents the plain text and encrypted text of payload data
type EncryptionPayload struct {
	plain     []byte
	encrypted []byte
	locked    bool
}

func (ep *EncryptionPayload) lock() {
	ep.locked = true
}

// PayloadEncryptionManager is responsible for encrypting and decrypting payload data
type PayloadEncryptionManager struct {
	logger   *zap.Logger
	aesKey   []byte
	toSend   *EncryptionPayload
	received *EncryptionPayload
}

func NewPayloadEncryptionManager(aesKey []byte, logger *zap.Logger) (*PayloadEncryptionManager, error) {
	return &PayloadEncryptionManager{logger.Named("PayloadEncryptionManager"), aesKey, new(EncryptionPayload), new(EncryptionPayload)}, nil
}

// EncryptPlain encrypts any given plain text using the internal AES key and returns the encrypted value
// This function is different to Encrypt as the internal EncryptionPayload.encrypted value is not set
func (pem *PayloadEncryptionManager) EncryptPlain(plaintext []byte) ([]byte, error) {
	l := pem.logger.Named("EncryptPlain()")
	l.Debug("fired")

	return common.Encrypt(plaintext, pem.aesKey, rand.Reader)
}

func (pem *PayloadEncryptionManager) Encrypt(data []byte) error {
	l := pem.logger.Named("Encrypt()")
	l.Debug("fired")

	ep, err := common.Encrypt(data, pem.aesKey, rand.Reader)
	if err != nil {
		return err
	}

	pem.toSend.plain = data
	pem.toSend.encrypted = ep

	l.Debug(
		"after common.Encrypt",
		zap.Binary("data", data),
		zap.Binary("pem.aesKey", pem.aesKey),
		zap.Binary("ep", ep),
	)

	return nil
}

func (pem *PayloadEncryptionManager) Decrypt(data []byte) error {
	l := pem.logger.Named("Decrypt()")
	l.Debug("fired")

	pd, err := common.Decrypt(data, pem.aesKey)
	l.Debug(
		"after common.Decrypt(data, pem.aesKey)",
		zap.Binary("data", data),
		zap.Binary("pem.aesKey", pem.aesKey),
		zap.Binary("pd", pd),
		zap.Error(err),
	)
	if err != nil {
		return err
	}

	pem.received.encrypted = data
	pem.received.plain = pd
	return nil
}

func (pem *PayloadEncryptionManager) ToSend() []byte {
	if pem.toSend.locked {
		return nil
	}
	return pem.toSend.encrypted
}

func (pem *PayloadEncryptionManager) Received() []byte {
	if pem.toSend.locked {
		return nil
	}
	return pem.received.plain
}

func (pem *PayloadEncryptionManager) ResetPayload() {
	pem.toSend = new(EncryptionPayload)
	pem.received = new(EncryptionPayload)
}

func (pem *PayloadEncryptionManager) LockPayload() {
	l := pem.logger.Named("LockPayload")
	l.Debug("fired")

	pem.toSend.lock()
	pem.received.lock()
}

// PayloadEncryptor is responsible for encrypting and decrypting payload data
type PayloadEncryptor struct {
	aesKey  []byte
	payload *EncryptionPayload
}

func NewPayloadEncryptor(aesKey []byte) *PayloadEncryptor {
	return &PayloadEncryptor{
		aesKey,
		new(EncryptionPayload),
	}
}

// Renew regenerates the whole PayloadEncryptor and returns the new instance, only the aesKey is preserved
func (pem *PayloadEncryptor) Renew() *PayloadEncryptor {
	return &PayloadEncryptor{
		aesKey:  pem.aesKey,
		payload: new(EncryptionPayload),
	}
}

// encryptPlain encrypts any given plain text using the internal AES key and returns the encrypted value
// This function is different to Encrypt as the internal EncryptionPayload.encrypted value is not set
func (pem *PayloadEncryptor) encryptPlain(plaintext []byte) ([]byte, error) {
	return common.Encrypt(plaintext, pem.aesKey, rand.Reader)
}

func (pem *PayloadEncryptor) encrypt(data []byte) error {
	ep, err := common.Encrypt(data, pem.aesKey, rand.Reader)
	if err != nil {
		return err
	}

	pem.payload.plain = data
	pem.payload.encrypted = ep
	return nil
}

func (pem *PayloadEncryptor) decrypt(data []byte) error {
	pd, err := common.Decrypt(data, pem.aesKey)
	if err != nil {
		return err
	}

	pem.payload.encrypted = data
	pem.payload.plain = pd
	return nil
}

func (pem *PayloadEncryptor) getEncrypted() []byte {
	if pem.payload.locked {
		return nil
	}
	return pem.payload.encrypted
}

func (pem *PayloadEncryptor) getDecrypted() []byte {
	if pem.payload.locked {
		return nil
	}
	return pem.payload.plain
}

func (pem *PayloadEncryptor) resetPayload() {
	pem.payload = new(EncryptionPayload)
}

func (pem *PayloadEncryptor) lockPayload() {
	pem.payload.lock()
}
