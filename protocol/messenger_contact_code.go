package protocol

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
)

func (m *Messenger) buildContactCodeAdvertisement() (*protobuf.ContactCodeAdvertisement, error) {
	if m.pushNotificationClient == nil || !m.pushNotificationClient.Enabled() {
		return nil, nil
	}
	m.logger.Debug("adding push notification info to contact code bundle")
	info, err := m.pushNotificationClient.MyPushNotificationQueryInfo()
	if err != nil {
		return nil, err
	}
	if len(info) == 0 {
		return nil, nil
	}
	return &protobuf.ContactCodeAdvertisement{
		PushNotificationInfo: info,
	}, nil
}

func (m *Messenger) PublishIdentityImage() error {
	// Reset last published time for ChatIdentity so new contact can receive data
	err := m.resetLastPublishedTimeForChatIdentity()
	if err != nil {
		m.logger.Error("failed to reset publish time", zap.Error(err))
		return err
	}

	// If not online, we schedule it
	if !m.online() {
		m.shouldPublishContactCode = true
		return nil
	}

	return m.publishContactCode()
}

func (m *Messenger) buildContactCodeRawMessage() (common.RawMessage, error) {
	var payload []byte
	m.logger.Debug("sending contact code")
	contactCodeAdvertisement, err := m.buildContactCodeAdvertisement()
	if err != nil {
		m.logger.Error("could not build contact code advertisement", zap.Error(err))
	}

	if contactCodeAdvertisement == nil {
		contactCodeAdvertisement = &protobuf.ContactCodeAdvertisement{}
	}

	err = m.attachChatIdentity(contactCodeAdvertisement)
	if err != nil {
		return common.RawMessage{}, err
	}

	if contactCodeAdvertisement.ChatIdentity != nil {
		m.logger.Debug("attached chat identity", zap.Int("images len", len(contactCodeAdvertisement.ChatIdentity.Images)))
	} else {
		m.logger.Debug("no attached chat identity")
	}

	payload, err = proto.Marshal(contactCodeAdvertisement)
	if err != nil {
		return common.RawMessage{}, err
	}

	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	return common.RawMessage{
		LocalChatID: contactCodeTopic,
		MessageType: protobuf.ApplicationMetadataMessage_CONTACT_CODE_ADVERTISEMENT,
		Payload:     payload,
	}, nil

}

func (m *Messenger) publishContactCodeInCommunity(community *communities.Community) error {
	rawMessage, err := m.buildContactCodeRawMessage()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rawMessage.LocalChatID = community.MemberUpdateChannelID()
	rawMessage.PubsubTopic = community.PubsubTopic()
	_, err = m.sender.SendPublic(ctx, rawMessage.LocalChatID, rawMessage)
	return err
}

// publishContactCode sends a public message wrapped in the encryption
// layer, which will propagate our bundle
func (m *Messenger) publishContactCode() error {
	rawMessage, err := m.buildContactCodeRawMessage()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = m.sender.SendPublic(ctx, rawMessage.LocalChatID, rawMessage)
	if err != nil {
		m.logger.Warn("failed to send a contact code", zap.Error(err))
	}

	joinedCommunities, err := m.communitiesManager.Joined()
	if err != nil {
		return err
	}
	for _, community := range joinedCommunities {
		rawMessage.LocalChatID = community.MemberUpdateChannelID()
		rawMessage.PubsubTopic = community.PubsubTopic()
		_, err = m.sender.SendPublic(ctx, rawMessage.LocalChatID, rawMessage)
		if err != nil {
			return err
		}
	}

	m.logger.Debug("contact code sent")
	return err
}

// contactCodeAdvertisement attaches a protobuf.ChatIdentity to the given protobuf.ContactCodeAdvertisement,
// if the `shouldPublish` conditions are met
func (m *Messenger) attachChatIdentity(cca *protobuf.ContactCodeAdvertisement) error {
	contactCodeTopic := transport.ContactCodeTopic(&m.identity.PublicKey)
	shouldPublish, err := m.shouldPublishChatIdentity(contactCodeTopic)
	if err != nil {
		return err
	}

	if !shouldPublish {
		return nil
	}

	cca.ChatIdentity, err = m.createChatIdentity(privateChat)
	if err != nil {
		return err
	}

	img, err := m.multiAccounts.GetIdentityImage(m.account.KeyUID, images.SmallDimName)
	if err != nil {
		return err
	}

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	bio, err := m.settings.Bio()
	if err != nil {
		return err
	}

	socialLinks, err := m.settings.GetSocialLinks()
	if err != nil {
		return err
	}

	identityHash, err := m.getIdentityHash(displayName, bio, img, socialLinks)
	if err != nil {
		return err
	}

	err = m.persistence.SaveWhenChatIdentityLastPublished(contactCodeTopic, identityHash)
	if err != nil {
		return err
	}

	return nil
}
