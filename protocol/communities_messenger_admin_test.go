package protocol

import (
	"context"
	"fmt"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/account/generator"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/waku"
)

func TestAdminMessengerCommunitiesSuite(t *testing.T) {
	suite.Run(t, new(AdminMessengerCommunitiesSuite))
}

type AdminMessengerCommunitiesSuite struct {
	suite.Suite
	owner *Messenger
	admin *Messenger
	alice *Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *AdminMessengerCommunitiesSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.owner = s.newMessenger()
	s.admin = s.newMessenger()
	s.alice = s.newMessenger()
	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.admin.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *AdminMessengerCommunitiesSuite) TearDownTest() {
	s.Require().NoError(s.owner.Shutdown())
	s.Require().NoError(s.admin.Shutdown())
	s.Require().NoError(s.alice.Shutdown())
	_ = s.logger.Sync()
}

func (s *AdminMessengerCommunitiesSuite) newMessengerWithOptions(shh types.Waku, privateKey *ecdsa.PrivateKey, options []Option) *Messenger {
	m, err := NewMessenger(
		"Test",
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		nil,
		nil,
		options...,
	)
	s.Require().NoError(err)

	err = m.Init()
	s.Require().NoError(err)

	config := params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}

	networks := json.RawMessage("{}")
	setting := settings.Settings{
		Address:                   types.HexToAddress("0x1122334455667788990011223344556677889900"),
		AnonMetricsShouldSend:     false,
		CurrentNetwork:            "mainnet_rpc",
		DappsAddress:              types.HexToAddress("0x1122334455667788990011223344556677889900"),
		InstallationID:            "d3efcff6-cffa-560e-a547-21d3858cbc51",
		KeyUID:                    "0x1122334455667788990011223344556677889900",
		Name:                      "Test",
		Networks:                  &networks,
		PhotoPath:                 "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADIAAAAyCAIAAACRXR/mAAAAjklEQVR4nOzXwQmFMBAAUZXUYh32ZB32ZB02sxYQQSZGsod55/91WFgSS0RM+SyjA56ZRZhFmEWYRRT6h+M6G16zrxv6fdJpmUWYRbxsYr13dKfanpN0WmYRZhGzXz6AWYRZRIfbaX26fT9Jk07LLMIsosPt9I/dTDotswizCG+nhFmEWYRZhFnEHQAA///z1CFkYamgfQAAAABJRU5ErkJggg==",
		PreviewPrivacy:            false,
		PublicKey:                 "0x04112233445566778899001122334455667788990011223344556677889900112233445566778899001122334455667788990011223344556677889900",
		SigningPhrase:             "yurt joey vibe",
		SendPushNotifications:     true,
		ProfilePicturesVisibility: 1,
		DefaultSyncPeriod:         777600,
		UseMailservers:            true,
		LinkPreviewRequestEnabled: true,
		SendStatusUpdates:         true,
		WalletRootAddress:         types.HexToAddress("0x1122334455667788990011223344556677889900")}

	_ = m.settings.CreateSettings(setting, config)

	return m
}

func (s *AdminMessengerCommunitiesSuite) newMessengerWithKey(shh types.Waku, privateKey *ecdsa.PrivateKey) *Messenger {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	s.Require().NoError(err)
	madb, err := multiaccounts.InitializeDB(tmpfile.Name())
	s.Require().NoError(err)

	acc := generator.NewAccount(privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo("")

	options := []Option{
		WithCustomLogger(s.logger),
		WithDatabaseConfig(":memory:", "somekey", sqlite.ReducedKDFIterationsNumber),
		WithMultiAccounts(madb),
		WithAccount(iai.ToMultiAccount()),
		WithDatasync(),
	}
	return s.newMessengerWithOptions(shh, privateKey, options)
}

func (s *AdminMessengerCommunitiesSuite) newMessenger() *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	return s.newMessengerWithKey(s.shh, privateKey)
}

// func (s *AdminMessengerCommunitiesSuite) TestAdminEditCommunityDescription() {
// 	// TODO admin test: update to include edit tags, logo, banner, request to join required setting, pin setting, etc...
// 	community := s.setUpCommunityAndRoles()
// 	s.adminEditsCommunityDescription(community)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminCreateEditDeleteChannels() {
// 	community := s.setUpCommunityAndRoles()

// 	newAdminChat := &protobuf.CommunityChat{
// 		Permissions: &protobuf.CommunityPermissions{
// 			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
// 		},
// 		Identity: &protobuf.ChatIdentity{
// 			DisplayName: "chat from admin",
// 			Emoji:       "",
// 			Description: "chat created by an admin",
// 		},
// 	}

// 	newChatID := s.adminCreateCommunityChannel(community, newAdminChat)

// 	newAdminChat.Identity.DisplayName = "modified chat from admin"
// 	s.adminEditCommunityChannel(community, newAdminChat, newChatID)

// 	s.adminDeleteCommunityChannel(community, newChatID)
// }
// func (s *AdminMessengerCommunitiesSuite) TestAdminCreateEditDeleteCategories() {
// 	s.setUpCommunityAndRoles()
// 	// TODO admin test: Create, edit and delete categories (allowed)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminReorderChannelsAndCategories() {
// 	s.setUpCommunityAndRoles()
// 	// TODO admin test: Reorder channels and categories (allowed)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminCreateBecomeMemberPermission() {
// 	community := s.setUpCommunityAndRoles()
// 	s.adminCreateTestTokenPermission(community)

// 	response, err := WaitOnMessengerResponse(
// 		s.owner,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"owner did not receive community",
// 	)
// 	s.Require().NoError(err)
// 	s.Require().Len(response.Communities(), 2)
// 	s.assertAdminTokenPermissionCreated(response.Communities()[0])

// 	ownerCommunity, err := s.owner.GetCommunityByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionCreated(ownerCommunity)

// 	response, err = WaitOnMessengerResponse(
// 		s.alice,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"alice did not receive community",
// 	)
// 	s.Require().NoError(err)
// 	s.Require().Len(response.Communities(), 2)
// 	s.assertAdminTokenPermissionCreated(response.Communities()[0])

// 	aliceCommunity, err := s.alice.GetCommunityByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionCreated(aliceCommunity)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminEditBecomeMemberPermission() {
// 	// first, create token permission
// 	community := s.setUpCommunityAndRoles()
// 	tokenPermissionID, createTokenPermission := s.adminCreateTestTokenPermission(community)

// 	// then, ensure owner receives it
// 	response, err := WaitOnMessengerResponse(
// 		s.owner,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"owner did not receive community",
// 	)
// 	s.Require().NoError(err)
// 	ownerCommunity, err := s.owner.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionCreated(ownerCommunity)

// 	// then, ensure alice receives it
// 	response, err = WaitOnMessengerResponse(
// 		s.alice,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"alice did not receive community",
// 	)
// 	s.Require().NoError(err)
// 	aliceCommunity, err := s.alice.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionCreated(aliceCommunity)

// 	createTokenPermission.TokenCriteria[0].Symbol = "UPDATED"
// 	createTokenPermission.TokenCriteria[0].Amount = "200"

// 	editTokenPermission := &requests.EditCommunityTokenPermission{
// 		PermissionID:                   tokenPermissionID,
// 		CreateCommunityTokenPermission: *createTokenPermission,
// 	}

// 	s.refreshMessengerResponses()
// 	// then, admin edits the permission
// 	response, err = s.admin.EditCommunityTokenPermission(editTokenPermission)
// 	s.Require().NoError(err)
// 	s.Require().Len(response.Communities(), 1)
// 	s.assertAdminTokenPermissionEdited(response.Communities()[0])

// 	// then, ensure owner receives and applies edits
// 	response, err = WaitOnMessengerResponse(
// 		s.owner,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"owner did not receive updated community",
// 	)
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionEdited(response.Communities()[0])
// 	ownerCommunity, err = s.owner.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionEdited(ownerCommunity)

// 	// then, ensure alice receives and applies edits
// 	response, err = WaitOnMessengerResponse(
// 		s.alice,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"alice did not receive updated community",
// 	)
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionEdited(response.Communities()[0])
// 	aliceCommunity, err = s.alice.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionEdited(aliceCommunity)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminDeleteBecomeMemberPermission() {
// 	community := s.setUpCommunityAndRoles()
// 	tokenPermissionID, _ := s.adminCreateTestTokenPermission(community)

// 	// then, ensure owner receives it
// 	_, err := WaitOnMessengerResponse(
// 		s.owner,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"owner did not receive community",
// 	)
// 	s.Require().NoError(err)
// 	ownerCommunity, err := s.owner.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionCreated(ownerCommunity)

// 	// then, ensure alice receives it
// 	_, err = WaitOnMessengerResponse(
// 		s.alice,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"alice did not receive community",
// 	)
// 	s.Require().NoError(err)
// 	aliceCommunity, err := s.alice.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionCreated(aliceCommunity)

// 	deleteTokenPermission := &requests.DeleteCommunityTokenPermission{
// 		CommunityID: community.ID(),
// 		PermissionID: tokenPermissionID,
// 	}

// 	s.refreshMessengerResponses()

// 	// then, admin deletes previously created token permission
// 	_, err = s.admin.DeleteCommunityTokenPermission(deleteTokenPermission)
// 	s.Require().NoError(err)
// 	adminCommunity, err := s.admin.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.Require().Len(adminCommunity.TokenPermissions(), 0)

// 	// then, ensure owner receives and applies deletion
// 	_, err = WaitOnMessengerResponse(
// 		s.owner,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"owner did not receive updated community",
// 	)
// 	s.Require().NoError(err)
// 	ownerCommunity, err = s.owner.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.Require().Len(ownerCommunity.TokenPermissions(), 0)

// 	// then, ensure alice receives and applies deletion
// 	_, err = WaitOnMessengerResponse(
// 		s.alice,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"alice did not receive updated community",
// 	)
// 	s.Require().NoError(err)
// 	aliceCommunity, err = s.alice.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.Require().Len(aliceCommunity.TokenPermissions(), 0)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminCannotCreateBecomeAdminPermission() {
// 	community := s.setUpCommunityAndRoles()

// 	permissionRequest := createTestPermissionRequest(community)
// 	permissionRequest.Type = protobuf.CommunityTokenPermission_BECOME_ADMIN

// 	response, err := s.admin.CreateCommunityTokenPermission(permissionRequest)
// 	s.Require().Nil(response)
// 	s.Require().Error(err)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminCannotEditBecomeAdminPermission() {

// 	community := s.setUpCommunityAndRoles()
// 	permissionRequest := createTestPermissionRequest(community)
// 	permissionRequest.Type = protobuf.CommunityTokenPermission_BECOME_ADMIN

// 	// owner creates BECOME_ADMIN permission
// 	response, err := s.owner.CreateCommunityTokenPermission(permissionRequest)
// 	s.Require().NoError(err)

// 	var tokenPermissionID string
// 	for id := range response.CommunityChanges[0].TokenPermissionsAdded {
// 		tokenPermissionID = id
// 	}
// 	s.Require().NotEqual(tokenPermissionID, "")

// 	ownerCommunity, err := s.owner.communitiesManager.GetByID(community.ID())
// 	s.assertAdminTokenPermissionCreated(ownerCommunity)

// 	// then, ensure admin receives updated community
// 	_, err = WaitOnMessengerResponse(
// 		s.admin,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"admin did not receive updated community",
// 	)
// 	s.Require().NoError(err)
// 	adminCommunity, err := s.admin.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionCreated(adminCommunity)

// 	permissionRequest.TokenCriteria[0].Symbol = "UPDATED"
// 	permissionRequest.TokenCriteria[0].Amount = "200"

// 	permissionEditRequest := &requests.EditCommunityTokenPermission{
// 		PermissionID: tokenPermissionID,
// 		CreateCommunityTokenPermission: *permissionRequest,
// 	}

// 	// then, admin tries to edit permission
// 	response, err = s.admin.EditCommunityTokenPermission(permissionEditRequest)
// 	s.Require().Error(err)
// 	s.Require().Nil(response)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminCannotDeleteBecomeAdminPermission() {

// 	community := s.setUpCommunityAndRoles()
// 	permissionRequest := createTestPermissionRequest(community)
// 	permissionRequest.Type = protobuf.CommunityTokenPermission_BECOME_ADMIN

// 	// owner creates BECOME_ADMIN permission
// 	response, err := s.owner.CreateCommunityTokenPermission(permissionRequest)
// 	s.Require().NoError(err)

// 	var tokenPermissionID string
// 	for id := range response.CommunityChanges[0].TokenPermissionsAdded {
// 		tokenPermissionID = id
// 	}
// 	s.Require().NotEqual(tokenPermissionID, "")

// 	// then, ensure admin receives updated community
// 	_, err = WaitOnMessengerResponse(
// 		s.admin,
// 		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
// 		"admin did not receive updated community",
// 	)
// 	s.Require().NoError(err)
// 	adminCommunity, err := s.admin.communitiesManager.GetByID(community.ID())
// 	s.Require().NoError(err)
// 	s.assertAdminTokenPermissionCreated(adminCommunity)

// 	deleteTokenPermission := &requests.DeleteCommunityTokenPermission{
// 		CommunityID: community.ID(),
// 		PermissionID: tokenPermissionID,
// 	}

// 	// then admin tries to delete BECOME_ADMIN permission which should fail
// 	response, err = s.admin.DeleteCommunityTokenPermission(deleteTokenPermission)
// 	s.Require().Error(err)
// 	s.Require().Nil(response)
// }

func (s *AdminMessengerCommunitiesSuite) TestAdminAcceptMemberRequestToJoin() {
	community := s.setUpCommunityAndRoles()

	// set up additional user that will send request to join
	user := s.newMessenger()
	_, err := user.Start()
	s.Require().NoError(err)

	s.advertiseCommunityTo(community, user)

	fmt.Println("\n\n>>>> USER SENDS REQUEST TO JOIN")
	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)
	_ = response.RequestsToJoinCommunity[0]

	// s.refreshMessengerResponses()
	// s.refreshMessengerResponses()


	fmt.Println("\n\n>>>> ADMINS FILTERS")
	for _, filter := range s.admin.transport.Filters() {
		fmt.Println("\n>>>>> LOADED FILTER: ", filter)
		fmt.Println(">>>>> LOADED FILTER (CHATID): ", filter.ChatID)
		fmt.Println(">>>>> LOADED FILTER (FilterID): ", filter.FilterID)
		fmt.Println(">>>>> LOADED FILTER (TOPIC): ", filter.Topic)
		fmt.Println(">>>>> LOADED FILTER (IdentityStr): ", filter.Identity)
		fmt.Println(">>>>> LOADED FILTER (OneToOne): ", filter.OneToOne)
	}


	fmt.Println("\n\n>>>> ADMIN WANTS TO RECEIVE REQUEST TO JOIN")
	// admin receives request to join
	response, err = WaitOnMessengerResponse(
		s.admin,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		// func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		// func(r *MessengerResponse) bool { return true },
		"admin did not receive community request to join",
	)
	s.Require().NoError(err)
	// fmt.Println("\n\n>>>> ADMIN WANTS TO RECEIVE REQUEST TO JOIN")
	// response, err = WaitOnMessengerResponse(
	// 	s.admin,
	// 	// func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
	// 	func(r *MessengerResponse) bool { return true },
	// 	"admin did not receive community request to join",
	// )
	// s.Require().NoError(err)
	// s.Require().Len(response.RequestsToJoinCommunity, 1)
	s.Require().True(false)

	// fmt.Println("\n >>>>> RECEIVING REQUEST TO JOIN")
	// // admin receives request to join
	// err = tt.RetryWithBackOff(func() error {
	// 	_, err := s.owner.RetrieveAll()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if len(response.RequestsToJoinCommunity) == 0 {
	// 		return errors.New("no requesto to join received")
	// 	}
	// 	return nil
	// })
	// s.Require().NoError(err)
	// s.Require().Len(response.RequestsToJoinCommunity, 1)

	// receivedRequest := response.RequestsToJoinCommunity[0]
	// s.Require().Equal(userRequest.ID, receivedRequest.ID)
	// fmt.Println("\n>>>>> REQUEST ID: ", receivedRequest.ID)
	// s.Require().Equal(receivedRequest.PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// // admin has not accepted request yet
	// adminCommunity, err := s.owner.GetCommunityByID(community.ID())
	// s.Require().NoError(err)
	// s.Require().False(adminCommunity.HasMember(&user.identity.PublicKey))

	// fmt.Println("\n >>>>> ACCEPTING REQUEST TO JOIN")
	// acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: receivedRequest.ID}
	// response, err = s.owner.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	// s.Require().NoError(err)
	// s.Require().NotNil(response)
	// s.Require().Len(response.Communities(), 1)
	// s.Require().True(response.Communities()[0].HasMember(&user.identity.PublicKey))
	// s.Require().False(true)

	// TODO admin test: Receive 'request to join' notifications, and ability to Accept or Reject (accept must be approved by owner node)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminCreateEditDeleteCategories() {
	community := s.setUpCommunityAndRoles()
	newCategory := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "admin-category-name",
	}
	categoryID := s.adminCreateCommunityCategory(community, newCategory)

	editCategory := &requests.EditCommunityCategory{
		CommunityID:  community.ID(),
		CategoryID:   categoryID,
		CategoryName: "edited-admin-category-name",
	}

	s.adminEditCommunityCategory(community.IDString(), editCategory)

	deleteCategory := &requests.DeleteCommunityCategory{
		CommunityID: community.ID(),
		CategoryID:  categoryID,
	}

	s.adminDeleteCommunityCategory(community.IDString(), deleteCategory)
}

// func (s *AdminMessengerCommunitiesSuite) TestAdminKickMember() {
// 	s.setUpCommunityAndRoles()
// 	// TODO admin test: Kick member (kick must be approved by owner node)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminBanMember() {
// 	s.setUpCommunityAndRoles()
// 	// TODO admin test: Ban members (ban must be approved by owner node)
// }

func (s *AdminMessengerCommunitiesSuite) TestAdminDeleteAnyMessageInTheCommunity() {
	community := s.setUpCommunityAndRoles()
	chatID := community.ChatIDs()[0]

	inputMessage := common.Message{}
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "owner text"

	messageID := s.ownerSendMessage(chatID, &inputMessage)

	s.adminDeleteMessage(messageID)
}

func (s *AdminMessengerCommunitiesSuite) TestAdminPinMessage() {
	community := s.setUpCommunityAndRoles()
	s.Require().False(community.AllowsAllMembersToPinMessage())
	chatID := community.ChatIDs()[0]

	inputMessage := common.Message{}
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "owner text"

	messageID := s.ownerSendMessage(chatID, &inputMessage)

	pinnedMessage := common.PinMessage{}
	pinnedMessage.MessageId = messageID
	pinnedMessage.ChatId = chatID
	pinnedMessage.Pinned = true

	s.adminPinMessage(&pinnedMessage)
}

// func (s *AdminMessengerCommunitiesSuite) TestAdminMintToken() {
// 	s.setUpCommunityAndRoles()
// 	// TODO admin test: Mint Tokens (rescticted)
// }

// func (s *AdminMessengerCommunitiesSuite) TestAdminAirdropTokens() {
// 	s.setUpCommunityAndRoles()
// 	// TODO admin test: Airdrop Tokens (restricted)
// }

// TODO admin test:
//	- would be nice to test on a regression and check that simple user can't do this actions
//  - test when user loses his admin permissions
//  - some other tests scenarious (review)

func (s *AdminMessengerCommunitiesSuite) setUpOnRequestCommunityAndRoles() *communities.Community {
	tcs2, err := s.owner.communitiesManager.All()
	s.Require().NoError(err, "admin.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 community")

	// owner creates a community and chat
	community := s.createCommunity(protobuf.CommunityPermissions_ON_REQUEST)
	s.advertiseCommunityTo(community, s.admin)
	s.advertiseCommunityTo(community, s.alice)

	s.refreshMessengerResponses()

	s.joinOnRequestCommunity(community, s.admin)
	s.joinOnRequestCommunity(community, s.alice)

	s.refreshMessengerResponses()

	// grant admin permissions to the admin
	s.grantAdminPermissions(community, s.admin)
	return community
}

func (s *AdminMessengerCommunitiesSuite) setUpCommunityAndRoles() *communities.Community {
	tcs2, err := s.owner.communitiesManager.All()
	s.Require().NoError(err, "admin.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 community")

	// owner creates a community and chat
	community := s.createCommunity(protobuf.CommunityPermissions_NO_MEMBERSHIP)
	//_ = s.createCommunityChat(community)
	s.refreshMessengerResponses()

	// add admin and alice to the community
	s.advertiseCommunityTo(community, s.admin)
	s.advertiseCommunityTo(community, s.alice)
	s.joinCommunity(community, s.admin)
	s.joinCommunity(community, s.alice)

	s.refreshMessengerResponses()

	// grant admin permissions to the admin
	s.grantAdminPermissions(community, s.admin)
	return community
}

func (s *AdminMessengerCommunitiesSuite) advertiseCommunityTo(community *communities.Community, user *Messenger) {
	chat := CreateOneToOneChat(common.PubkeyToHex(&user.identity.PublicKey), &user.identity.PublicKey, user.transport)

	inputMessage := &common.Message{}
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err := s.owner.SaveChat(chat)
	s.Require().NoError(err)
	_, err = s.owner.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)

	// Ensure community is received
	err = tt.RetryWithBackOff(func() error {
		response, err := user.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}
		return nil
	})
	s.Require().NoError(err)
}

func (s *AdminMessengerCommunitiesSuite) joinOnRequestCommunity(community *communities.Community, user *Messenger) {
	// Request to join the community
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin := response.RequestsToJoinCommunity[0]
	s.Require().Equal(requestToJoin.PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	response, err = WaitOnMessengerResponse(
		s.owner,
		func(r *MessengerResponse) bool { 
			return len(r.RequestsToJoinCommunity) > 0
		},
		"owner did not receive community request to join",
	)
	s.Require().NoError(err)

	userRequestToJoin := response.RequestsToJoinCommunity[0]
	s.Require().Equal(userRequestToJoin.PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// accept join request
	 acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoin.ID}
	 response, err = s.owner.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	 s.Require().NoError(err)
	 s.Require().NotNil(response)

	updatedCommunity := response.Communities()[0]
	s.Require().NotNil(updatedCommunity)
	s.Require().True(updatedCommunity.HasMember(&user.identity.PublicKey))

	// receive request to join response
	response, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool { 
			return len(r.Communities()) > 0
		},
		"user did not receive request to join response",
	)
	s.Require().NoError(err)
	userCommunity, err := user.GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(userCommunity.HasMember(&user.identity.PublicKey))
}

func (s *AdminMessengerCommunitiesSuite) joinCommunity(community *communities.Community, user *Messenger) {
	// Request to join the community
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	// Retrieve and accept join request
	err = tt.RetryWithBackOff(func() error {
		response, err := s.owner.RetrieveAll()
		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities in response (accept join request)")
		}
		if !response.Communities()[0].HasMember(&user.identity.PublicKey) {
			return errors.New("user not accepted")
		}
		return nil
	})
	s.Require().NoError(err)

	// Retrieve join request response
	err = tt.RetryWithBackOff(func() error {
		response, err := user.RetrieveAll()

		if err != nil {
			return err
		}
		if len(response.Communities()) == 0 {
			return errors.New("no communities in response (join request response)")
		}
		if !response.Communities()[0].HasMember(&user.identity.PublicKey) {
			return errors.New("user not a member")
		}
		return nil
	})
	s.Require().NoError(err)
}

func (s *AdminMessengerCommunitiesSuite) createCommunity(membershipType protobuf.CommunityPermissions_Access) *communities.Community {
	description := &requests.CreateCommunity{
		Membership:  membershipType,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
		PinMessageAllMembersEnabled: false,
	}
	response, err := s.owner.CreateCommunity(description, true)

	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.Chats(), 1)

	return response.Communities()[0]
}

func (s *AdminMessengerCommunitiesSuite) grantAdminPermissions(community *communities.Community, target *Messenger) {
	response_add_role, err := s.owner.AddRoleToMember(&requests.AddRoleToMember{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(target.IdentityPublicKey()),
		Role:        protobuf.CommunityMember_ROLE_ALL,
	})
	s.Require().NoError(err)

	checkAdminRole := func(response *MessengerResponse) bool {
		if len(response.Communities()) == 0 {
			return false
		}
		r_communities := response.Communities()
		s.Require().Len(r_communities, 1)
		s.Require().True(r_communities[0].IsMemberAdmin(target.IdentityPublicKey()))
		return true
	}

	checkAdminRole(response_add_role)

	_, err = WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
		return checkAdminRole(response)
	}, "community description changed message not received")
	s.Require().NoError(err)
	_, err = WaitOnMessengerResponse(s.alice, func(response *MessengerResponse) bool {
		return checkAdminRole(response)
	}, "community description changed message not received")
	s.Require().NoError(err)

	s.refreshMessengerResponses()
}

func (s *AdminMessengerCommunitiesSuite) adminEditsCommunityDescription(community *communities.Community) {
	expected_name := "edited community name"
	expected_color := "#000000"
	expected_descr := "edited community description"

	response, err := s.admin.EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_ON_REQUEST,
			Name:        expected_name,
			Color:       expected_color,
			Description: expected_descr,
		},
	})

	checkCommunityEdit := func(response *MessengerResponse) bool {
		if len(response.Communities()) == 0 {
			return false
		}

		r_communities := response.Communities()
		s.Require().Len(r_communities, 1)
		s.Equal(expected_name, r_communities[0].Name())
		s.Equal(expected_color, r_communities[0].Color())
		s.Equal(expected_descr, r_communities[0].DescriptionText())

		return true
	}

	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	checkCommunityEdit(response)

	_, err = WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
		return checkCommunityEdit(response)
	}, "admin edit community message not received by owner")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.alice, func(response *MessengerResponse) bool {
		return checkCommunityEdit(response)
	}, "admin edit community message not received by alice")
	s.Require().NoError(err)
}

func (s *AdminMessengerCommunitiesSuite) refreshMessengerResponses() {
	_, err := WaitOnMessengerResponse(s.owner, func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.alice, func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	s.Require().NoError(err)
}

type MessageResponseValidator func(*MessengerResponse) error
type WaitResponseValidator func(*MessengerResponse) bool

func WaitCommunityCondition(r *MessengerResponse) bool {
	return len(r.Communities()) > 0
}

func WaitMessageCondition(response *MessengerResponse) bool {
	return len(response.Messages()) > 0
}

func (s *AdminMessengerCommunitiesSuite) checkClientsReceivedAdminEvent(fnWait WaitResponseValidator, fn MessageResponseValidator) {
	response, err := WaitOnMessengerResponse(
		s.alice,
		fnWait,
		"MessengerResponse data not received",
	)
	s.Require().NoError(err)
	s.Require().NoError(fn(response))

	response, err = WaitOnMessengerResponse(
		s.owner,
		fnWait,
		"MessengerResponse data not received",
	)
	s.Require().NoError(err)
	s.Require().NoError(fn(response))

	s.refreshMessengerResponses()
}

func (s *AdminMessengerCommunitiesSuite) adminCreateTokenPermission(community *communities.Community, request *requests.CreateCommunityTokenPermission, assertFn func(*communities.Community)) (string, *requests.CreateCommunityTokenPermission) {
	response, err := s.admin.CreateCommunityTokenPermission(request)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	adminCommunity, err := s.admin.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertFn(adminCommunity)

	var tokenPermissionID string
	for id := range response.CommunityChanges[0].TokenPermissionsAdded {
		tokenPermissionID = id
	}
	s.Require().NotEqual(tokenPermissionID, "")

	return tokenPermissionID, request
}

func createTestPermissionRequest(community *communities.Community) *requests.CreateCommunityTokenPermission {
	return &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{uint64(1): "0x123"},
				Symbol:            "TEST",
				Amount:            "100",
				Decimals:          uint64(18),
			},
		},
	}
}

func (s *AdminMessengerCommunitiesSuite) adminCreateTestTokenPermission(community *communities.Community) (string, *requests.CreateCommunityTokenPermission) {
	createTokenPermission := createTestPermissionRequest(community)
	return s.adminCreateTokenPermission(community, createTokenPermission, s.assertAdminTokenPermissionCreated)
}

func (s *AdminMessengerCommunitiesSuite) assertAdminTokenPermissionCreated(community *communities.Community) {
	permissions := make([]*protobuf.CommunityTokenPermission, 0)
	tokenPermissions := community.TokenPermissions()
	for _, p := range tokenPermissions {
		permissions = append(permissions, p)
	}
	s.Require().Len(permissions, 1)
	s.Require().Len(permissions[0].TokenCriteria, 1)
	s.Require().Equal(permissions[0].TokenCriteria[0].Type, protobuf.CommunityTokenType_ERC20)
	s.Require().Equal(permissions[0].TokenCriteria[0].Symbol, "TEST")
	s.Require().Equal(permissions[0].TokenCriteria[0].Amount, "100")
	s.Require().Equal(permissions[0].TokenCriteria[0].Decimals, uint64(18))
}

func (s *AdminMessengerCommunitiesSuite) assertAdminTokenPermissionEdited(community *communities.Community) {
	permissions := community.TokenPermissionsByType(protobuf.CommunityTokenPermission_BECOME_MEMBER)
	s.Require().Len(permissions, 1)
	s.Require().Len(permissions[0].TokenCriteria, 1)
	s.Require().Equal(permissions[0].TokenCriteria[0].Type, protobuf.CommunityTokenType_ERC20)
	s.Require().Equal(permissions[0].TokenCriteria[0].Symbol, "UPDATED")
	s.Require().Equal(permissions[0].TokenCriteria[0].Amount, "200")
	s.Require().Equal(permissions[0].TokenCriteria[0].Decimals, uint64(18))
}

func (s *AdminMessengerCommunitiesSuite) adminCreateCommunityChannel(community *communities.Community, newChannel *protobuf.CommunityChat) string {
	checkChannelCreated := func(response *MessengerResponse) error {
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}

		var modifiedCommmunity *communities.Community = nil
		for _, c := range response.Communities() {
			if c.IDString() == community.IDString() {
				modifiedCommmunity = c
			}
		}

		if modifiedCommmunity == nil {
			return errors.New("couldn't find community in response")
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == newChannel.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find created chat in response")
	}

	response, err := s.admin.CreateCommunityChat(community.ID(), newChannel)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelCreated(response))
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].ChatsAdded, 1)
	var addedChatID string
	for addedChatID = range response.CommunityChanges[0].ChatsAdded {
		break
	}

	waitForResponse := func(r *MessengerResponse) bool { return len(r.Communities()) > 0 }
	s.checkClientsReceivedAdminEvent(waitForResponse, checkChannelCreated)

	return addedChatID
}

func (s *AdminMessengerCommunitiesSuite) adminEditCommunityChannel(community *communities.Community, editChannel *protobuf.CommunityChat, channelID string) {
	checkChannelEdited := func(response *MessengerResponse) error {
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}

		var modifiedCommmunity *communities.Community = nil
		for _, c := range response.Communities() {
			if c.IDString() == community.IDString() {
				modifiedCommmunity = c
			}
		}

		if modifiedCommmunity == nil {
			return errors.New("couldn't find community in response")
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == editChannel.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find modified chat in response")
	}

	_, err := WaitOnMessengerResponse(s.admin, func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	s.Require().NoError(err)

	response, err := s.admin.EditCommunityChat(community.ID(), channelID, editChannel)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelEdited(response))

	s.checkClientsReceivedAdminEvent(WaitCommunityCondition, checkChannelEdited)
}

func (s *AdminMessengerCommunitiesSuite) adminDeleteCommunityChannel(community *communities.Community, channelID string) {
	checkChannelDeleted := func(response *MessengerResponse) error {
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}

		var modifiedCommmunity *communities.Community = nil
		for _, c := range response.Communities() {
			if c.IDString() == community.IDString() {
				modifiedCommmunity = c
			}
		}

		if modifiedCommmunity == nil {
			return errors.New("couldn't find community in response")
		}

		if _, exists := modifiedCommmunity.Chats()[channelID]; exists {
			return errors.New("channel was not deleted")
		}

		return nil
	}

	response, err := s.admin.DeleteCommunityChat(community.ID(), channelID)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelDeleted(response))

	s.checkClientsReceivedAdminEvent(WaitCommunityCondition, checkChannelDeleted)
}

func (s *AdminMessengerCommunitiesSuite) adminCreateCommunityCategory(community *communities.Community, newCategory *requests.CreateCommunityCategory) string {
	checkCategoryCreated := func(response *MessengerResponse) error {
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}

		var modifiedCommmunity *communities.Community = nil
		for _, c := range response.Communities() {
			if c.IDString() == community.IDString() {
				modifiedCommmunity = c
			}
		}

		if modifiedCommmunity == nil {
			return errors.New("couldn't find community in response")
		}

		for _, category := range modifiedCommmunity.Categories() {
			if category.GetName() == newCategory.CategoryName {
				return nil
			}
		}

		return errors.New("couldn't find created Category in the response")
	}

	response, err := s.admin.CreateCommunityCategory(newCategory)
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryCreated(response))
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunityChanges[0].CategoriesAdded, 1)

	var categoryId string
	for categoryId = range response.CommunityChanges[0].CategoriesAdded {
		break
	}

	s.checkClientsReceivedAdminEvent(WaitCommunityCondition, checkCategoryCreated)

	return categoryId
}

func (s *AdminMessengerCommunitiesSuite) adminEditCommunityCategory(communityID string, editCategory *requests.EditCommunityCategory) {
	checkCategoryEdited := func(response *MessengerResponse) error {
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}

		var modifiedCommmunity *communities.Community = nil
		for _, c := range response.Communities() {
			if c.IDString() == communityID {
				modifiedCommmunity = c
			}
		}

		if modifiedCommmunity == nil {
			return errors.New("couldn't find community in response")
		}

		for _, category := range modifiedCommmunity.Categories() {
			if category.GetName() == editCategory.CategoryName {
				return nil
			}
		}

		return errors.New("couldn't find created Category in the response")
	}

	response, err := s.admin.EditCommunityCategory(editCategory)
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryEdited(response))

	s.checkClientsReceivedAdminEvent(WaitCommunityCondition, checkCategoryEdited)
}

func (s *AdminMessengerCommunitiesSuite) adminDeleteCommunityCategory(communityID string, deleteCategory *requests.DeleteCommunityCategory) {
	checkCategoryDeleted := func(response *MessengerResponse) error {
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}

		var modifiedCommmunity *communities.Community = nil
		for _, c := range response.Communities() {
			if c.IDString() == communityID {
				modifiedCommmunity = c
			}
		}

		if modifiedCommmunity == nil {
			return errors.New("couldn't find community in response")
		}

		if _, exists := modifiedCommmunity.Chats()[deleteCategory.CategoryID]; exists {
			return errors.New("community was not deleted")
		}

		return nil
	}

	response, err := s.admin.DeleteCommunityCategory(deleteCategory)
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryDeleted(response))

	s.checkClientsReceivedAdminEvent(WaitCommunityCondition, checkCategoryDeleted)
}

func (s *AdminMessengerCommunitiesSuite) ownerSendMessage(chatId string, inputMessage *common.Message) string {
	response, err := s.owner.SendChatMessage(context.Background(), inputMessage)
	s.Require().NoError(err)
	message := response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)
	messageID := message.ID

	response, err = WaitOnMessengerResponse(s.admin, WaitMessageCondition, "messages not received")
	s.Require().NoError(err)
	message = response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)

	response, err = WaitOnMessengerResponse(s.alice, WaitMessageCondition, "messages not received")
	s.Require().NoError(err)
	message = response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)

	s.refreshMessengerResponses()

	return messageID
}

func (s *AdminMessengerCommunitiesSuite) adminDeleteMessage(messageID string) {
	checkMessageDeleted := func(response *MessengerResponse) error {
		if len(response.RemovedMessages()) > 0 {
			return nil
		}
		return errors.New("message was not deleted")
	}

	response, err := s.admin.DeleteMessageAndSend(context.Background(), messageID)
	s.Require().NoError(err)
	s.Require().NoError(checkMessageDeleted(response))

	waitMessageCondition := func(response *MessengerResponse) bool {
		return len(response.RemovedMessages()) > 0
	}
	s.checkClientsReceivedAdminEvent(waitMessageCondition, checkMessageDeleted)
}

func (s *AdminMessengerCommunitiesSuite) adminPinMessage(pinnedMessage *common.PinMessage) {
	checkPinned := func(response *MessengerResponse) error {
		if len(response.PinMessages()) > 0 {
			return nil
		}
		return errors.New("pin messages was not added")
	}

	response, err := s.admin.SendPinMessage(context.Background(), pinnedMessage)
	s.Require().NoError(err)
	s.Require().NoError(checkPinned(response))

	s.checkClientsReceivedAdminEvent(WaitMessageCondition, checkPinned)
}
