package protocol

import (
	"crypto/ecdsa"
	"fmt"
	"regexp"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/status-im/status-go/api/multiformat"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/urls"
)

type CommunityURLData struct {
	DisplayName  string `json:"displayName"`
	Description  string `json:"description"`
	MembersCount uint32 `json:"membersCount"`
	Color        string `json:"color"`
}

type CommunityChannelURLData struct {
	Emoji       string `json:"emoji"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type ContactURLData struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

type URLDataResponse struct {
	Community CommunityURLData        `json:"community"`
	Channel   CommunityChannelURLData `json:"channel"`
	Contact   ContactURLData          `json:"contact"`
}

const baseShareURL = "https://status.app"
const channelUUIDRegExp = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-5][0-9a-f]{3}-[089ab][0-9a-f]{3}-[0-9a-f]{12}$"

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

func (m *Messenger) ShareCommunityURLWithChatKey(communityID types.HexBytes) (string, error) {
	shortKey, err := m.SerializePublicKey(communityID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/c#%s", baseShareURL, shortKey), nil
}

func (m *Messenger) prepareCommunityData(community *communities.Community) CommunityURLData {
	return CommunityURLData{
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
	}
}

func (m *Messenger) parseCommunityURLWithChatKey(urlData string) (*URLDataResponse, error) {
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

	return &URLDataResponse{
		Community: m.prepareCommunityData(community),
	}, nil
}

func (m *Messenger) prepareEncodedRawData(rawData []byte, privateKey *ecdsa.PrivateKey) (string, string, error) {
	encodedData, err := urls.EncodeDataURL(rawData)
	if err != nil {
		return "", "", err
	}

	signature, err := crypto.SignBytes([]byte(encodedData), privateKey)
	if err != nil {
		return "", "", err
	}

	encodedSignature, err := urls.EncodeDataURL(signature)
	if err != nil {
		return "", "", err
	}

	return encodedData, encodedSignature, nil
}

func (m *Messenger) prepareEncodedCommunityData(community *communities.Community) (string, string, error) {
	communityProto := &protobuf.Community{
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
		TagIndices:   community.TagsIndices(),
	}

	communityData, err := proto.Marshal(communityProto)
	if err != nil {
		return "", "", err
	}

	return m.prepareEncodedRawData(communityData, community.PrivateKey())
}

func (m *Messenger) ShareCommunityURLWithData(communityID types.HexBytes) (string, error) {
	community, err := m.GetCommunityByID(communityID)
	if err != nil {
		return "", err
	}

	if community == nil {
		return "", fmt.Errorf("community with communityID %s not found", communityID)
	}

	data, signature, err := m.prepareEncodedCommunityData(community)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/c/%s#%s", baseShareURL, data, signature), nil
}

func (m *Messenger) verifySignature(data string, rawSignature string) (*ecdsa.PublicKey, error) {
	signature, err := urls.DecodeDataURL(rawSignature)
	if err != nil {
		return nil, err
	}

	return crypto.SigToPub(crypto.Keccak256([]byte(data)), signature)
}

func (m *Messenger) parseCommunityURLWithData(data string, signature string) (*URLDataResponse, error) {
	_, err := m.verifySignature(data, signature)
	if err != nil {
		return nil, err
	}

	communityData, err := urls.DecodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var communityProto protobuf.Community
	err = proto.Unmarshal(communityData, &communityProto)
	if err != nil {
		return nil, err
	}

	return &URLDataResponse{
		Community: CommunityURLData{
			DisplayName:  communityProto.DisplayName,
			Description:  communityProto.Description,
			MembersCount: communityProto.MembersCount,
			Color:        communityProto.Color,
		},
	}, nil
}

func (m *Messenger) ShareCommunityChannelURLWithChatKey(request *requests.CommunityChannelShareURL) (string, error) {
	if err := request.Validate(); err != nil {
		return "", err
	}

	shortKey, err := m.SerializePublicKey(request.CommunityID)
	if err != nil {
		return "", err
	}

	valid, err := regexp.MatchString(channelUUIDRegExp, request.ChannelID)
	if err != nil {
		return "", err
	}

	if !valid {
		return "", fmt.Errorf("channelID should be UUID, got %s", request.ChannelID)
	}

	return fmt.Sprintf("%s/cc/%s#%s", baseShareURL, request.ChannelID, shortKey), nil
}

func (m *Messenger) prepareCommunityChannelData(channel *protobuf.CommunityChat) CommunityChannelURLData {
	return CommunityChannelURLData{
		Emoji:       channel.Identity.Emoji,
		DisplayName: channel.Identity.DisplayName,
		Description: channel.Identity.Description,
		Color:       channel.Identity.Color,
	}
}

func (m *Messenger) parseCommunityChannelURLWithChatKey(channelID string, publickKey string) (*URLDataResponse, error) {
	valid, err := regexp.MatchString(channelUUIDRegExp, channelID)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, fmt.Errorf("channelID should be UUID, got %s", channelID)
	}

	communityID, err := m.DeserializePublicKey(publickKey)
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

	channel, ok := community.Chats()[channelID]
	if !ok {
		return nil, fmt.Errorf("channel with channelID %s not found", channelID)
	}

	return &URLDataResponse{
		Community: m.prepareCommunityData(community),
		Channel:   m.prepareCommunityChannelData(channel),
	}, nil
}

func (m *Messenger) prepareEncodedCommunityChannelData(community *communities.Community, channel *protobuf.CommunityChat, channelID string) (string, string, error) {
	communityProto := &protobuf.Community{
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
	}

	channelProto := &protobuf.Channel{
		DisplayName: channel.Identity.DisplayName,
		Description: channel.Identity.Description,
		Emoji:       channel.Identity.Emoji,
		Color:       channel.GetIdentity().Color,
		Community:   communityProto,
		Uuid:        channelID,
	}

	channelData, err := proto.Marshal(channelProto)
	if err != nil {
		return "", "", err
	}

	return m.prepareEncodedRawData(channelData, community.PrivateKey())
}

func (m *Messenger) ShareCommunityChannelURLWithData(request *requests.CommunityChannelShareURL) (string, error) {
	if err := request.Validate(); err != nil {
		return "", err
	}

	valid, err := regexp.MatchString(channelUUIDRegExp, request.ChannelID)
	if err != nil {
		return "", err
	}

	if !valid {
		return "nil", fmt.Errorf("channelID should be UUID, got %s", request.ChannelID)
	}

	community, err := m.GetCommunityByID(request.CommunityID)
	if err != nil {
		return "", err
	}

	channel := community.Chats()[request.ChannelID]
	if channel == nil {
		return "", fmt.Errorf("channel with channelID %s not found", request.ChannelID)
	}

	data, signature, err := m.prepareEncodedCommunityChannelData(community, channel, request.ChannelID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/cc/%s#%s", baseShareURL, data, signature), nil
}

func (m *Messenger) parseCommunityChannelURLWithData(data string, signature string) (*URLDataResponse, error) {
	_, err := m.verifySignature(data, signature)
	if err != nil {
		return nil, err
	}

	channelData, err := urls.DecodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var channelProto protobuf.Channel
	err = proto.Unmarshal(channelData, &channelProto)
	if err != nil {
		return nil, err
	}

	return &URLDataResponse{
		Community: CommunityURLData{
			DisplayName:  channelProto.Community.DisplayName,
			Description:  channelProto.Community.Description,
			MembersCount: channelProto.Community.MembersCount,
			Color:        channelProto.Community.Color,
		},
		Channel: CommunityChannelURLData{
			Emoji:       channelProto.Emoji,
			DisplayName: channelProto.DisplayName,
			Description: channelProto.Description,
			Color:       channelProto.Color,
		},
	}, nil
}

func (m *Messenger) ShareUserURLWithChatKey(contactID string) (string, error) {
	publicKey, err := common.HexToPubkey(contactID)
	if err != nil {
		return "", err
	}

	shortKey, err := m.SerializePublicKey(crypto.CompressPubkey(publicKey))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/u#%s", baseShareURL, shortKey), nil
}

func (m *Messenger) prepareContactData(contact *Contact) ContactURLData {
	return ContactURLData{
		DisplayName: contact.DisplayName,
	}
}

func (m *Messenger) parseUserURLWithChatKey(urlData string) (*URLDataResponse, error) {
	pubKeyBytes, err := m.DeserializePublicKey(urlData)
	if err != nil {
		return nil, err
	}

	pubKey, err := crypto.DecompressPubkey(pubKeyBytes)
	if err != nil {
		return nil, err
	}

	contactID := common.PubkeyToHex(pubKey)

	contact, ok := m.allContacts.Load(contactID)
	if !ok {
		return nil, ErrContactNotFound
	}

	return &URLDataResponse{
		Contact: m.prepareContactData(contact),
	}, nil
}

func (m *Messenger) ShareUserURLWithENS(contactID string) (string, error) {
	contact, ok := m.allContacts.Load(contactID)
	if !ok {
		return "", ErrContactNotFound
	}
	return fmt.Sprintf("%s/u#%s", baseShareURL, contact.EnsName), nil
}

func (m *Messenger) parseUserURLWithENS(ensName string) (*URLDataResponse, error) {
	// TODO: fetch contact by ens name
	return nil, fmt.Errorf("not implemented yet")
}

func (m *Messenger) prepareEncodedUserData(contact *Contact) (string, string, error) {
	userProto := &protobuf.User{
		DisplayName: contact.DisplayName,
	}

	userData, err := proto.Marshal(userProto)
	if err != nil {
		return "", "", err
	}

	return m.prepareEncodedRawData(userData, m.identity)
}

func (m *Messenger) ShareUserURLWithData(contactID string) (string, error) {
	contact, ok := m.allContacts.Load(contactID)
	if !ok {
		return "", ErrContactNotFound
	}

	data, signature, err := m.prepareEncodedUserData(contact)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/u/%s#%s", baseShareURL, data, signature), nil
}

func (m *Messenger) parseUserURLWithData(data string, signature string) (*URLDataResponse, error) {
	_, err := m.verifySignature(data, signature)
	if err != nil {
		return nil, err
	}

	userData, err := urls.DecodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var userProto protobuf.User
	err = proto.Unmarshal(userData, &userProto)
	if err != nil {
		return nil, err
	}

	return &URLDataResponse{
		Contact: ContactURLData{
			DisplayName: userProto.DisplayName,
			Description: userProto.Description,
		},
	}, nil
}

func (m *Messenger) ParseSharedURL(url string) (*URLDataResponse, error) {
	if !strings.HasPrefix(url, baseShareURL) {
		return nil, fmt.Errorf("url should start with '%s'", baseShareURL)
	}

	urlContents := regexp.MustCompile(`\#`).Split(strings.TrimPrefix(url, baseShareURL+"/"), 2)
	if len(urlContents) != 2 {
		return nil, fmt.Errorf("url should contain at least one `#` separator")
	}

	if urlContents[0] == "c" {
		return m.parseCommunityURLWithChatKey(urlContents[1])
	}

	if strings.HasPrefix(urlContents[0], "c/") {
		return m.parseCommunityURLWithData(strings.TrimPrefix(urlContents[0], "c/"), urlContents[1])
	}

	if strings.HasPrefix(urlContents[0], "cc/") {
		first := strings.TrimPrefix(urlContents[0], "cc/")

		isChannel, err := regexp.MatchString(channelUUIDRegExp, first)
		if err != nil {
			return nil, err
		}
		if isChannel {
			return m.parseCommunityChannelURLWithChatKey(first, urlContents[1])
		}
		return m.parseCommunityChannelURLWithData(first, urlContents[1])
	}

	if urlContents[0] == "u" {
		if strings.HasPrefix(urlContents[1], "zQ3sh") {
			return m.parseUserURLWithChatKey(urlContents[1])
		}
		return m.parseUserURLWithENS(urlContents[1])
	}

	if strings.HasPrefix(urlContents[0], "u/") {
		return m.parseUserURLWithData(strings.TrimPrefix(urlContents[0], "u/"), urlContents[1])
	}

	return nil, fmt.Errorf("unhandled shared url: %s", url)
}
