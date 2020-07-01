package protocol

import (
	"errors"
	"fmt"
	"io"

	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/protobuf"
)

const encryptedPayloadKeyLength = 16
const nonceLength = 12

var ErrInvalidPushNotificationOptionsVersion = errors.New("invalid version")
var ErrEmptyPushNotificationOptionsPayload = errors.New("empty payload")
var ErrMalformedPushNotificationOptionsInstallationID = errors.New("invalid installationID")
var ErrEmptyPushNotificationOptionsPublicKey = errors.New("no public key")
var ErrCouldNotUnmarshalPushNotificationOptions = errors.New("could not unmarshal preferences")
var ErrInvalidCiphertextLength = errors.New("invalid cyphertext length")
var ErrMalformedPushNotificationOptionsAccessToken = errors.New("invalid access token")
var ErrMalformedPushNotificationOptionsDeviceToken = errors.New("invalid device token")

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

func (p *Server) validateUUID(u string) error {
	if len(u) == 0 {
		return errors.New("empty uuid")
	}
	_, err := uuid.Parse(u)
	return err
}

func (p *Server) ValidateRegistration(previousPreferences *protobuf.PushNotificationOptions, publicKey *ecdsa.PublicKey, payload []byte) error {
	if payload == nil {
		return ErrEmptyPushNotificationOptionsPayload
	}

	if publicKey == nil {
		return ErrEmptyPushNotificationOptionsPublicKey
	}

	sharedKey, err := p.generateSharedKey(publicKey)
	if err != nil {
		return err
	}

	decryptedPayload, err := decrypt(payload, sharedKey)
	if err != nil {
		return err
	}

	preferences := &protobuf.PushNotificationOptions{}

	if err := proto.Unmarshal(decryptedPayload, preferences); err != nil {
		return ErrCouldNotUnmarshalPushNotificationOptions
	}

	if preferences.Version < 1 {
		return ErrInvalidPushNotificationOptionsVersion
	}

	if previousPreferences != nil && preferences.Version <= previousPreferences.Version {
		return ErrInvalidPushNotificationOptionsVersion
	}

	if err := p.validateUUID(preferences.InstallationId); err != nil {
		return ErrMalformedPushNotificationOptionsInstallationID
	}

	// Unregistering message
	if preferences.Unregister {
		return nil
	}

	if err := p.validateUUID(preferences.AccessToken); err != nil {
		return ErrMalformedPushNotificationOptionsAccessToken
	}

	if len(preferences.Token) == 0 {
		return ErrMalformedPushNotificationOptionsDeviceToken
	}
	fmt.Println(decryptedPayload)

	/*if newRegistration.Version < 1 {
		return ErrInvalidPushNotificationOptionsVersion
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
