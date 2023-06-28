package protocol

import (
	"encoding/json"
	"errors"
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

type UrlDataResponse struct {
	Community CommunityUrlData        `json:"community"`
	Channel   CommunityChannelUrlData `json:"channel"`
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

	pubKey, err := m.SerializePublicKey(communityID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/c#%s", baseShareUrl, pubKey), nil
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
	// TODO: no ID here!
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

func (m *Messenger) parseCommunityURLWithData(keyString string, dataString string) (*UrlDataResponse, error) {
	communityID, err := m.DeserializePublicKey(keyString)
	if err != nil {
		return nil, err
	}

	communityData, err := urls.DecodeDataURL(dataString)
	if err != nil {
		return nil, err
	}

	var communityProto protobuf.Community
	// TODO: does not restored properly, maybe decoding is wrong?
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

	communityKey, err := m.SerializePublicKey(request.CommunityID)
	if err != nil {
		return "", err
	}

	// TODO: convert ChannelID to 003cdcd5-e065-48f9-b166-b1a94ac75a11 format
	return fmt.Sprintf("%s/cc/%s#%s", baseShareUrl, request.ChannelID, communityKey), nil
}

func (m *Messenger) prepareCommunityChannelData(channel *protobuf.CommunityChat) CommunityChannelUrlData {
	return CommunityChannelUrlData{
		Emoji:       channel.Identity.Emoji,
		DisplayName: channel.Identity.DisplayName,
		Description: channel.Identity.Description,
		Color:       channel.Identity.Color,
	}
}

func (m *Messenger) parseCommunityChannelWithChatKey(channelId string, publickKey string) (*UrlDataResponse, error) {
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

func (m *Messenger) parseCommunityChannelWithData(data string, signature string) (*UrlDataResponse, error) {
	// TODO: get PubKey from signature

	channelData, err := urls.DecodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var channelProto protobuf.Channel
	// TODO: does not restored properly, maybe decoding is wrong?
	err = proto.Unmarshal(channelData, &channelProto)
	if err != nil {
		return nil, err
	}

	return &UrlDataResponse{
		Community: CommunityUrlData{
			//			CommunityID:  communityID,
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

func (m *Messenger) parseProfileSharedURL(urlData string) (*UrlDataResponse, error) {
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

	// TODDO: impl
	return nil, nil
}

func (m *Messenger) ParseSharedURL(url string) (*UrlDataResponse, error) {
	if !strings.HasPrefix(url, baseShareUrl) {
		return nil, fmt.Errorf("url should start with '%s'", baseShareUrl)
	}
	urlContents := strings.Split(strings.TrimPrefix(url, baseShareUrl+"/"), "#")

	fmt.Println("-----> ParseSharedURL::: url: ", url, "urlContents: ", len(urlContents))

	if len(urlContents) == 2 && urlContents[0] == "c" {
		// TODO: encrypted protobuf can contain '#'!
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
			return m.parseCommunityChannelWithChatKey(first, urlContents[1])
		} else {
			return m.parseCommunityChannelWithData(first, urlContents[1])
		}
	}

	return nil, fmt.Errorf("unhandled shared url: %s", url)
}
