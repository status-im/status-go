package sender

import (
	"crypto/ecdsa"
	"runtime/debug"

	"github.com/golang/protobuf/proto"
	otelattribute "go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	cryptotypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/pkg/messaging/common"
	"github.com/status-im/status-go/pkg/messaging/controller/utils"
	"github.com/status-im/status-go/pkg/messaging/layers/encryption"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

type Sender struct {
	identity *ecdsa.PrivateKey
	stack    *common.MessagingStack

	publisher *pubsub.Publisher
	logger    *zap.Logger
	tracer    trace.Tracer
}

func NewSender(
	identity *ecdsa.PrivateKey,
	stack *common.MessagingStack,
	logger *zap.Logger,
	tracer trace.Tracer,
) *Sender {
	return &Sender{
		identity:  identity,
		stack:     stack,
		logger:    logger.Named("sender"),
		tracer:    tracer,
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
		_, err := s.stack.Transport.ProcessNegotiatedSecret(messagingtypes.NegotiatedSecret{
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

func linkSpanWithHashes(span oteltrace.Span, hashes [][]byte) {
	span.SetAttributes(
		otelattribute.StringSlice("hashes", cryptotypes.EncodeHexes(hashes)),
		otelattribute.String("stack", string(debug.Stack())),
	)
	linkSpanCtx := trace.DeriveSpanContext(utils.MergeByteSlices(hashes), false)
	span.AddLink(oteltrace.Link{SpanContext: linkSpanCtx})
}
