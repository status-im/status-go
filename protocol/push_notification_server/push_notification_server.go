package protocol

import (
	"errors"
	"fmt"
	"io"

	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/protobuf"
)

const encryptedPayloadKeyLength = 16
const nonceLength = 12

var ErrInvalidPushNotificationPreferencesVersion = errors.New("invalid version")
var ErrEmptyPushNotificationPreferencesPayload = errors.New("empty payload")
var ErrEmptyPushNotificationPreferencesPublicKey = errors.New("no public key")
var ErrCouldNotUnmarshalPushNotificationPreferences = errors.New("could not unmarshal preferences")
var ErrInvalidCiphertextLength = errors.New("invalid cyphertext length")

type Config struct {
	// Identity is our identity key
	Identity *ecdsa.PrivateKey
	// GorushUrl is the url for the gorush service
	GorushURL string
}

type Server struct {
	persistence *Persistence
	config      *Config
}

func New(persistence *Persistence) *Server {
	return &Server{persistence: persistence}
}

func (p *Server) generateSharedKey(publicKey *ecdsa.PublicKey) ([]byte, error) {
	return ecies.ImportECDSA(p.config.Identity).GenerateShared(
		ecies.ImportECDSAPublic(publicKey),
		encryptedPayloadKeyLength,
		encryptedPayloadKeyLength,
	)
}

func (p *Server) ValidateRegistration(previousPreferences *protobuf.PushNotificationPreferences, publicKey *ecdsa.PublicKey, payload []byte) error {
	if payload == nil {
		return ErrEmptyPushNotificationPreferencesPayload
	}

	if publicKey == nil {
		return ErrEmptyPushNotificationPreferencesPublicKey
	}

	sharedKey, err := p.generateSharedKey(publicKey)
	if err != nil {
		return err
	}

	decryptedPayload, err := decrypt(payload, sharedKey)
	if err != nil {
		return err
	}

	preferences := &protobuf.PushNotificationPreferences{}

	if err := proto.Unmarshal(decryptedPayload, preferences); err != nil {
		return ErrCouldNotUnmarshalPushNotificationPreferences
	}

	if preferences.Version < 1 {
		return ErrInvalidPushNotificationPreferencesVersion
	}
	fmt.Println(decryptedPayload)

	/*if newRegistration.Version < 1 {
		return ErrInvalidPushNotificationPreferencesVersion
	}*/
	return nil
}

func decrypt(cyphertext []byte, key []byte) ([]byte, error) {
	if len(cyphertext) < nonceLength {
		return nil, ErrInvalidCiphertextLength
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := cyphertext[:nonceLength]
	return gcm.Open(nil, nonce, cyphertext[nonceLength:], nil)
}

func encrypt(plaintext []byte, key []byte, reader io.Reader) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
