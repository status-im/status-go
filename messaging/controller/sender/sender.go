package sender

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	ethtypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/messaging/common"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/pkg/pubsub"
)

type Sender struct {
	identity *ecdsa.PrivateKey
	stack    *common.MessagingStack

	publisher *pubsub.Publisher
	logger    *zap.Logger
}

func NewSender(
	identity *ecdsa.PrivateKey,
	stack *common.MessagingStack,
	logger *zap.Logger,
) *Sender {
	return &Sender{
		identity:  identity,
		stack:     stack,
		logger:    logger.Named("sender"),
		publisher: pubsub.NewPublisher(),
	}
}

func (s *Sender) Publisher() *pubsub.Publisher {
	return s.publisher
}

func (s *Sender) processAndMarshalMessageSpec(spec *encryption.ProtocolMessageSpec) ([]byte, []byte, error) {
	// The shared secret needs to be handle before we send a message
	// otherwise the topic might not be set up before we receive a message
	if spec.SharedSecret != nil {
		_, err := s.stack.Transport.ProcessNegotiatedSecret(ethtypes.NegotiatedSecret{
			PublicKey: spec.SharedSecret.Identity,
			Key:       spec.SharedSecret.Key,
		})
		if err != nil {
			return nil, nil, err
		}
	}

	var sharedSecretKey []byte
	if spec.AgreedSecret {
		sharedSecretKey = spec.SharedSecret.Key
	}

	messageBytes, err := proto.Marshal(spec.Message)
	if err != nil {
		return nil, nil, err
	}

	return messageBytes, sharedSecretKey, nil
}
