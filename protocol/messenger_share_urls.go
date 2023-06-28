package protocol

import (
	"encoding/json"
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

type CommunityUrlData struct {
	CommunityID  types.HexBytes `json:"communityId"`
	DisplayName  string         `json:"displayName"`
	Description  string         `json:"description"`
	MembersCount uint32         `json:"membersCount"`
	Color        string         `json:"color"`
}

type CommunityChannelUrlData struct {
	Emoji       string `json:"emoji"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

type ContactUrlData struct {
	ContactID   string `json:"contactId"`
	DisplayName string `json:"displayName"`
}

type UrlDataResponse struct {
	Community CommunityUrlData        `json:"community"`
	Channel   CommunityChannelUrlData `json:"channel"`
	Contact   ContactUrlData          `json:"contact"`
}

const baseShareUrl = "https://status.app"
const channelUuidRegExp = "/^[0-9a-f]{8}-[0-9a-f]{4}-[0-5][0-9a-f]{3}-[089ab][0-9a-f]{3}-[0-9a-f]{12}$/i"

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
	if len(communityID) == 0 {
		return "", ErrChatIDEmpty
	}

	shortKey, err := m.SerializePublicKey(communityID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/c#%s", baseShareUrl, shortKey), nil
}

func (m *Messenger) prepareCommunityData(community *communities.Community) CommunityUrlData {
	return CommunityUrlData{
		CommunityID:  community.ID(),
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
	}
}

func (m *Messenger) parseCommunityURLWithChatKey(urlData string) (*UrlDataResponse, error) {
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

	return &UrlDataResponse{
		Community: m.prepareCommunityData(community),
	}, nil
}

func (m *Messenger) prepareEncodedCommunityData(community *communities.Community) (string, string, error) {
	communityProto := &protobuf.Community{
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
	}

	communityData, err := json.Marshal(communityProto)
	if err != nil {
		return "", "", err
	}

	encodedData, err := urls.EncodeDataURL(communityData)
	if err != nil {
		return "", "", err
	}

	signature, err := crypto.SignBytes([]byte(encodedData), community.PrivateKey())
	if err != nil {
		return "", "", err
	}

	return encodedData, string(signature), nil
}

func (m *Messenger) ShareCommunityURLWithData(communityID types.HexBytes) (string, error) {
	community, err := m.GetCommunityByID(communityID)
	if err != nil {
		return "", err
	}

	if community == nil {
		return "", fmt.Errorf("community with communityID %s not found", communityID)
	}

	if community.Encrypted() {
		// TODO: not sure, is it right?
		return m.ShareCommunityURLWithChatKey(communityID)
	}

	data, signature, err := m.prepareEncodedCommunityData(community)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/c/%s#%s", baseShareUrl, data, signature), nil
}

func (m *Messenger) parseCommunityURLWithData(data string, signature string) (*UrlDataResponse, error) {
	pubKey, err := crypto.SigToPub([]byte(data), []byte(signature))
	if err != nil {
		return nil, err
	}

	communityID := crypto.CompressPubkey(pubKey)

	communityData, err := urls.DecodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var communityProto protobuf.Community
	err = proto.Unmarshal(communityData, &communityProto)
	if err != nil {
		return nil, err
	}

	return &UrlDataResponse{
		Community: CommunityUrlData{
			CommunityID:  communityID,
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

	// TODO: convert ChannelID to 003cdcd5-e065-48f9-b166-b1a94ac75a11 format
	return fmt.Sprintf("%s/cc/%s#%s", baseShareUrl, request.ChannelID, shortKey), nil
}

func (m *Messenger) prepareCommunityChannelData(channel *protobuf.CommunityChat) CommunityChannelUrlData {
	return CommunityChannelUrlData{
		Emoji:       channel.Identity.Emoji,
		DisplayName: channel.Identity.DisplayName,
		Description: channel.Identity.Description,
		Color:       channel.Identity.Color,
	}
}

func (m *Messenger) parseCommunityChannelURLWithChatKey(channelId string, publickKey string) (*UrlDataResponse, error) {
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

	channel, ok := community.Chats()[channelId]
	if !ok {
		return nil, fmt.Errorf("channel with channelId %s not found", channelId)
	}

	return &UrlDataResponse{
		Community: m.prepareCommunityData(community),
		Channel:   m.prepareCommunityChannelData(channel),
	}, nil
}

func (m *Messenger) prepareEncodedCommunityChannelData(community *communities.Community, channel *protobuf.CommunityChat, channelId string) (string, string, error) {

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
		Uuid:        channelId,
	}

	channelData, err := json.Marshal(channelProto)
	if err != nil {
		return "", "", err
	}

	encodedData, err := urls.EncodeDataURL(channelData)
	if err != nil {
		return "", "", err
	}

	signature, err := crypto.SignBytes([]byte(encodedData), community.PrivateKey())
	if err != nil {
		return "", "", err
	}

	return encodedData, string(signature), nil
}

func (m *Messenger) CreateCommunityChannelURLWithData(request *requests.CommunityChannelShareURL) (string, error) {
	if err := request.Validate(); err != nil {
		return "", err
	}

	community, err := m.GetCommunityByID(request.CommunityID)
	if err != nil {
		return "", err
	}

	channel := community.Chats()[request.ChannelID]
	if channel == nil {
		return "", fmt.Errorf("channel with channelId %s not found", request.ChannelID)
	}

	data, signature, err := m.prepareEncodedCommunityChannelData(community, channel, request.ChannelID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/cc/%s#%s", baseShareUrl, data, signature), nil
}

func (m *Messenger) parseCommunityChannelURLWithData(data string, signature string) (*UrlDataResponse, error) {
	pubKey, err := crypto.SigToPub([]byte(data), []byte(signature))
	if err != nil {
		return nil, err
	}

	communityID := crypto.CompressPubkey(pubKey)

	channelData, err := urls.DecodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var channelProto protobuf.Channel
	err = proto.Unmarshal(channelData, &channelProto)
	if err != nil {
		return nil, err
	}

	return &UrlDataResponse{
		Community: CommunityUrlData{
			CommunityID:  communityID,
			DisplayName:  channelProto.Community.DisplayName,
			Description:  channelProto.Community.Description,
			MembersCount: channelProto.Community.MembersCount,
			Color:        channelProto.Community.Color,
		},
		Channel: CommunityChannelUrlData{
			Emoji:       channelProto.Emoji,
			DisplayName: channelProto.DisplayName,
			Description: channelProto.Description,
			Color:       channelProto.Color,
		},
	}, nil
}

func (m *Messenger) ShareUserURLWithChatKey(contactId string) (string, error) {
	if len(contactId) == 0 {
		return "", ErrChatIDEmpty
	}

	publicKey, err := common.HexToPubkey(contactId)
	if err != nil {
		return "", err
	}

	shortKey, err := m.SerializePublicKey(crypto.CompressPubkey(publicKey))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/u#%s", baseShareUrl, shortKey), nil
}

func (m *Messenger) prepareContactData(contact *Contact) ContactUrlData {
	return ContactUrlData{
		ContactID:   contact.ID,
		DisplayName: contact.DisplayName,
	}
}

func (m *Messenger) parseUserURLWithChatKey(urlData string) (*UrlDataResponse, error) {
	contactId, err := m.DeserializePublicKey(urlData)
	if err != nil {
		return nil, err
	}

	contact, ok := m.allContacts.Load(contactId.String())
	if !ok {
		return nil, ErrContactNotFound
	}

	return &UrlDataResponse{
		Contact: m.prepareContactData(contact),
	}, nil
}

func (m *Messenger) ShareUserURLWithENS(contactId string) (string, error) {
	if len(contactId) == 0 {
		return "", ErrChatIDEmpty
	}

	contact, ok := m.allContacts.Load(contactId)
	if !ok {
		return "", ErrContactNotFound
	}
	return fmt.Sprintf("%s/u#%s", baseShareUrl, contact.EnsName), nil
}

func (m *Messenger) parseUserURLWithENS(ensName string) (*UrlDataResponse, error) {
	if len(ensName) == 0 {
		return nil, ErrChatIDEmpty
	}

	// TODO: fetch contact by ens name
	return nil, fmt.Errorf("not implemented yet")
}

func (m *Messenger) prepareEncodedUserData(contact *Contact) (string, string, error) {
	userProto := &protobuf.User{
		DisplayName: contact.DisplayName,
	}

	userData, err := json.Marshal(userProto)
	if err != nil {
		return "", "", err
	}

	encodedData, err := urls.EncodeDataURL(userData)
	if err != nil {
		return "", "", err
	}

	signature, err := crypto.SignBytes([]byte(encodedData), m.identity)
	if err != nil {
		return "", "", err
	}

	return encodedData, string(signature), nil
}

func (m *Messenger) ShareUserURLWithData(contactId string) (string, error) {
	contact, ok := m.allContacts.Load(contactId)
	if !ok {
		return "", ErrContactNotFound
	}

	data, signature, err := m.prepareEncodedUserData(contact)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/u/%s#%s", baseShareUrl, data, signature), nil
}

func (m *Messenger) parseUserURLWithData(data string, signature string) (*UrlDataResponse, error) {
	_, err := crypto.SigToPub([]byte(data), []byte(signature))
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

	return &UrlDataResponse{
		Contact: ContactUrlData{
			DisplayName: userProto.DisplayName,
		},
	}, nil
}

func (m *Messenger) ParseSharedURL(url string) (*UrlDataResponse, error) {
	if !strings.HasPrefix(url, baseShareUrl) {
		return nil, fmt.Errorf("url should start with '%s'", baseShareUrl)
	}
	urlContents := strings.Split(strings.TrimPrefix(url, baseShareUrl+"/"), "#")

	fmt.Println("-----> ParseSharedURL::: url: ", url, "urlContents: ", len(urlContents))

	if len(urlContents) == 2 && urlContents[0] == "c" {
		return m.parseCommunityURLWithChatKey(urlContents[1])
	}

	if len(urlContents) == 2 && strings.HasPrefix(urlContents[0], "c/") {
		return m.parseCommunityURLWithData(strings.TrimPrefix(urlContents[0], "c/"), urlContents[1])
	}

	if len(urlContents) == 2 && strings.HasPrefix(urlContents[0], "cc/") {
		first := strings.TrimPrefix(urlContents[0], "cc/")

		isChannel, err := regexp.MatchString(channelUuidRegExp, first)
		if err != nil {
			return nil, err
		}
		if isChannel {
			return m.parseCommunityChannelURLWithChatKey(first, urlContents[1])
		} else {
			return m.parseCommunityChannelURLWithData(first, urlContents[1])
		}
	}

	if len(urlContents) == 2 && urlContents[0] == "u" {
		if strings.HasPrefix(urlContents[1], "zQ3sh") {
			return m.parseUserURLWithChatKey(urlContents[1])
		} else {
			return m.parseUserURLWithENS(urlContents[1])
		}
	}

	if len(urlContents) == 2 && strings.HasPrefix(urlContents[0], "u/") {
		return m.parseUserURLWithData(strings.TrimPrefix(urlContents[0], "c/"), urlContents[1])
	}

	return nil, fmt.Errorf("unhandled shared url: %s", url)
}
