package protocol

import (
	"errors"
	"fmt"

	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"

	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/protobuf"
)

const encryptedPayloadKeyLength = 16
const nonceLength = 12

var ErrInvalidPushNotificationRegisterVersion = errors.New("invalid version")
var ErrEmptyPushNotificationRegisterPayload = errors.New("empty payload")
var ErrEmptyPushNotificationRegisterPublicKey = errors.New("no public key")

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

func (p *Server) ValidateRegistration(previousPreferences *protobuf.PushNotificationPreferences, publicKey *ecdsa.PublicKey, payload []byte) error {
	if payload == nil {
		return ErrEmptyPushNotificationRegisterPayload
	}

	if publicKey == nil {
		return ErrEmptyPushNotificationRegisterPublicKey
	}

	sharedKey, err := ecies.ImportECDSA(p.config.Identity).GenerateShared(
		ecies.ImportECDSAPublic(publicKey),
		encryptedPayloadKeyLength,
		encryptedPayloadKeyLength,
	)
	if err != nil {
		return err
	}

	decryptedPayload, err := decrypt(payload, sharedKey)
	if err != nil {
		return err
	}

	fmt.Println(decryptedPayload)

	/*if newRegistration.Version < 1 {
		return ErrInvalidPushNotificationRegisterVersion
	}*/
	return nil
}

func decrypt(cyphertext []byte, key []byte) ([]byte, error) {
	if len(cyphertext) < nonceLength {
		return nil, errors.New("invalid cyphertext length")
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
	return gcm.Open(nil, nonce, cyphertext, nil)
}
