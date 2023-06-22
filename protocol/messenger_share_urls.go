package protocol

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/urls"
)

const baseShareUrl = "https://status.app/%s%s%s"

func (m *Messenger) CreateCommunityURLWithChatKey(communityID string) (string, error) {
	if len(communityID) == 0 {
		return "", ErrChatIDEmpty
	}

	community, err := m.GetCommunityByID(types.HexBytes(communityID))

	if err != nil {
		return "", err
	}

	if community == nil {
		return "", errors.New("community is nil")
	}

	pubKey, err := common.HexToPubkey(communityID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(baseShareUrl, "/c", "#", types.EncodeHex(crypto.CompressPubkey(pubKey))), nil
}

func (m *Messenger) CreateCommunityURLWithData(communityID string) (string, error) {
	if len(communityID) == 0 {
		return "", ErrChatIDEmpty
	}

	community, err := m.GetCommunityByID(types.HexBytes(communityID))

	if err != nil {
		return "", err
	}

	if community == nil {
		return "", errors.New("community is nil")
	}

	if community.Encrypted() {
		pubKey, err := common.HexToPubkey(communityID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(baseShareUrl, "/c", "#", types.EncodeHex(crypto.CompressPubkey(pubKey))), nil
	}

	communityProto := &protobuf.Community{
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
	}

	data, err := json.Marshal(communityProto)
	if err != nil {
		return "", err
	}

	communityBase64, err := urls.EncodeDataURL(data)
	if err != nil {
		return "", err
	}

	signature, err := crypto.SignBytes([]byte(communityBase64), community.PrivateKey())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(baseShareUrl, "/c", "/", communityBase64+"#"+string(signature)), nil
}

func (m *Messenger) CreateCommunityChannelURLWithChatKey(request *requests.CommunityChannelShareURL) (string, error) {
	if err := request.Validate(); err != nil {
		return "", err
	}

	community, err := m.GetCommunityByID(request.CommunityID)

	if err != nil {
		return "", err
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(baseShareUrl, "/cc", "/", uuid.String()+"#"+types.EncodeHex(crypto.CompressPubkey(community.PublicKey()))), nil
}

func (m *Messenger) CreateCommunityChannelURLWithData(request *requests.CommunityChannelShareURL) (string, error) {
	if err := request.Validate(); err != nil {
		return "", err
	}

	community, err := m.GetCommunityByID(request.CommunityID)

	if err != nil {
		return "", err
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	chat := community.Chats()[request.ChannelID]

	communityProto := &protobuf.Community{
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
	}

	chatProto := &protobuf.Channel{
		DisplayName: chat.GetIdentity().DisplayName,
		Description: chat.GetIdentity().Description,
		Emoji:       chat.Identity.Emoji,
		Color:       chat.GetIdentity().Color,
		Community:   communityProto,
		Uuid:        uuid.String(),
	}

	data, err := json.Marshal(chatProto)
	if err != nil {
		return "", err
	}

	chatBase64, err := urls.EncodeDataURL(data)
	if err != nil {
		return "", err
	}

	signature, err := crypto.SignBytes([]byte(chatBase64), community.PrivateKey())
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(baseShareUrl, "/cc", "/", request.ChannelID+"#"+string(signature)), nil
}

func (m *Messenger) CreateUserURLWithChatKey(pubKey string) (string, error) {
	if len(pubKey) == 0 {
		return "", errors.New("pubkey is empty")
	}
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		return "", ErrContactNotFound
	}
	pubkey, err := contact.PublicKey()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(baseShareUrl, "/u", "#", types.EncodeHex(crypto.CompressPubkey(pubkey))), nil
}

func (m *Messenger) CreateUserURLWithENS(pubKey string) (string, error) {
	if len(pubKey) == 0 {
		return "", errors.New("pubkey is empty")
	}
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		return "", ErrContactNotFound
	}
	return fmt.Sprintf(baseShareUrl, "/u", "#", contact.EnsName), nil
}

func (m *Messenger) CreateUserURLWithData(pubKey string) (string, error) {
	if len(pubKey) == 0 {
		return "", errors.New("pubkey is empty")
	}
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		return "", ErrContactNotFound
	}

	userProto := &protobuf.User{
		DisplayName: contact.DisplayName,
	}

	data, err := json.Marshal(userProto)
	if err != nil {
		return "", err
	}

	userBase64, err := urls.EncodeDataURL(data)
	if err != nil {
		return "", err
	}

	signature, err := crypto.SignBytes([]byte(userBase64), m.identity)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(baseShareUrl, "/u", "/", userBase64+"#"+string(signature)), nil

}
