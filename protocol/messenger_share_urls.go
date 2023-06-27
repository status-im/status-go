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

type UrlDataResponse struct {
	Community CommunityUrlData `json:"community"`
}

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
		Community: CommunityUrlData{
			CommunityID:  community.ID(),
			DisplayName:  community.Identity().DisplayName,
			Description:  community.DescriptionText(),
			MembersCount: uint32(community.MembersCount()),
			Color:        community.Identity().GetColor(),
		},
	}, nil
}

func (m *Messenger) prepareEncodedCommunityData(community *communities.Community) (string, error) {
	// TODO: no ID here!
	communityProto := &protobuf.Community{
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
	}

	communityData, err := json.Marshal(communityProto)
	if err != nil {
		return "", err
	}

	encodedData, err := urls.EncodeDataURL(communityData)
	if err != nil {
		return "", err
	}

	return encodedData, nil
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

	// return fmt.Sprintf("%s/c/%s#%s", baseShareUrl, communityBase64, string(signature)), nil
	pubKey, err := m.SerializePublicKey(communityID)
	if err != nil {
		return "", err
	}

	data, err := m.prepareEncodedCommunityData(community)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/c/%s#%s", baseShareUrl, pubKey, data), nil
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

	return fmt.Sprintf("%s/cc/%s/%s", baseShareUrl, communityKey, request.ChannelID), nil
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

func (m *Messenger) parseCommuntyChannelSharedURL(url string) (*UrlDataResponse, error) {
	return nil, errors.New("not implemented yet")
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
		// TODO: encrypted protobuf can contain '#' and it could not be parsed
		return m.parseCommunityURLWithChatKey(urlContents[1])
	}

	if len(urlContents) == 2 && strings.HasPrefix(urlContents[0], "c/") {
		return m.parseCommunityURLWithData(strings.TrimPrefix(urlContents[0], "c/"), urlContents[1])
	}

	return nil, errors.New("unhandled url group")
}
