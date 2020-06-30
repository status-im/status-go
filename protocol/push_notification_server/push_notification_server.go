package protocol

import (
	"crypto/ecdsa"
	"errors"
	"github.com/status-im/status-go/protocol/protobuf"
)

var ErrEmptyPushNotificationRegisterMessage = errors.New("empty PushNotificationRegisterMessage")

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

func (p *Server) ValidateRegistration(previousRegistration *protobuf.PushNotificationRegister, newRegistration *protobuf.PushNotificationRegister) error {
	if newRegistration == nil {
		return ErrEmptyPushNotificationRegisterMessage
	}
	return nil
}
