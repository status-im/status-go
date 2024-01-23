package communities

import (
	"crypto/ecdsa"
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
)

func TestCommunityEncryptionDescriptionSuite(t *testing.T) {
	suite.Run(t, new(CommunityEncryptionDescriptionSuite))
}

type CommunityEncryptionDescriptionSuite struct {
	suite.Suite

	descriptionEncryptor *DescriptionEncryptorMock
	identity             *ecdsa.PrivateKey
	communityID          []byte
	logger               *zap.Logger
}

func (s *CommunityEncryptionDescriptionSuite) SetupTest() {
	s.descriptionEncryptor = &DescriptionEncryptorMock{
		descriptions:          map[string]*protobuf.CommunityDescription{},
		channelIDToKeyIDSeqNo: map[string]string{},
	}

	identity, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.identity = identity
	s.communityID = crypto.CompressPubkey(&identity.PublicKey)

	s.logger, err = zap.NewDevelopment()
	s.Require().NoError(err)
}

type DescriptionEncryptorMock struct {
	descriptions          map[string]*protobuf.CommunityDescription
	channelIDToKeyIDSeqNo map[string]string
}

func (dem *DescriptionEncryptorMock) encryptCommunityDescription(community *Community, d *protobuf.CommunityDescription) (string, []byte, error) {
	keyIDSeqNo := uuid.New().String()
	dem.descriptions[keyIDSeqNo] = d
	return keyIDSeqNo, []byte("encryptedDescription"), nil
}

func (dem *DescriptionEncryptorMock) encryptCommunityDescriptionChannel(community *Community, channelID string, d *protobuf.CommunityDescription) (string, []byte, error) {
	keyIDSeqNo := uuid.New().String()
	dem.descriptions[keyIDSeqNo] = d
	dem.channelIDToKeyIDSeqNo[channelID] = keyIDSeqNo
	return keyIDSeqNo, []byte("encryptedDescription"), nil
}

func (dem *DescriptionEncryptorMock) decryptCommunityDescription(keyIDSeqNo string, d []byte) (*DecryptCommunityResponse, error) {
	description := dem.descriptions[keyIDSeqNo]
	if description == nil {
		return nil, errors.New("no key to decrypt private data")
	}
	return &DecryptCommunityResponse{Description: description}, nil
}

func (dem *DescriptionEncryptorMock) forgetAllKeys() {
	dem.descriptions = make(map[string]*protobuf.CommunityDescription)
}

func (dem *DescriptionEncryptorMock) forgetChannelKeys() {
	for _, keyIDSeqNo := range dem.channelIDToKeyIDSeqNo {
		delete(dem.descriptions, keyIDSeqNo)
	}
}

func (s *CommunityEncryptionDescriptionSuite) description() *protobuf.CommunityDescription {
	return &protobuf.CommunityDescription{
		IntroMessage: "one of not encrypted fields",
		Members: map[string]*protobuf.CommunityMember{
			"memberA": &protobuf.CommunityMember{},
			"memberB": &protobuf.CommunityMember{},
		},
		Chats: map[string]*protobuf.CommunityChat{
			"channelA": &protobuf.CommunityChat{
				Members: map[string]*protobuf.CommunityMember{
					"memberA": &protobuf.CommunityMember{},
					"memberB": &protobuf.CommunityMember{},
				},
			},
			"channelB": &protobuf.CommunityChat{
				Members: map[string]*protobuf.CommunityMember{
					"memberA": &protobuf.CommunityMember{},
				},
			},
		},
		PrivateData: map[string][]byte{},

		// ensure community and channel encryption
		TokenPermissions: map[string]*protobuf.CommunityTokenPermission{
			"community-level-permission": &protobuf.CommunityTokenPermission{
				Id:            "community-level-permission",
				Type:          protobuf.CommunityTokenPermission_BECOME_MEMBER,
				TokenCriteria: []*protobuf.TokenCriteria{},
				ChatIds:       []string{},
			},
			"channel-level-permission": &protobuf.CommunityTokenPermission{
				Id:            "channel-level-permission",
				Type:          protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
				TokenCriteria: []*protobuf.TokenCriteria{},
				ChatIds:       []string{types.EncodeHex(crypto.CompressPubkey(&s.identity.PublicKey)) + "channelB"},
			},
		},
	}
}

func (s *CommunityEncryptionDescriptionSuite) TestEncryptionDecryption() {
	description := s.description()

	err := encryptDescription(s.descriptionEncryptor, &Community{
		config: &Config{ID: &s.identity.PublicKey, CommunityDescription: description},
	}, description)
	s.Require().NoError(err)
	s.Require().Len(description.PrivateData, 2)

	// members and chats should become empty (encrypted)
	s.Require().Empty(description.Members)
	s.Require().Empty(description.Chats)
	s.Require().Equal(description.IntroMessage, "one of not encrypted fields")

	// members and chats should be brought back
	_, err = decryptDescription([]byte("some-id"), s.descriptionEncryptor, description, s.logger)
	s.Require().NoError(err)
	s.Require().Len(description.Members, 2)
	s.Require().Len(description.Chats, 2)
	s.Require().Len(description.Chats["channelA"].Members, 2)
	s.Require().Len(description.Chats["channelB"].Members, 1)
	s.Require().Equal(description.IntroMessage, "one of not encrypted fields")
}

func (s *CommunityEncryptionDescriptionSuite) TestDecryption_NoKeys() {
	encryptedDescription := func() *protobuf.CommunityDescription {
		description := s.description()

		err := encryptDescription(s.descriptionEncryptor, &Community{
			config: &Config{ID: &s.identity.PublicKey, CommunityDescription: description},
		}, description)
		s.Require().NoError(err)

		return description
	}()

	description := proto.Clone(encryptedDescription).(*protobuf.CommunityDescription)
	// forget channel keys, so channel members can't be decrypted
	s.descriptionEncryptor.forgetChannelKeys()

	// encrypted channel should have no members
	_, err := decryptDescription([]byte("some-id"), s.descriptionEncryptor, description, s.logger)
	s.Require().NoError(err)
	s.Require().Len(description.Members, 2)
	s.Require().Len(description.Chats, 2)
	s.Require().Len(description.Chats["channelA"].Members, 2)
	s.Require().Len(description.Chats["channelB"].Members, 0) // encrypted channel
	s.Require().Equal(description.IntroMessage, "one of not encrypted fields")

	description = proto.Clone(encryptedDescription).(*protobuf.CommunityDescription)
	// forget the keys, so chats and members can't be decrypted
	s.descriptionEncryptor.forgetAllKeys()

	// members and chats should be empty
	_, err = decryptDescription([]byte("some-id"), s.descriptionEncryptor, description, s.logger)
	s.Require().NoError(err)
	s.Require().Empty(description.Members)
	s.Require().Empty(description.Chats)
	s.Require().Equal(description.IntroMessage, "one of not encrypted fields")
}
