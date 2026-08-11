package protocol

import (
	"context"
	"crypto/ecdsa"

	otelattribute "go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/instrumentation/trace"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
)

type CommunitiesKeyDistributorImpl struct {
	messaging            *messaging.API
	sender               *common.MessageSender
	tracer               trace.Tracer
	addRawMessageToWatch func(*common.RawMessage) (*common.RawMessage, error)
}

func (ckd *CommunitiesKeyDistributorImpl) Generate(community *communities.Community, keyActions *communities.EncryptionKeyActions) error {
	if !community.IsControlNode() {
		return communities.ErrNotControlNode
	}
	return iterateActions(community, keyActions, ckd.generateKey)
}

func (ckd *CommunitiesKeyDistributorImpl) Distribute(community *communities.Community, keyActions *communities.EncryptionKeyActions) error {
	if !community.IsControlNode() {
		return communities.ErrNotControlNode
	}
	return iterateActions(community, keyActions, ckd.distributeKey)
}

func iterateActions(community *communities.Community, keyActions *communities.EncryptionKeyActions, fn func(community *communities.Community, hashRatchetGroupID []byte, keyAction *communities.EncryptionKeyAction) error) error {
	err := fn(community, community.ID(), &keyActions.CommunityKeyAction)
	if err != nil {
		return err
	}

	for channelID := range keyActions.ChannelKeysActions {
		keyAction := keyActions.ChannelKeysActions[channelID]
		err := fn(community, []byte(community.IDString()+channelID), &keyAction)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ckd *CommunitiesKeyDistributorImpl) generateKey(community *communities.Community, hashRatchetGroupID []byte, keyAction *communities.EncryptionKeyAction) error {
	if keyAction.ActionType != communities.EncryptionKeyAdd {
		return nil
	}
	return ckd.messaging.GenerateHashRatchetKey(hashRatchetGroupID)
}

func (ckd *CommunitiesKeyDistributorImpl) distributeKey(community *communities.Community, hashRatchetGroupID []byte, keyAction *communities.EncryptionKeyAction) error {
	pubkeys := make([]*ecdsa.PublicKey, len(keyAction.Members))
	i := 0
	for hex := range keyAction.Members {
		pubkeys[i], _ = crypto.HexToPubkey(hex)
		i++
	}

	switch keyAction.ActionType {
	case communities.EncryptionKeyAdd:
		// key must be already generated
		err := ckd.sendKeyExchangeMessage(community, hashRatchetGroupID, pubkeys, messagingtypes.KeyExMsgReuse)
		if err != nil {
			return err
		}

	case communities.EncryptionKeyRekey:
		err := ckd.sendKeyExchangeMessage(community, hashRatchetGroupID, pubkeys, messagingtypes.KeyExMsgRekey)
		if err != nil {
			return err
		}

	case communities.EncryptionKeySendToMembers:
		err := ckd.sendKeyExchangeMessage(community, hashRatchetGroupID, pubkeys, messagingtypes.KeyExMsgReuse)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ckd *CommunitiesKeyDistributorImpl) sendKeyExchangeMessage(community *communities.Community, hashRatchetGroupID []byte, pubkeys []*ecdsa.PublicKey, msgType messagingtypes.CommKeyExMsgType) error {
	rawMessage := common.RawMessage{
		Sender:                community.PrivateKey(),
		SkipEncryptionLayer:   false,
		CommunityID:           community.ID(),
		CommunityKeyExMsgType: msgType,
		Recipients:            pubkeys,
		MessageType:           protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		HashRatchetGroupID:    hashRatchetGroupID,
		PubsubTopic:           community.PubsubTopic(), // TODO: confirm if it should be sent in community pubsub topic
		ResendType:            common.ResendTypeRawMessage,
		ResendMethod:          common.ResendMethodSendCommunityMessage,
	}

	ctx, span := ckd.tracer.Start(context.Background(), "CommunitiesKeyDistributor.sendKeyExchangeMessage",
		oteltrace.WithAttributes(
			otelattribute.String("groupID", types.ToHex(hashRatchetGroupID)),
			otelattribute.StringSlice("recipients", crypto.PubkeysToHex(pubkeys)),
		),
	)
	defer span.End()

	_, err := ckd.sender.SendCommunity(ctx, &rawMessage)
	if err != nil {
		return err
	}
	if ckd.addRawMessageToWatch != nil {
		_, err = ckd.addRawMessageToWatch(&rawMessage)
	}
	return nil
}
