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
	"github.com/status-im/status-go/protocol/storenodes"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

func (m *Messenger) sendCommunityPublicStorenodesInfo(community *communities.Community, snodes storenodes.Storenodes) error {
	if !community.IsControlNode() {
		return communities.ErrNotControlNode
	}

	clock, chat := m.getLastClockWithRelatedChat()
	pb := &protobuf.CommunityStorenodes{
		Clock:       clock,
		CommunityId: community.ID(),
		Storenodes:  snodes.ToProtobuf(),
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
	signedStorenodesInfo := &protobuf.CommunityPublicStorenodesInfo{
		Signature: signature,
		Payload:   snPayload,
	}
	signedPayload, err := proto.Marshal(signedStorenodesInfo)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload:             signedPayload,
		Sender:              community.PrivateKey(),
		SkipEncryptionLayer: true,
		MessageType:         protobuf.ApplicationMetadataMessage_COMMUNITY_PUBLIC_STORENODES_INFO,
		PubsubTopic:         shard.DefaultNonProtectedPubsubTopic(),
	}

	_, err = m.sender.SendPublic(context.Background(), chat.Name, rawMessage)
	return err
}

func (m *Messenger) HandleCommunityPublicStorenodesInfo(state *ReceivedMessageState, a *protobuf.CommunityPublicStorenodesInfo, statusMessage *v1protocol.StatusMessage) error {
	sn := &protobuf.CommunityStorenodes{}
	err := proto.Unmarshal(a.Payload, sn)
	if err != nil {
		return err
	}

	logError := func(err error) {
		m.logger.Error("HandleCommunityPublicStorenodesInfo failed: ", zap.Error(err), zap.String("communityID", types.EncodeHex(sn.CommunityId)))
	}

	err = m.verifyCommunitySignature(a.Payload, a.Signature, sn.CommunityId, sn.ChainId)
	if err != nil {
		logError(err)
		return err
	}

	if err := m.communityStorenodes.UpdateStorenodesInDB(sn.CommunityId, storenodes.FromProtobuf(sn.Storenodes, sn.Clock), sn.Clock); err != nil {
		logError(err)
		return err
	}
	return nil
}
