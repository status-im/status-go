package sharedurls

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/crypto"
	types2 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/multiformat"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/contacts"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/services/utils"
)

const baseShareURL = "https://status.app"
const userPath = "u#"
const userWithDataPath = "u/"
const communityPath = "c#"
const communityWithDataPath = "c/"
const channelPath = "cc/"

const sharedURLUserPrefix = baseShareURL + "/" + userPath
const sharedURLUserPrefixWithData = baseShareURL + "/" + userWithDataPath
const sharedURLCommunityPrefix = baseShareURL + "/" + communityPath
const sharedURLCommunityPrefixWithData = baseShareURL + "/" + communityWithDataPath
const sharedURLChannelPrefixWithData = baseShareURL + "/" + channelPath

const channelUUIDRegExp = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-5][0-9a-f]{3}-[089ab][0-9a-f]{3}-[0-9a-f]{12}$"

var (
	channelRegExp         = regexp.MustCompile(channelUUIDRegExp)
	errInvalidCommunityID = errors.New("invalid community id")
	errNoDataProvider     = errors.New("no data provider")
	errCommunityNil       = errors.New("community is nil")
)

var (
	ErrContactNotFound = errors.New("contact not found")
)

func isNil(data interface{}) bool {
	v := reflect.ValueOf(data).Kind()
	return data == nil || v == reflect.Ptr && reflect.ValueOf(data).IsNil()
}

type Service struct {
	provider DataProvider
}

func NewService(provider DataProvider) *Service {
	s := &Service{}
	s.SetDataProvider(provider)
	return s
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "sharedurls",
			Version:   "0.1.0",
			Service: &PublicAPI{
				service: s,
			},
		},
	}
}

func (s *Service) SetDataProvider(provider DataProvider) {
	if isNil(provider) {
		provider = &NopDataProvider{}
	}
	s.provider = provider
}

func decodeCommunityID(serialisedPublicKey string) (string, error) {
	deserializedCommunityID, err := multiformat.DeserializeCompressedKey(serialisedPublicKey)
	if err != nil {
		return "", err
	}

	communityID, err := crypto.HexToPubkey(deserializedCommunityID)
	if err != nil {
		return "", err
	}

	return types2.EncodeHex(crypto.CompressPubkey(communityID)), nil
}

func serializePublicKey(compressedKey types2.HexBytes) (string, error) {
	return utils.SerializePublicKey(compressedKey)
}

func deserializePublicKey(compressedKey string) (types2.HexBytes, error) {
	return utils.DeserializePublicKey(compressedKey)
}

func (s *Service) ShareCommunityURLWithChatKey(communityID types2.HexBytes) (string, error) {
	shortKey, err := serializePublicKey(communityID)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/c#%s", baseShareURL, shortKey), nil
}

func parseCommunityURLWithChatKey(urlData string) (*URLDataResponse, error) {
	communityID, err := decodeCommunityID(urlData)
	if err != nil {
		return nil, err
	}

	return &URLDataResponse{
		Community: &CommunityURLData{
			CommunityID: communityID,
			TagIndices:  []uint32{},
		},
	}, nil
}

func prepareEncodedCommunityData(community *communities.Community) (string, string, error) {
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

	urlDataProto := &protobuf.URLData{
		Content: communityData,
	}

	urlData, err := proto.Marshal(urlDataProto)
	if err != nil {
		return "", "", err
	}

	shortKey, err := serializePublicKey(community.ID())
	if err != nil {
		return "", "", err
	}

	encodedData, err := encodeDataURL(urlData)
	if err != nil {
		return "", "", err
	}

	return encodedData, shortKey, nil
}

func (s *Service) ShareCommunityURLWithData(communityID types2.HexBytes) (string, error) {
	community, err := s.provider.GetCommunityByID(communityID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get community")
	}

	return ShareCommunityURLWithData(community)
}

func ShareCommunityURLWithData(community *communities.Community) (string, error) {
	if community == nil {
		return "", errCommunityNil
	}

	data, shortKey, err := prepareEncodedCommunityData(community)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/c/%s#%s", baseShareURL, data, shortKey), nil
}

func parseCommunityURLWithData(data string, chatKey string) (*URLDataResponse, error) {
	communityID, err := deserializePublicKey(chatKey)
	if err != nil {
		return nil, err
	}

	urlData, err := decodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var urlDataProto protobuf.URLData
	err = proto.Unmarshal(urlData, &urlDataProto)
	if err != nil {
		return nil, err
	}

	var communityProto protobuf.Community
	err = proto.Unmarshal(urlDataProto.Content, &communityProto)
	if err != nil {
		return nil, err
	}

	var tagIndices []uint32
	if communityProto.TagIndices != nil {
		tagIndices = communityProto.TagIndices
	} else {
		tagIndices = []uint32{}
	}

	return &URLDataResponse{
		Community: &CommunityURLData{
			DisplayName:  communityProto.DisplayName,
			Description:  communityProto.Description,
			MembersCount: communityProto.MembersCount,
			Color:        communityProto.Color,
			TagIndices:   tagIndices,
			CommunityID:  types2.EncodeHex(communityID),
		},
	}, nil
}

func (s *Service) ShareCommunityChannelURLWithChatKey(communityID types2.HexBytes, channelID string) (string, error) {
	if len(communityID) == 0 {
		return "", errInvalidCommunityID
	}

	shortKey, err := serializePublicKey(communityID)
	if err != nil {
		return "", err
	}

	valid, err := regexp.MatchString(channelUUIDRegExp, channelID)
	if err != nil {
		return "", err
	}

	if !valid {
		return "", fmt.Errorf("channelID should be UUID, got %s", gocommon.TruncateWithDot(channelID))
	}

	return fmt.Sprintf("%s/cc/%s#%s", baseShareURL, channelID, shortKey), nil
}

func parseCommunityChannelURLWithChatKey(channelID string, publicKey string) (*URLDataResponse, error) {
	valid, err := regexp.MatchString(channelUUIDRegExp, channelID)
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, fmt.Errorf("channelID should be UUID, got %s", gocommon.TruncateWithDot(channelID))
	}

	communityID, err := decodeCommunityID(publicKey)
	if err != nil {
		return nil, err
	}

	return &URLDataResponse{
		Community: &CommunityURLData{
			CommunityID: communityID,
			TagIndices:  []uint32{},
		},
		Channel: &CommunityChannelURLData{
			ChannelUUID: channelID,
		},
	}, nil
}

func (s *Service) prepareEncodedCommunityChannelData(community *communities.Community, channel *protobuf.CommunityChat, channelID string) (string, string, error) {
	communityProto := &protobuf.Community{
		DisplayName:  community.Identity().DisplayName,
		Description:  community.DescriptionText(),
		MembersCount: uint32(community.MembersCount()),
		Color:        community.Identity().GetColor(),
		TagIndices:   community.TagsIndices(),
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

	urlDataProto := &protobuf.URLData{
		Content: channelData,
	}

	urlData, err := proto.Marshal(urlDataProto)
	if err != nil {
		return "", "", err
	}

	shortKey, err := serializePublicKey(community.ID())
	if err != nil {
		return "", "", err
	}
	encodedData, err := encodeDataURL(urlData)
	if err != nil {
		return "", "", err
	}

	return encodedData, shortKey, nil
}

func (s *Service) ShareCommunityChannelURLWithData(communityID types2.HexBytes, channelID string) (string, error) {
	if len(communityID) == 0 {
		return "", errInvalidCommunityID
	}

	valid, err := regexp.MatchString(channelUUIDRegExp, channelID)
	if err != nil {
		return "", err
	}

	if !valid {
		return "", fmt.Errorf("channelID should be UUID, got %s", gocommon.TruncateWithDot(channelID))
	}

	community, err := s.provider.GetCommunityByID(communityID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get community")
	}

	channel := community.Chats()[channelID]
	if channel == nil {
		return "", fmt.Errorf("channel with channelID %s not found", gocommon.TruncateWithDot(channelID))
	}

	data, shortKey, err := s.prepareEncodedCommunityChannelData(community, channel, channelID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/cc/%s#%s", baseShareURL, data, shortKey), nil
}

func parseCommunityChannelURLWithData(data string, chatKey string) (*URLDataResponse, error) {
	communityID, err := deserializePublicKey(chatKey)
	if err != nil {
		return nil, err
	}

	urlData, err := decodeDataURL(data)
	if err != nil {
		return nil, err
	}

	var urlDataProto protobuf.URLData
	err = proto.Unmarshal(urlData, &urlDataProto)
	if err != nil {
		return nil, err
	}

	var channelProto protobuf.Channel
	err = proto.Unmarshal(urlDataProto.Content, &channelProto)
	if err != nil {
		return nil, err
	}

	var tagIndices []uint32
	if channelProto.Community.TagIndices != nil {
		tagIndices = channelProto.Community.TagIndices
	} else {
		tagIndices = []uint32{}
	}

	return &URLDataResponse{
		Community: &CommunityURLData{
			DisplayName:  channelProto.Community.DisplayName,
			Description:  channelProto.Community.Description,
			MembersCount: channelProto.Community.MembersCount,
			Color:        channelProto.Community.Color,
			TagIndices:   tagIndices,
			CommunityID:  types2.EncodeHex(communityID),
		},
		Channel: &CommunityChannelURLData{
			Emoji:       channelProto.Emoji,
			DisplayName: channelProto.DisplayName,
			Description: channelProto.Description,
			Color:       channelProto.Color,
			ChannelUUID: channelProto.Uuid,
		},
	}, nil
}

func (s *Service) ShareUserURLWithChatKey(contactID string) (string, error) {
	publicKey, err := crypto.HexToPubkey(contactID)
	if err != nil {
		return "", err
	}

	shortKey, err := serializePublicKey(crypto.CompressPubkey(publicKey))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/u#%s", baseShareURL, shortKey), nil
}

func parseUserURLWithChatKey(urlData string) (*URLDataResponse, error) {
	pubKeyBytes, err := deserializePublicKey(urlData)
	if err != nil {
		return nil, err
	}

	pubKey, err := crypto.DecompressPubkey(pubKeyBytes)
	if err != nil {
		return nil, err
	}

	serializedPublicKey, err := multiformat.SerializeLegacyKey(crypto.PubkeyToHex(pubKey))
	if err != nil {
		return nil, err
	}

	return &URLDataResponse{
		Contact: &ContactURLData{
			PublicKey: serializedPublicKey,
		},
	}, nil
}

func (s *Service) ShareUserURLWithENS(contactID string) (string, error) {
	contact, err := s.provider.GetContactByID(contactID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get contact")
	}
	if contact == nil {
		return "", ErrContactNotFound
	}
	return fmt.Sprintf("%s/u#%s", baseShareURL, contact.EnsName), nil
}

func parseUserURLWithENS(ensName string) (*URLDataResponse, error) {
	// TODO: fetch contact by ens name
	return nil, fmt.Errorf("not implemented yet")
}

func prepareEncodedUserData(contact *contacts.Contact) (string, string, error) {
	pk, err := contact.PublicKey()
	if err != nil {
		return "", "", err
	}

	shortKey, err := serializePublicKey(crypto.CompressPubkey(pk))
	if err != nil {
		return "", "", err
	}

	if contact.DisplayName == "" && contact.Bio == "" {
		return "", shortKey, nil
	}

	userProto := &protobuf.User{
		DisplayName: contact.DisplayName,
		Description: contact.Bio,
	}

	userData, err := proto.Marshal(userProto)
	if err != nil {
		return "", "", err
	}

	urlDataProto := &protobuf.URLData{
		Content: userData,
	}

	urlData, err := proto.Marshal(urlDataProto)
	if err != nil {
		return "", "", err
	}

	encodedData, err := encodeDataURL(urlData)
	if err != nil {
		return "", "", err
	}

	return encodedData, shortKey, nil
}

func (s *Service) ShareUserURLWithData(contactID string) (string, error) {
	contact, err := s.provider.GetContactByID(contactID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get contact")
	}
	if contact == nil {
		return "", ErrContactNotFound
	}
	return ShareUserURLWithData(contact)
}

func ShareUserURLWithData(contact *contacts.Contact) (string, error) {
	data, shortKey, err := prepareEncodedUserData(contact)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/u/%s#%s", baseShareURL, data, shortKey), nil
}
