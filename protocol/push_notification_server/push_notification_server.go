package push_notification_server

import (
	"errors"
	"io"

	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"golang.org/x/crypto/sha3"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/protocol/protobuf"
)

const encryptedPayloadKeyLength = 16
const nonceLength = 12

type Config struct {
	// Identity is our identity key
	Identity *ecdsa.PrivateKey
	// GorushUrl is the url for the gorush service
	GorushURL string
}

type Server struct {
	persistence Persistence
	config      *Config
}

func New(config *Config, persistence Persistence) *Server {
	return &Server{persistence: persistence, config: config}
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

func (p *Server) decryptRegistration(publicKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {
	sharedKey, err := p.generateSharedKey(publicKey)
	if err != nil {
		return nil, err
	}

	return decrypt(payload, sharedKey)
}

// ValidateRegistration validates a new message against the last one received for a given installationID and and public key
// and return the decrypted message
func (p *Server) ValidateRegistration(publicKey *ecdsa.PublicKey, payload []byte) (*protobuf.PushNotificationRegistration, error) {
	if payload == nil {
		return nil, ErrEmptyPushNotificationRegistrationPayload
	}

	if publicKey == nil {
		return nil, ErrEmptyPushNotificationRegistrationPublicKey
	}

	decryptedPayload, err := p.decryptRegistration(publicKey, payload)
	if err != nil {
		return nil, err
	}

	registration := &protobuf.PushNotificationRegistration{}

	if err := proto.Unmarshal(decryptedPayload, registration); err != nil {
		return nil, ErrCouldNotUnmarshalPushNotificationRegistration
	}

	if registration.Version < 1 {
		return nil, ErrInvalidPushNotificationRegistrationVersion
	}

	if err := p.validateUUID(registration.InstallationId); err != nil {
		return nil, ErrMalformedPushNotificationRegistrationInstallationID
	}

	previousRegistration, err := p.persistence.GetPushNotificationRegistrationByPublicKeyAndInstallationID(publicKey, registration.InstallationId)
	if err != nil {
		return nil, err
	}

	if previousRegistration != nil && registration.Version <= previousRegistration.Version {
		return nil, ErrInvalidPushNotificationRegistrationVersion
	}

	// Unregistering message
	if registration.Unregister {
		return registration, nil
	}

	if err := p.validateUUID(registration.AccessToken); err != nil {
		return nil, ErrMalformedPushNotificationRegistrationAccessToken
	}

	if len(registration.Token) == 0 {
		return nil, ErrMalformedPushNotificationRegistrationDeviceToken
	}

	if registration.TokenType == protobuf.PushNotificationRegistration_UNKNOWN_TOKEN_TYPE {
		return nil, ErrUnknownPushNotificationRegistrationTokenType
	}

	return registration, nil
}

func (p *Server) HandlePushNotificationQuery(query *protobuf.PushNotificationQuery) *protobuf.PushNotificationQueryResponse {
	response := &protobuf.PushNotificationQueryResponse{}
	if query == nil || len(query.PublicKeys) == 0 {
		return response
	}

	registrations, err := p.persistence.GetPushNotificationRegistrationByPublicKeys(query.PublicKeys)
	if err != nil {
		// TODO: log errors
		return response
	}

	for _, idAndResponse := range registrations {

		registration := idAndResponse.Registration
		info := &protobuf.PushNotificationQueryInfo{
			PublicKey:      idAndResponse.ID,
			InstallationId: registration.InstallationId,
		}

		if len(registration.AllowedUserList) > 0 {
			info.AllowedUserList = registration.AllowedUserList
		} else {
			info.AccessToken = registration.AccessToken
		}
		response.Info = append(response.Info, info)
	}

	response.Success = true
	return response
}

func (p *Server) HandlePushNotificationRegistration(publicKey *ecdsa.PublicKey, payload []byte) *protobuf.PushNotificationRegistrationResponse {
	response := &protobuf.PushNotificationRegistrationResponse{
		RequestId: shake256(payload),
	}

	registration, err := p.ValidateRegistration(publicKey, payload)
	if registration != nil {
	}

	if err != nil {
		if err == ErrInvalidPushNotificationRegistrationVersion {
			response.Error = protobuf.PushNotificationRegistrationResponse_VERSION_MISMATCH
		} else {
			response.Error = protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE
		}
		return response
	}

	if registration.Unregister {
		// We save an empty registration, only keeping version and installation-id
		emptyRegistration := &protobuf.PushNotificationRegistration{
			Version:        registration.Version,
			InstallationId: registration.InstallationId,
		}
		if err := p.persistence.SavePushNotificationRegistration(publicKey, emptyRegistration); err != nil {
			response.Error = protobuf.PushNotificationRegistrationResponse_INTERNAL_ERROR
			return response
		}

	} else if err := p.persistence.SavePushNotificationRegistration(publicKey, registration); err != nil {
		response.Error = protobuf.PushNotificationRegistrationResponse_INTERNAL_ERROR
		return response
	}

	response.Success = true

	return response
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

func shake256(buf []byte) []byte {
	h := make([]byte, 64)
	sha3.ShakeSum256(h, buf)
	return h
}
