package protocol

import (
	"context"
	"crypto/ecdsa"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/encryption"
	"github.com/status-im/status-go/protocol/protobuf"
)

type CommunitiesKeyDistributor interface {
	Distribute(community *communities.Community, keyActions *communities.EncryptionKeyActions) error
	Rekey(community *communities.Community) error
}

type CommunitiesKeyDistributorImpl struct {
	sender    *common.MessageSender
	encryptor *encryption.Protocol
}

func (ckd *CommunitiesKeyDistributorImpl) Distribute(community *communities.Community, keyActions *communities.EncryptionKeyActions) error {
	if !community.IsControlNode() {
		return communities.ErrNotControlNode
	}

	err := ckd.distributeKey(community, community.ID(), &keyActions.CommunityKeyAction)
	if err != nil {
		return err
	}

	for channelID := range keyActions.ChannelKeysActions {
		keyAction := keyActions.ChannelKeysActions[channelID]
		err := ckd.distributeKey(community, []byte(community.IDString()+channelID), &keyAction)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ckd *CommunitiesKeyDistributorImpl) Rekey(community *communities.Community) error {
	if !community.IsControlNode() {
		return communities.ErrNotControlNode
	}

	err := ckd.distributeKey(community, community.ID(), &communities.EncryptionKeyAction{
		ActionType: communities.EncryptionKeyRekey,
		Members:    community.Members(),
	})
	if err != nil {
		return err
	}

	for channelID, channel := range community.Chats() {
		err := ckd.distributeKey(community, []byte(community.IDString()+channelID), &communities.EncryptionKeyAction{
			ActionType: communities.EncryptionKeyRekey,
			Members:    channel.Members,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (ckd *CommunitiesKeyDistributorImpl) distributeKey(community *communities.Community, hashRatchetGroupID []byte, keyAction *communities.EncryptionKeyAction) error {
	pubkeys := make([]*ecdsa.PublicKey, len(keyAction.Members))
	i := 0
	for hex := range keyAction.Members {
		pubkeys[i], _ = common.HexToPubkey(hex)
		i++
	}

	switch keyAction.ActionType {
	case communities.EncryptionKeyAdd:
		_, err := ckd.encryptor.GenerateHashRatchetKey(hashRatchetGroupID)
		if err != nil {
			return err
		}

		err = ckd.sendKeyExchangeMessage(community, hashRatchetGroupID, pubkeys, common.KeyExMsgReuse)
		if err != nil {
			return err
		}

	case communities.EncryptionKeyRekey:
		err := ckd.sendKeyExchangeMessage(community, hashRatchetGroupID, pubkeys, common.KeyExMsgRekey)
		if err != nil {
			return err
		}

	case communities.EncryptionKeySendToMembers:
		err := ckd.sendKeyExchangeMessage(community, hashRatchetGroupID, pubkeys, common.KeyExMsgReuse)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ckd *CommunitiesKeyDistributorImpl) sendKeyExchangeMessage(community *communities.Community, hashRatchetGroupID []byte, pubkeys []*ecdsa.PublicKey, msgType common.CommKeyExMsgType) error {
	rawMessage := common.RawMessage{
		SkipProtocolLayer:     false,
		CommunityID:           community.ID(),
		CommunityKeyExMsgType: msgType,
		Recipients:            pubkeys,
		MessageType:           protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
		HashRatchetGroupID:    hashRatchetGroupID,
		PubsubTopic:           community.PubsubTopic(), // TODO: confirm if it should be sent in community pubsub topic
	}
	_, err := ckd.sender.SendCommunityMessage(context.Background(), rawMessage)

	if err != nil {
		return err
	}
	return nil
}
