package communities

import (
	"bytes"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/types"
	userimages "github.com/status-im/status-go/images"
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
	dbPath, err := ioutil.TempFile("", "")
	s.NoError(err, "creating temp file for db")
	db, err := appdatabase.InitializeDB(dbPath.Name(), "")
	s.NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.NoError(err, "protocol migrate")

	key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	s.Require().NoError(err)
	m, err := NewManager(&key.PublicKey, db, nil, nil, nil, nil)
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

	community, err := s.manager.CreateCommunity(request, true)
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

func (s *ManagerSuite) TestCreateCommunity_WithBanner() {
	// Generate test image bigger than BannerDim
	testImage := image.NewRGBA(image.Rect(0, 0, 20, 10))

	tmpTestFilePath := s.T().TempDir() + "/test.png"
	file, err := os.Create(tmpTestFilePath)
	s.NoError(err)
	defer file.Close()

	err = png.Encode(file, testImage)
	s.Require().NoError(err)

	request := &requests.CreateCommunity{
		Name:        "with_banner",
		Description: "community with banner ",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Banner: userimages.CroppedImage{
			ImagePath: tmpTestFilePath,
			X:         1,
			Y:         1,
			Width:     10,
			Height:    5,
		},
	}

	community, err := s.manager.CreateCommunity(request, true)
	s.Require().NoError(err)
	s.Require().NotNil(community)

	communities, err := s.manager.All()
	s.Require().NoError(err)
	// Consider status default community
	s.Require().Len(communities, 2)
	s.Require().Equal(len(community.config.CommunityDescription.Identity.Images), 1)
	testIdentityImage, isMapContainsKey := community.config.CommunityDescription.Identity.Images[userimages.BannerIdentityName]
	s.Require().True(isMapContainsKey)
	s.Require().Positive(len(testIdentityImage.Payload))
}

func (s *ManagerSuite) TestEditCommunity() {
	//create community
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}

	community, err := s.manager.CreateCommunity(createRequest, true)
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

	community, _, err := s.buildCommunityWithChat()
	s.Require().NoError(err)
	s.Require().NotNil(community)

	adminChatIDs, err := s.manager.GetAdminCommunitiesChatIDs()
	s.Require().NoError(err)
	s.Require().Len(adminChatIDs, 1)
}

func buildMessage(timestamp time.Time, topic types.TopicType, hash []byte) types.Message {
	message := types.Message{
		Sig:       []byte{1},
		Timestamp: uint32(timestamp.Unix()),
		Topic:     topic,
		Payload:   []byte{1},
		Padding:   []byte{1},
		Hash:      hash,
	}
	return message
}

func (s *ManagerSuite) buildCommunityWithChat() (*Community, string, error) {
	createRequest := &requests.CreateCommunity{
		Name:        "status",
		Description: "status community description",
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
	}
	community, err := s.manager.CreateCommunity(createRequest, true)
	if err != nil {
		return nil, "", err
	}
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
	_, changes, err := s.manager.CreateChat(community.ID(), chat, true)
	if err != nil {
		return nil, "", err
	}

	chatID := ""
	for cID := range changes.ChatsAdded {
		chatID = cID
		break
	}
	return community, chatID, nil
}
