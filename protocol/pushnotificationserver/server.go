package pushnotificationserver

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/crypto/ecies"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

const encryptedPayloadKeyLength = 16
const defaultGorushURL = "https://gorush.status.im"

type Config struct {
	Enabled bool
	// Identity is our identity key
	Identity *ecdsa.PrivateKey
	// GorushUrl is the url for the gorush service
	GorushURL string

	Logger *zap.Logger
}

type Server struct {
	persistence      Persistence
	config           *Config
	messageProcessor *common.MessageProcessor
}

func New(config *Config, persistence Persistence, messageProcessor *common.MessageProcessor) *Server {
	if len(config.GorushURL) == 0 {
		config.GorushURL = defaultGorushURL

	}
	return &Server{persistence: persistence, config: config, messageProcessor: messageProcessor}
}

func (s *Server) Start() error {
	if s.config.Logger == nil {
		logger, err := zap.NewDevelopment()
		if err != nil {
			return errors.Wrap(err, "failed to create a logger")
		}
		s.config.Logger = logger
	}

	s.config.Logger.Info("starting push notification server")
	if s.config.Identity == nil {
		s.config.Logger.Info("Identity nil")
		// Pull identity from database
		identity, err := s.persistence.GetIdentity()
		if err != nil {
			return err
		}
		if identity == nil {
			identity, err = crypto.GenerateKey()
			if err != nil {
				return err
			}
			if err := s.persistence.SaveIdentity(identity); err != nil {
				return err
			}
		}
		s.config.Identity = identity
	}

	pks, err := s.persistence.GetPushNotificationRegistrationPublicKeys()
	if err != nil {
		return err
	}
	// listen to all topics for users registered
	for _, pk := range pks {
		if err := s.listenToPublicKeyQueryTopic(pk); err != nil {
			return err
		}
	}

	s.config.Logger.Info("started push notification server", zap.String("identity", types.EncodeHex(crypto.FromECDSAPub(&s.config.Identity.PublicKey))))

	return nil
}

// HandlePushNotificationRegistration builds a response for the registration and sends it back to the user
func (s *Server) HandlePushNotificationRegistration(publicKey *ecdsa.PublicKey, payload []byte) error {
	response := s.buildPushNotificationRegistrationResponse(publicKey, payload)
	if response == nil {
		return nil
	}
	encodedMessage, err := proto.Marshal(response)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_REGISTRATION_RESPONSE,
		// we skip encryption as might be sent from an ephemeral key
		SkipEncryption: true,
	}

	_, err = s.messageProcessor.SendPrivate(context.Background(), publicKey, rawMessage)
	return err
}

// HandlePushNotificationQuery builds a response for the query and sends it back to the user
func (s *Server) HandlePushNotificationQuery(publicKey *ecdsa.PublicKey, messageID []byte, query protobuf.PushNotificationQuery) error {
	response := s.buildPushNotificationQueryResponse(&query)
	if response == nil {
		return nil
	}
	response.MessageId = messageID
	encodedMessage, err := proto.Marshal(response)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_QUERY_RESPONSE,
		// we skip encryption as sent from an ephemeral key
		SkipEncryption: true,
	}

	_, err = s.messageProcessor.SendPrivate(context.Background(), publicKey, rawMessage)
	return err
}

// HandlePushNotificationRequest will send a gorush notification and send a response back to the user
func (s *Server) HandlePushNotificationRequest(publicKey *ecdsa.PublicKey,
	messageID []byte,
	request protobuf.PushNotificationRequest) error {
	s.config.Logger.Info("handling pn request", zap.Binary("message-id", messageID))

	// This is at-most-once semantic for now
	exists, err := s.persistence.PushNotificationExists(messageID)
	if err != nil {
		return err
	}

	if exists {
		s.config.Logger.Info("already handled")
		return nil
	}

	response := s.buildPushNotificationRequestResponseAndSendNotification(&request)
	if response == nil {
		return nil
	}
	encodedMessage, err := proto.Marshal(response)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_PUSH_NOTIFICATION_RESPONSE,
		// We skip encryption here as the message has been sent from an ephemeral key
		SkipEncryption: true,
	}

	_, err = s.messageProcessor.SendPrivate(context.Background(), publicKey, rawMessage)
	return err
}

// buildGrantSignatureMaterial builds a grant for a specific server.
// We use 3 components:
// 1) The client public key. Not sure this applies to our signature scheme, but best to be conservative. https://crypto.stackexchange.com/questions/15538/given-a-message-and-signature-find-a-public-key-that-makes-the-signature-valid
// 2) The server public key
// 3) The access token
// By verifying this signature, a client can trust the server was instructed to store this access token.
func (s *Server) buildGrantSignatureMaterial(clientPublicKey *ecdsa.PublicKey, serverPublicKey *ecdsa.PublicKey, accessToken string) []byte {
	var signatureMaterial []byte
	signatureMaterial = append(signatureMaterial, crypto.CompressPubkey(clientPublicKey)...)
	signatureMaterial = append(signatureMaterial, crypto.CompressPubkey(serverPublicKey)...)
	signatureMaterial = append(signatureMaterial, []byte(accessToken)...)
	a := crypto.Keccak256(signatureMaterial)
	return a
}

func (s *Server) verifyGrantSignature(clientPublicKey *ecdsa.PublicKey, accessToken string, grant []byte) error {
	signatureMaterial := s.buildGrantSignatureMaterial(clientPublicKey, &s.config.Identity.PublicKey, accessToken)
	recoveredPublicKey, err := crypto.SigToPub(signatureMaterial, grant)
	if err != nil {
		return err
	}

	if !common.IsPubKeyEqual(recoveredPublicKey, clientPublicKey) {
		return errors.New("pubkey mismatch")
	}
	return nil

}

func (s *Server) generateSharedKey(publicKey *ecdsa.PublicKey) ([]byte, error) {
	return ecies.ImportECDSA(s.config.Identity).GenerateShared(
		ecies.ImportECDSAPublic(publicKey),
		encryptedPayloadKeyLength,
		encryptedPayloadKeyLength,
	)
}

func (s *Server) validateUUID(u string) error {
	if len(u) == 0 {
		return errors.New("empty uuid")
	}
	_, err := uuid.Parse(u)
	return err
}

func (s *Server) decryptRegistration(publicKey *ecdsa.PublicKey, payload []byte) ([]byte, error) {
	sharedKey, err := s.generateSharedKey(publicKey)
	if err != nil {
		return nil, err
	}

	return common.Decrypt(payload, sharedKey)
}

// validateRegistration validates a new message against the last one received for a given installationID and and public key
// and return the decrypted message
func (s *Server) validateRegistration(publicKey *ecdsa.PublicKey, payload []byte) (*protobuf.PushNotificationRegistration, error) {
	if payload == nil {
		return nil, ErrEmptyPushNotificationRegistrationPayload
	}

	if publicKey == nil {
		return nil, ErrEmptyPushNotificationRegistrationPublicKey
	}

	decryptedPayload, err := s.decryptRegistration(publicKey, payload)
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

	if err := s.validateUUID(registration.InstallationId); err != nil {
		return nil, ErrMalformedPushNotificationRegistrationInstallationID
	}

	previousVersion, err := s.persistence.GetPushNotificationRegistrationVersion(common.HashPublicKey(publicKey), registration.InstallationId)
	if err != nil {
		return nil, err
	}

	if registration.Version <= previousVersion {
		return nil, ErrInvalidPushNotificationRegistrationVersion
	}

	// unregistering message
	if registration.Unregister {
		return registration, nil
	}

	if err := s.validateUUID(registration.AccessToken); err != nil {
		return nil, ErrMalformedPushNotificationRegistrationAccessToken
	}

	if len(registration.Grant) == 0 {
		return nil, ErrMalformedPushNotificationRegistrationGrant
	}

	if err := s.verifyGrantSignature(publicKey, registration.AccessToken, registration.Grant); err != nil {

		s.config.Logger.Error("failed to verify grant", zap.Error(err))
		return nil, ErrMalformedPushNotificationRegistrationGrant
	}

	if len(registration.DeviceToken) == 0 {
		return nil, ErrMalformedPushNotificationRegistrationDeviceToken
	}

	if registration.TokenType == protobuf.PushNotificationRegistration_UNKNOWN_TOKEN_TYPE {
		return nil, ErrUnknownPushNotificationRegistrationTokenType
	}

	return registration, nil
}

// buildPushNotificationQueryResponse check if we have the client information and send them back
func (s *Server) buildPushNotificationQueryResponse(query *protobuf.PushNotificationQuery) *protobuf.PushNotificationQueryResponse {

	s.config.Logger.Info("handling push notification query")
	response := &protobuf.PushNotificationQueryResponse{}
	if query == nil || len(query.PublicKeys) == 0 {
		return response
	}

	registrations, err := s.persistence.GetPushNotificationRegistrationByPublicKeys(query.PublicKeys)
	if err != nil {
		s.config.Logger.Error("failed to retrieve registration", zap.Error(err))
		return response
	}

	for _, idAndResponse := range registrations {

		registration := idAndResponse.Registration

		info := &protobuf.PushNotificationQueryInfo{
			PublicKey:      idAndResponse.ID,
			Grant:          registration.Grant,
			Version:        registration.Version,
			InstallationId: registration.InstallationId,
		}

		// if instructed to only allow from contacts, send back a list
		if registration.AllowFromContactsOnly {
			info.AllowedKeyList = registration.AllowedKeyList
		} else {
			info.AccessToken = registration.AccessToken
		}
		response.Info = append(response.Info, info)
	}

	response.Success = true
	return response
}

func (s *Server) blockedChatID(blockedChatIDs [][]byte, chatID []byte) bool {
	for _, blockedChatID := range blockedChatIDs {
		if bytes.Equal(blockedChatID, chatID) {
			return true
		}
	}
	return false
}

// buildPushNotificationRequestResponseAndSendNotification will build a response
// and fire-and-forget send a query to the gorush instance
func (s *Server) buildPushNotificationRequestResponseAndSendNotification(request *protobuf.PushNotificationRequest) *protobuf.PushNotificationResponse {
	response := &protobuf.PushNotificationResponse{}
	// We don't even send a response in this case
	if request == nil || len(request.MessageId) == 0 {
		s.config.Logger.Warn("empty message id")
		return nil
	}

	response.MessageId = request.MessageId

	// collect successful requests & registrations
	var requestAndRegistrations []*RequestAndRegistration

	for _, pn := range request.Requests {
		registration, err := s.persistence.GetPushNotificationRegistrationByPublicKeyAndInstallationID(pn.PublicKey, pn.InstallationId)
		report := &protobuf.PushNotificationReport{
			PublicKey:      pn.PublicKey,
			InstallationId: pn.InstallationId,
		}

		if pn.Type == protobuf.PushNotification_UNKNOWN_PUSH_NOTIFICATION_TYPE {
			s.config.Logger.Warn("unhandled type")
			continue
		}

		if err != nil {
			s.config.Logger.Error("failed to retrieve registration", zap.Error(err))
			report.Error = protobuf.PushNotificationReport_UNKNOWN_ERROR_TYPE
		} else if registration == nil {
			s.config.Logger.Warn("empty registration")
			report.Error = protobuf.PushNotificationReport_NOT_REGISTERED
		} else if registration.AccessToken != pn.AccessToken {
			report.Error = protobuf.PushNotificationReport_WRONG_TOKEN
		} else if s.blockedChatID(registration.BlockedChatList, pn.ChatId) {
			// We report as successful but don't send the notification
			report.Success = true
		} else {
			// For now we just assume that the notification will be successful
			requestAndRegistrations = append(requestAndRegistrations, &RequestAndRegistration{
				Request:      pn,
				Registration: registration,
			})
			report.Success = true
		}

		response.Reports = append(response.Reports, report)
	}

	s.config.Logger.Info("built pn request")
	if len(requestAndRegistrations) == 0 {
		s.config.Logger.Warn("no request and registration")
		return response
	}

	// This can be done asynchronously
	goRushRequest := PushNotificationRegistrationToGoRushRequest(requestAndRegistrations)
	err := sendGoRushNotification(goRushRequest, s.config.GorushURL, s.config.Logger)
	if err != nil {
		s.config.Logger.Error("failed to send go rush notification", zap.Error(err))
		// TODO: handle this error?
		// GoRush will not let us know that the sending of the push notification has failed,
		// so this likely mean that the actual HTTP request has failed, or there was some unexpected error
	}

	return response
}

// listenToPublicKeyQueryTopic listen to a topic derived from the hashed public key
func (s *Server) listenToPublicKeyQueryTopic(hashedPublicKey []byte) error {
	if s.messageProcessor == nil {
		return nil
	}
	encodedPublicKey := hex.EncodeToString(hashedPublicKey)
	return s.messageProcessor.JoinPublic(encodedPublicKey)
}

// buildPushNotificationRegistrationResponse will check the registration is valid, save it, and listen to the topic for the queries
func (s *Server) buildPushNotificationRegistrationResponse(publicKey *ecdsa.PublicKey, payload []byte) *protobuf.PushNotificationRegistrationResponse {

	s.config.Logger.Info("handling push notification registration")
	response := &protobuf.PushNotificationRegistrationResponse{
		RequestId: common.Shake256(payload),
	}

	registration, err := s.validateRegistration(publicKey, payload)

	if err != nil {
		if err == ErrInvalidPushNotificationRegistrationVersion {
			response.Error = protobuf.PushNotificationRegistrationResponse_VERSION_MISMATCH
		} else {
			response.Error = protobuf.PushNotificationRegistrationResponse_MALFORMED_MESSAGE
		}
		s.config.Logger.Warn("registration did not validate", zap.Error(err))
		return response
	}

	if registration.Unregister {
		s.config.Logger.Info("unregistering client")
		// We save an empty registration, only keeping version and installation-id
		if err := s.persistence.UnregisterPushNotificationRegistration(common.HashPublicKey(publicKey), registration.InstallationId, registration.Version); err != nil {
			response.Error = protobuf.PushNotificationRegistrationResponse_INTERNAL_ERROR
			s.config.Logger.Error("failed to unregister ", zap.Error(err))
			return response
		}

	} else if err := s.persistence.SavePushNotificationRegistration(common.HashPublicKey(publicKey), registration); err != nil {
		response.Error = protobuf.PushNotificationRegistrationResponse_INTERNAL_ERROR
		s.config.Logger.Error("failed to save registration", zap.Error(err))
		return response
	}

	if err := s.listenToPublicKeyQueryTopic(common.HashPublicKey(publicKey)); err != nil {
		response.Error = protobuf.PushNotificationRegistrationResponse_INTERNAL_ERROR
		s.config.Logger.Error("failed to listen to topic", zap.Error(err))
		return response

	}
	response.Success = true

	s.config.Logger.Info("handled push notification registration successfully")

	return response
}
