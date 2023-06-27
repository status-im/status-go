package protocol

import (
	"crypto/ecdsa"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestMessengerShareUrlsSuite(t *testing.T) {
	suite.Run(t, new(MessengerShareUrlsSuite))
}

type MessengerShareUrlsSuite struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerShareUrlsSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessenger()
	s.privateKey = s.m.identity
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerShareUrlsSuite) TearDownTest() {
	s.Require().NoError(s.m.Shutdown())
}

func (s *MessengerShareUrlsSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newMessengerWithKey(s.shh, privateKey, s.logger, nil)
	s.Require().NoError(err)
	return messenger
}

func (s *MessengerShareUrlsSuite) createCommunity() *communities.Community {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := s.m.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	return response.Communities()[0]
}

func (s *MessengerShareUrlsSuite) TestSerializePublicKey() {
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)

	serializedKey, err := s.m.SerializePublicKey(crypto.CompressPubkey(&key.PublicKey))

	s.Require().NoError(err)
	s.Require().Len(serializedKey, 49)
	s.Require().True(strings.HasPrefix(serializedKey, "zQ3sh"))
}

func (s *MessengerShareUrlsSuite) TestDeserializePublicKey() {
	serializedKey := "zQ3shPyZJnxZK4Bwyx9QsaksNKDYTPmpwPvGSjMYVHoXHeEgB"

	publicKey, err := s.m.DeserializePublicKey(serializedKey)

	s.Require().NoError(err)
	s.Require().Len(publicKey, 33)
	s.Require().True(strings.HasPrefix(publicKey.String(), "0x"))
}

func (s *MessengerShareUrlsSuite) TestCreateCommunityURLWithChatKey() {
	community := s.createCommunity()

	url, err := s.m.ShareCommunityURLWithChatKey(community.ID())
	s.Require().NoError(err)

	publicKey, err := s.m.SerializePublicKey(community.ID())
	s.Require().NoError(err)

	expectedUrl := fmt.Sprintf("%s/%s%s", baseShareUrl, "c#", publicKey)
	s.Require().Equal(expectedUrl, url)
}

func (s *MessengerShareUrlsSuite) TestParseCommunityURLWithChatKey() {
	community := s.createCommunity()

	publicKey, err := s.m.SerializePublicKey(community.ID())
	s.Require().NoError(err)

	url := fmt.Sprintf("%s/%s%s", baseShareUrl, "c#", publicKey)

	urlData, err := s.m.ParseSharedURL(url)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Community)
	s.Require().Equal(community.ID(), urlData.Community.CommunityID)
	s.Require().Equal(community.Identity().DisplayName, urlData.Community.DisplayName)
	s.Require().Equal(community.DescriptionText(), urlData.Community.Description)
	s.Require().Equal(uint32(community.MembersCount()), urlData.Community.MembersCount)
	s.Require().Equal(community.Identity().GetColor(), urlData.Community.Color)
}

func (s *MessengerShareUrlsSuite) TestCreateCommunityURLWithData() {
	community := s.createCommunity()

	url, err := s.m.CreateCommunityURLWithData(community.ID())
	s.Require().NoError(err)

	communityID, err := s.m.SerializePublicKey(community.ID())
	s.Require().NoError(err)

	communityData, err := s.m.prepareEncodedCommunityData(community)
	s.Require().NoError(err)

	expectedUrl := fmt.Sprintf("%s/c/%s#%s", baseShareUrl, communityID, communityData)
	s.Require().Equal(expectedUrl, url)
}

func (s *MessengerShareUrlsSuite) TestParseCommunityURLWithData() {
	community := s.createCommunity()

	communityID, err := s.m.SerializePublicKey(community.ID())
	s.Require().NoError(err)

	communityData, err := s.m.prepareEncodedCommunityData(community)
	s.Require().NoError(err)

	url := fmt.Sprintf("%s/c/%s#%s", baseShareUrl, communityID, communityData)

	urlData, err := s.m.ParseSharedURL(url)
	s.Require().NoError(err)
	s.Require().NotNil(urlData)

	s.Require().NotNil(urlData.Community)
	// TODO: s.Require().Equal(community.ID(), urlData.Community.CommunityID)
	s.Require().Equal(community.Identity().DisplayName, urlData.Community.DisplayName)
	s.Require().Equal(community.DescriptionText(), urlData.Community.Description)
	s.Require().Equal(uint32(community.MembersCount()), urlData.Community.MembersCount)
	s.Require().Equal(community.Identity().GetColor(), urlData.Community.Color)
}
