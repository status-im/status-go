package protocol

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/common/shard"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/services/mailservers"
)

func (m *Messenger) sendPublicSyncCommunityStorenodes(community *communities.Community, storenodes mailservers.Storenodes) error {
	if !community.IsControlNode() {
		return communities.ErrNotControlNode
	}

	clock, chat := m.getLastClockWithRelatedChat()
	pb := &protobuf.CommunityStorenodes{
		Clock:       clock,
		CommunityId: community.ID(),
		Storenodes:  storenodes.ToProtobuf(),
		ChainId:     communities.CommunityDescriptionTokenOwnerChainID(community.Description()),
	}
	snPayload, err := proto.Marshal(pb)
	if err != nil {
		return err
	}
	signature, err := crypto.Sign(crypto.Keccak256(snPayload), community.PrivateKey())
	if err != nil {
		return err
	}
	signedSyncCommunityStorenodes := &protobuf.SyncCommunityStorenodes{
		Signature: signature,
		Payload:   snPayload,
	}
	signedPayload, err := proto.Marshal(signedSyncCommunityStorenodes)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload:             signedPayload,
		Sender:              community.PrivateKey(),
		SkipEncryptionLayer: true,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_COMMUNITY_STORENODES,
		PubsubTopic:         shard.DefaultNonProtectedPubsubTopic(),
	}

	_, err = m.sender.SendPublic(context.Background(), chat.Name, rawMessage)
	return err
}

func (m *Messenger) HandleSyncCommunityStorenodes(state *ReceivedMessageState, a *protobuf.SyncCommunityStorenodes, statusMessage *v1protocol.StatusMessage) error {
	sn := &protobuf.CommunityStorenodes{}
	err := proto.Unmarshal(a.Payload, sn)
	if err != nil {
		return err
	}

	logError := func(err error) {
		m.logger.Error("HandlePublicSyncCommunityStorenodes failed: ", zap.Error(err), zap.String("communityID", types.EncodeHex(sn.CommunityId)))
	}

	err = m.verifyCommunitySignature(a.Payload, a.Signature, sn.CommunityId, sn.ChainId)
	if err != nil {
		logError(err)
		return err
	}

	if err := m.communityStorenodes.UpdateStorenodesInDB(sn.CommunityId, mailservers.FromProtobuf(sn.Storenodes)); err != nil {
		logError(err)
		return err
	}
	return nil
}
