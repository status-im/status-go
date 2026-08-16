package sender

import (
	"context"
	"crypto/ecdsa"
	"strings"

	"go.uber.org/zap"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	encryption "github.com/status-im/status-go/pkg/messaging/layers/encryption"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
	"github.com/status-im/status-go/pkg/pubsub"
)

const sdsForCommunitiesEnabled = true

type publicSDSWrapper interface {
	WrapPayloadForSDS(payload []byte, channelID string) (wrappedPayload []byte, messageID []byte, err error)
}

func wrapPayloadForPublicSDS(logger *zap.Logger, wrapper publicSDSWrapper, payload []byte, communityID string) ([]byte, []byte, error) {
	if len(communityID) == 0 {
		logger.Warn("SDS wrap skipped for public community payload due to missing community ID")
		return payload, nil, nil
	}

	sdsChannelID := reliability.BuildChannelID(communityID)
	sdsWrappedPayload, sdsMessageIDBytes, err := wrapper.WrapPayloadForSDS(payload, sdsChannelID)
	if err != nil {
		if strings.Contains(err.Error(), "reMessageTooLarge") {
			logger.Warn("SDS wrap skipped for oversized public community payload",
				zap.Int("payloadLength", len(payload)),
				zap.Error(err),
			)
			return payload, nil, nil
		}

		return payload, nil, errors.Wrap(err, "failed to wrap payload for SDS")
	}

	return sdsWrappedPayload, sdsMessageIDBytes, nil
}

func (s *Sender) SendPublic(ctx context.Context, params messagingtypes.SendPublicParams) error {
	messageID := messagingtypes.MessageID(params.Sender, params.Payload)
	var sdsMessageIDBytes []byte

	logger := s.logger.Named("sendPublic").With(
		zap.Stringer("messageID", messageID),
	)

	if params.CommunityPublicKey != nil {
		logger = logger.With(
			zap.String("community", cryptotypes.EncodeHex(crypto.CompressPubkey(params.CommunityPublicKey))),
		)
	}

	logger.Debug("sending public message",
		zap.String("sender", crypto.PubkeyToHex(params.Sender)),
		zap.String("pubsubTopic", params.PubsubTopic),
		zap.String("contentTopic", params.ContentTopic),
		zap.Any("hashRatchet", params.HashRatchet),
	)

	if sdsForCommunitiesEnabled {
		var wrapErr error
		communityID := params.CommunityID
		if len(communityID) == 0 && params.CommunityPublicKey != nil {
			communityID = cryptotypes.EncodeHex(crypto.CompressPubkey(params.CommunityPublicKey))
		}

		params.Payload, sdsMessageIDBytes, wrapErr = wrapPayloadForPublicSDS(logger, s.stack.Reliability, params.Payload, communityID)
		if wrapErr != nil {
			return wrapErr
		}
	}

	var err error
	var spec *encryption.ProtocolMessageSpec
	if params.HashRatchet != nil {
		if params.HashRatchet.Encrypt {
			logger.Debug("building hash ratchet message")
			spec, err = s.stack.Encryption.BuildHashRatchetMessage(params.HashRatchet.GroupID, params.Payload)
			if err != nil {
				return err
			}
		}

		payload := params.Payload
		if params.HashRatchet.KeyExType != messagingtypes.KeyExMsgNone {
			var ratchet *encryption.HashRatchetKeyCompatibility
			if params.HashRatchet.KeyExType == messagingtypes.KeyExMsgReuse {
				ratchet, err = s.stack.Encryption.GetCurrentKeyForGroup(params.HashRatchet.GroupID)
				if err != nil {
					return err
				}
			}

			// TODO: check if double-wrapping into encryption.ProtocolMessageSpec (Encrypt && (Rekey || Reuse) == true) actually works as intended
			if spec != nil {
				payload, err = proto.Marshal(spec.Message)
				if err != nil {
					return err
				}
			}

			logger.Debug("building hash ratchet rekey message")

			spec, err = s.stack.Encryption.BuildHashRatchetReKeyGroupMessage(s.identity, params.HashRatchet.Members, params.HashRatchet.GroupID, payload, ratchet)
			if err != nil {
				return err
			}
		}
	} else if !params.SkipEncryptionLayer {
		logger.Debug("wrapping public message in encryption layer")
		spec, err = s.stack.Encryption.BuildPublicMessage(s.identity, params.Payload)
		if err != nil {
			return errors.Wrap(err, "failed to wrap a public message in the encryption layer")
		}
	}

	payload := params.Payload
	if spec != nil {
		payload, err = proto.Marshal(spec.Message)
		if err != nil {
			return err
		}
	}

	hashes, wakuMessages, err := s.sendPublic(ctx, logger, sendPublicParams{
		payload:            payload,
		pubsubTopic:        params.PubsubTopic,
		contentTopic:       params.ContentTopic,
		ephemeral:          params.Ephemeral,
		priority:           params.Priority,
		communityPublicKey: params.CommunityPublicKey,
	})
	if err != nil {
		return errors.Wrap(err, "failed to send public message")
	}

	logger.Debug("sent-message",
		zap.Strings("hashes", cryptotypes.EncodeHexes(hashes)),
	)

	if len(sdsMessageIDBytes) > 0 {
		s.stack.Transport.TrackWithSDSAlias([]byte(messageID), sdsMessageIDBytes, hashes, wakuMessages)
	} else {
		s.stack.Transport.Track(messageID, hashes, wakuMessages)
	}

	if spec != nil {
		pubsub.Publish(s.publisher, SentMessage{
			Private:                false,
			RecipientInstallations: spec.Installations,
			MessageIDs: [][]byte{
				[]byte(messageID),
			},
		})
	}

	return nil
}

type sendPublicParams struct {
	payload            []byte
	pubsubTopic        string
	contentTopic       string
	ephemeral          bool
	priority           *messagingtypes.MessagePriority
	communityPublicKey *ecdsa.PublicKey
}

func (s *Sender) sendPublic(ctx context.Context, logger *zap.Logger, params sendPublicParams) ([][]byte, []*wakutypes.NewMessage, error) {
	segments, err := s.segmentMessage(params.payload)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to segment message")
	}

	ctx, span := s.tracer.Start(ctx, "Sender.sendPublic")
	defer span.End()

	hashes := make([][]byte, 0, len(segments))
	wakuMessages := make([]*wakutypes.NewMessage, len(segments))
	for i, segment := range segments {
		var hash []byte
		wakuMessage := &wakutypes.NewMessage{
			Payload:     segment,
			PubsubTopic: params.pubsubTopic,
			Ephemeral:   params.ephemeral,
			Priority:    params.priority,
		}
		wakuMessages[i] = wakuMessage
		if params.communityPublicKey != nil {
			logger.Debug("sending community public message")
			span.AddEvent("sending community public message")
			hash, err = s.stack.Transport.SendCommunityMessage(ctx, wakuMessage, params.communityPublicKey)
		} else {
			logger.Debug("sending public message")
			span.AddEvent("sending public message")
			hash, err = s.stack.Transport.SendPublic(ctx, wakuMessage, params.contentTopic)
		}
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to send message")
		}
		hashes = append(hashes, hash)
	}

	if s.tracer.Enabled() {
		linkSpanWithHashes(span, hashes)
	}

	return hashes, wakuMessages, nil
}
