package pairing

import (
	"crypto/rand"
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

// TODO resolve the many cases of other structs simply wrapping their encryptor rather than embedding the functionality

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

// decryptPlain decrypts any given plain text using the internal AES key and returns the encrypted value
// This function is different to Decrypt as the internal EncryptionPayload.plain value is not set
func (pem *PayloadEncryptor) decryptPlain(plaintext []byte) ([]byte, error) {
	return common.Decrypt(plaintext, pem.aesKey)
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
