package sender

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/crypto"
	cryptotypes "github.com/status-im/status-go/crypto/types"
	"github.com/status-im/status-go/messaging/layers/encryption"
	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/messaging/waku/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

func (s *Sender) SendPrivate(ctx context.Context, params types.SendPrivateParams) error {
	messageID := types.MessageID(&params.Sender.PublicKey, params.Payload)

	logger := s.logger.Named("sendPrivate").With(
		zap.Stringer("messageID", messageID),
		zap.String("recipient", crypto.PubkeyToHex(params.Recipient)),
		zap.String("pubsubTopic", params.PubsubTopic),
	)

	if params.WithReliability {
		logger.Debug("scheduling reliable send")
		reliabilityIDs, err := s.scheduleReliableSend(params.Recipient, params.Payload)
		if err == nil {

			pubsub.Publish(s.publisher, ScheduledReliableSend{
				Recipient:            params.Recipient,
				MessageID:            messageID,
				ReliabilityMessageID: reliabilityIDs,
			})

			return nil
		} else {
			s.logger.Error("failed to schedule reliable send", zap.Error(err))
		}
	}

	var spec *encryption.ProtocolMessageSpec
	var err error
	if len(params.HashRatchetGroupID) != 0 {
		spec, err = s.buildEncryptedMessageAttachHashRatchetKeys(params.Sender, params.Recipient, params.Payload, params.HashRatchetGroupID)
		if err != nil {
			return errors.Wrap(err, "failed to build encrypted message with hash ratchet keys")
		}
	} else if params.SendWithDH {
		spec, err = s.buildDHMessage(params.Sender, params.Recipient, params.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to build DH message")
		}
	} else if !params.SkipEncryptionLayer {
		spec, err = s.buildEncryptedMessage(params.Sender, params.Recipient, params.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to build encrypted message")
		}
	}

	payload := params.Payload
	var sharedSecretKey []byte
	if spec != nil {
		payload, sharedSecretKey, err = s.processAndMarshalMessageSpec(spec)
		if err != nil {
			return errors.Wrap(err, "failed to process and marshal message spec")
		}
	}

	hashes, wakuMessages, err := s.sendPrivate(ctx, logger, sendPrivateParams{
		recipient:           params.Recipient,
		payload:             payload,
		pubsubTopic:         params.PubsubTopic,
		sendOnPersonalTopic: params.SendOnPersonalTopic,
		sharedSecretKey:     sharedSecretKey,
	})
	if err != nil {
		return errors.Wrap(err, "failed to send private message")
	}

	logger.Debug("sent-message",
		zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)),
	)

	s.stack.Transport.Track(messageID, hashes, wakuMessages)

	if spec != nil {
		pubsub.Publish(s.publisher, SentMessage{
			Private:                true,
			Recipient:              params.Recipient,
			RecipientInstallations: spec.Installations,
			MessageIDs: [][]byte{
				[]byte(messageID),
			},
		})
	}

	return nil
}

func (s *Sender) SendPrivateHashRatchetKeys(ctx context.Context, recipients []*ecdsa.PublicKey, groupID []byte) error {
	keyExMessageSpecs, err := s.stack.Encryption.GetKeyExMessageSpecs(groupID, s.identity, recipients, false)
	if err != nil {
		return err
	}

	for i, spec := range keyExMessageSpecs {
		logger := s.logger.Named("sendPrivateHashRatchetKeys").With(
			zap.String("recipient", crypto.PubkeyToHex(recipients[i])),
			zap.Stringer("groupID", cryptotypes.HexBytes(groupID)),
		)

		payload, sharedSecretKey, err := s.processAndMarshalMessageSpec(spec)
		if err != nil {
			return err
		}

		hashes, _, err := s.sendPrivate(ctx, logger, sendPrivateParams{
			recipient:       recipients[i],
			payload:         payload,
			sharedSecretKey: sharedSecretKey,
		})
		if err != nil {
			return err
		}

		logger.Debug("sent-message",
			zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)),
		)
	}

	return nil
}

func (s *Sender) SendPrivateAdvertiseBundle(ctx context.Context, publicKey *ecdsa.PublicKey) error {
	now := time.Now().Unix()
	advertise, err := s.stack.Encryption.ShouldAdvertiseBundle(publicKey, now)
	if err != nil {
		return err
	}
	if !advertise {
		return nil
	}

	spec, err := s.stack.Encryption.BuildBundleAdvertiseMessage(s.identity, publicKey)
	if err != nil {
		return err
	}

	payload, sharedSecretKey, err := s.processAndMarshalMessageSpec(spec)
	if err != nil {
		return err
	}

	logger := s.logger.Named("sendPrivateAdvertiseBundle").With(
		zap.String("recipient", crypto.PubkeyToHex(publicKey)),
	)

	_, _, err = s.sendPrivate(ctx, logger, sendPrivateParams{
		recipient:       publicKey,
		payload:         payload,
		sharedSecretKey: sharedSecretKey,
	})
	if err != nil {
		return err
	}

	s.stack.Encryption.ConfirmBundleAdvertisement(publicKey, now)

	return nil
}

type sendPrivateParams struct {
	recipient           *ecdsa.PublicKey
	payload             []byte
	pubsubTopic         string
	sendOnPersonalTopic bool
	sharedSecretKey     []byte
}

func (s *Sender) sendPrivate(ctx context.Context, logger *zap.Logger, params sendPrivateParams) ([][]byte, []*wakutypes.NewMessage, error) {
	segments, err := s.segmentMessage(params.payload)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to segment message")
	}

	hashes := make([][]byte, 0, len(segments))
	wakuMessages := make([]*wakutypes.NewMessage, len(segments))
	for i, segment := range segments {
		var hash []byte
		wakuMessage := &wakutypes.NewMessage{
			Payload:     segment,
			PubsubTopic: params.pubsubTopic,
		}
		wakuMessages[i] = wakuMessage
		if params.sendOnPersonalTopic {
			logger.Debug("sending on personal topic")
			hash, err = s.stack.Transport.SendPrivateOnPersonalTopic(ctx, wakuMessage, params.recipient)
		} else if params.sharedSecretKey != nil {
			logger.Debug("sending on shared secret topic")
			hash, err = s.stack.Transport.SendPrivateWithSharedSecret(ctx, wakuMessage, params.recipient, params.sharedSecretKey)
		} else {
			logger.Debug("sending on partitioned topic")
			hash, err = s.stack.Transport.SendPrivateWithPartitioned(ctx, wakuMessage, params.recipient)
		}
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to send message")
		}
		hashes = append(hashes, hash)
	}

	return hashes, wakuMessages, nil
}

func (s *Sender) buildEncryptedMessage(sender *ecdsa.PrivateKey, recipient *ecdsa.PublicKey, payload []byte) (*encryption.ProtocolMessageSpec, error) {
	return s.stack.Encryption.BuildEncryptedMessage(sender, recipient, payload)
}

func (s *Sender) buildEncryptedMessageAttachHashRatchetKeys(sender *ecdsa.PrivateKey, recipient *ecdsa.PublicKey, payload []byte, groupID []byte) (*encryption.ProtocolMessageSpec, error) {
	ratchets, err := s.stack.Encryption.GetKeysForGroup(groupID)
	if err != nil {
		return nil, err
	}

	return s.stack.Encryption.BuildHashRatchetKeyExchangeMessageWithPayload(s.identity, recipient, groupID, ratchets, payload)
}

func (s *Sender) buildDHMessage(sender *ecdsa.PrivateKey, recipient *ecdsa.PublicKey, payload []byte) (*encryption.ProtocolMessageSpec, error) {
	return s.stack.Encryption.BuildDHMessage(sender, recipient, payload)
}
