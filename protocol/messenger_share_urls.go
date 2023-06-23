package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"

	"github.com/google/uuid"
	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/urls"
)

const baseShareUrl = "https://status.app"

func (m *Messenger) SerializePublicKey(compressedKey types.HexBytes) (string, error) {
	rawKey, err := crypto.DecompressPubkey(compressedKey)
	if err != nil {
		return "", err
	}
	pubKey := types.EncodeHex(crypto.FromECDSAPub(rawKey))

	secp256k1Code := "0xe701"
	base58btc := "z"
	multiCodecKey := secp256k1Code + strings.TrimPrefix(pubKey, "0x")
	cpk, err := multiformat.SerializePublicKey(multiCodecKey, base58btc)
	if err != nil {
		return "", err
	}
	return cpk, nil
}

func (m *Messenger) DeserializePublicKey(compressedKey string) (types.HexBytes, error) {
	rawKey, err := multiformat.DeserializePublicKey(compressedKey, "f")
	if err != nil {
		return nil, err
	}

	secp256k1Code := "fe701"
	pubKeyBytes := "0x" + strings.TrimPrefix(rawKey, secp256k1Code)

	pubKey, err := common.HexToPubkey(pubKeyBytes)
	if err != nil {
		return nil, err
	}

	return crypto.CompressPubkey(pubKey), nil
}

func (m *Messenger) CreateCommunityURLWithChatKey(communityID types.HexBytes) (string, error) {
	if len(communityID) == 0 {
		return "", ErrChatIDEmpty
	}

	community, err := m.GetCommunityByID(communityID)
	if err != nil {
		return "", err
	}

	if community == nil {
		return "", fmt.Errorf("community with communityID %s not found", communityID)
	}

	pubKey, err := m.SerializePublicKey(communityID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s%s", baseShareUrl, "c#", pubKey), nil
}

func (m *Messenger) parseCommunityURLWithChatKey(urlData string) (*MessengerResponse, error) {
	communityID, err := m.DeserializePublicKey(urlData)
	if err != nil {
		return nil, err
	}

	community, err := m.GetCommunityByID(communityID)
	if err != nil {
		return nil, err
	}

	if community == nil {
		return nil, fmt.Errorf("community with communityID %s not found", communityID)
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	return response, nil
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
		return fmt.Sprintf("%s/%s/%s", baseShareUrl, "c", types.EncodeHex(crypto.CompressPubkey(pubKey))), nil
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

	return fmt.Sprintf("%s/%s/%s#%s", baseShareUrl, "c", communityBase64, string(signature)), nil
}

func (m *Messenger) parseCommunityURLWithData(urlData string) (*MessengerResponse, error) {
	var communityProto protobuf.Community
	err := proto.Unmarshal([]byte(urlData), &communityProto)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	// TODO: response.AddCommunity(community)
	return response, nil
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

	return fmt.Sprintf("%s/%s/%s#%s", baseShareUrl, "cc", uuid.String(), types.EncodeHex(crypto.CompressPubkey(community.PublicKey()))), nil
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

	return fmt.Sprintf("%s/%s/%s#%s", baseShareUrl, "/cc", request.ChannelID, string(signature)), nil
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
	return fmt.Sprintf("%s/%s/%s", baseShareUrl, "u#", types.EncodeHex(crypto.CompressPubkey(pubkey))), nil
}

func (m *Messenger) CreateUserURLWithENS(pubKey string) (string, error) {
	if len(pubKey) == 0 {
		return "", errors.New("pubkey is empty")
	}
	contact, ok := m.allContacts.Load(pubKey)
	if !ok {
		return "", ErrContactNotFound
	}
	return fmt.Sprintf("%s/%s/%s", baseShareUrl, "/u#", contact.EnsName), nil
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

	return fmt.Sprintf("%s/%s/%s#%s", baseShareUrl, "u", userBase64, string(signature)), nil
}

func (m *Messenger) parseCommuntyChannelSharedURL(url string) (*MessengerResponse, error) {
	return nil, errors.New("not implemented yet")
}

func (m *Messenger) parseProfileSharedURL(urlData string) (*MessengerResponse, error) {
	// TODO: decompress public key
	pubKey, err := common.HexToPubkey(urlData)
	if err != nil {
		return nil, err
	}

	contact, err := m.BuildContact(&requests.BuildContact{PublicKey: types.EncodeHex(crypto.FromECDSAPub(pubKey))})
	if err != nil {
		return nil, err
	}

	if contact == nil {
		return nil, fmt.Errorf("contact with publick key %s not found", pubKey)
	}

	response := &MessengerResponse{}
	response.AddContact(contact)
	return response, nil
}

func (m *Messenger) ParseSharedURL(url string) (*MessengerResponse, error) {
	if !strings.HasPrefix(url, baseShareUrl) {
		return nil, fmt.Errorf("url should start with '%s'", baseShareUrl)
	}
	urlContents := strings.Split(strings.TrimPrefix(url, baseShareUrl+"/"), "#")

	fmt.Println("-----> url: ", url, "urlContents: ", urlContents)

	if len(urlContents) == 2 && urlContents[0] == "c" {
		return m.parseCommunityURLWithChatKey(urlContents[1])
	}

	// switch group {
	// case "c":
	// 	return m.parseCommunityURLWithData(urlData)
	// case "c#":
	// 	return m.parseCommunityURLWithChatKey(urlData)
	// case "cc":
	// 	return m.parseCommuntyChannelSharedURL(urlData)
	// case "u":
	// 	return m.parseProfileSharedURL(urlData)
	// }

	return nil, errors.New("unhandled url group")
}
