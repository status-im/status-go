package communities

import (
	"bytes"
	"testing"

	"github.com/status-im/status-go/protocol/requests"

	"github.com/golang/protobuf/proto"
	_ "github.com/mutecomm/go-sqlcipher" // require go-sqlcipher that overrides default implementation

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/sqlite"
)

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerSuite))
}

type ManagerSuite struct {
	suite.Suite
	manager *Manager
}

func (s *ManagerSuite) SetupTest() {
	db, err := sqlite.OpenInMemory()
	s.Require().NoError(err)
	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.Require().NoError(err)
	m, err := NewManager(&key.PublicKey, db, nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(m.Start())
	s.manager = m
}

func (s *ManagerSuite) TestCreateCommunity() {

	request := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	community, err := s.manager.CreateCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	communities, err := s.manager.All()
	s.Require().NoError(err)
	// Consider status default community
	s.Require().Len(communities, 2)

	actualCommunity := communities[0]
	if bytes.Equal(community.ID(), communities[1].ID()) {
		actualCommunity = communities[1]
	}

	s.Require().Equal(community.ID(), actualCommunity.ID())
	s.Require().Equal(community.PrivateKey(), actualCommunity.PrivateKey())
	s.Require().True(proto.Equal(community.config.CommunityDescription, actualCommunity.config.CommunityDescription))
}

func (s *ManagerSuite) TestEditCommunity() {
	//create community
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	community, err := s.manager.CreateCommunity(createRequest)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	update := &requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Name:        "statusEdited",
			Description: "status community description edited",
		},
	}

	updatedCommunity, err := s.manager.EditCommunity(update)
	s.Require().NoError(err)
	s.Require().NotNil(updatedCommunity)

	//ensure updated community successfully stored
	communities, err := s.manager.All()
	s.Require().NoError(err)
	// Consider status default community
	s.Require().Len(communities, 2)

	storedCommunity := communities[0]
	if bytes.Equal(community.ID(), communities[1].ID()) {
		storedCommunity = communities[1]
	}

	s.Require().Equal(storedCommunity.ID(), updatedCommunity.ID())
	s.Require().Equal(storedCommunity.PrivateKey(), updatedCommunity.PrivateKey())
	s.Require().Equal(storedCommunity.config.CommunityDescription.Identity.DisplayName, update.CreateCommunity.Name)
	s.Require().Equal(storedCommunity.config.CommunityDescription.Identity.Description, update.CreateCommunity.Description)
}

func (s *ManagerSuite) TestGetAdminCommuniesChatIDs() {

	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	community, err := s.manager.CreateCommunity(createRequest)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	chat := &protobuf.CommunityChat{
		Identity: &protobuf.ChatIdentity{
			DisplayName: "added-chat",
			Description: "description",
		},
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Members: make(map[string]*protobuf.CommunityMember),
	}

	_, _, err = s.manager.CreateChat(community.ID(), chat)
	s.Require().NoError(err)

	adminChatIDs, err := s.manager.GetAdminCommunitiesChatIDs()
	s.Require().NoError(err)
	s.Require().Len(adminChatIDs, 1)
}
