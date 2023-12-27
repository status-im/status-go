package protocol

import (
	"context"
	"errors"
	"math/big"

	"github.com/stretchr/testify/suite"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/communitytokens"
	"github.com/status-im/status-go/services/wallet/bigint"
)

type CommunityEventsTestsInterface interface {
	GetControlNode() *Messenger
	GetEventSender() *Messenger
	GetMember() *Messenger
	GetSuite() *suite.Suite
	GetCollectiblesServiceMock() *CollectiblesServiceMock
}

const communitiesEventsTestTokenAddress = "0x0400000000000000000000000000000000000000"
const aliceAccountAddress = "0x0777100000000000000000000000000000000000"
const bobAccountAddress = "0x0330000000000000000000000000000000000000"
const communitiesEventsTestChainID = 1
const eventsSenderAccountAddress = "0x0200000000000000000000000000000000000000"
const accountPassword = "qwerty"

type MessageResponseValidator func(*MessengerResponse) error

func WaitMessageCondition(response *MessengerResponse) bool {
	return len(response.Messages()) > 0
}

func waitOnMessengerResponse(s *suite.Suite, fnWait MessageResponseValidator, user *Messenger) {
	_, err := WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool {
			err := fnWait(r)
			return err == nil
		},
		"MessengerResponse data not received",
	)
	s.Require().NoError(err)
}

func checkClientsReceivedAdminEvent(base CommunityEventsTestsInterface, fn MessageResponseValidator) {
	s := base.GetSuite()
	// Wait and verify Member received community event
	waitOnMessengerResponse(s, fn, base.GetMember())
	// Wait and verify event sender received his own event
	waitOnMessengerResponse(s, fn, base.GetEventSender())
	// Wait and verify ControlNode received community event
	// ControlNode will publish CommunityDescription update
	waitOnMessengerResponse(s, fn, base.GetControlNode())
	// Wait and verify Member received the ControlNode CommunityDescription update
	waitOnMessengerResponse(s, fn, base.GetMember())
	// Wait and verify event sender received the ControlNode CommunityDescription update
	waitOnMessengerResponse(s, fn, base.GetEventSender())
	// Wait and verify ControlNode received his own CommunityDescription update
	waitOnMessengerResponse(s, fn, base.GetControlNode())
}

func refreshMessengerResponses(base CommunityEventsTestsInterface) {
	_, err := WaitOnMessengerResponse(base.GetControlNode(), func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	base.GetSuite().Require().NoError(err)

	_, err = WaitOnMessengerResponse(base.GetEventSender(), func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	base.GetSuite().Require().NoError(err)

	_, err = WaitOnMessengerResponse(base.GetMember(), func(response *MessengerResponse) bool {
		return true
	}, "community description changed message not received")
	base.GetSuite().Require().NoError(err)
}

func createMockedWalletBalance(s *suite.Suite) map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big {
	eventSenderAddress := gethcommon.HexToAddress(eventsSenderAccountAddress)

	mockedBalances := make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	mockedBalances[testChainID1] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	mockedBalances[testChainID1][eventSenderAddress] = make(map[gethcommon.Address]*hexutil.Big)

	// event sender will have token with `communitiesEventsTestTokenAddress``
	contractAddress := gethcommon.HexToAddress(communitiesEventsTestTokenAddress)
	balance, ok := new(big.Int).SetString("200", 10)
	s.Require().True(ok)
	decimalsFactor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(18)), nil)
	balance.Mul(balance, decimalsFactor)

	mockedBalances[communitiesEventsTestChainID][eventSenderAddress][contractAddress] = (*hexutil.Big)(balance)
	return mockedBalances
}

func setUpCommunityAndRoles(base CommunityEventsTestsInterface, role protobuf.CommunityMember_Roles) *communities.Community {
	tcs2, err := base.GetControlNode().communitiesManager.All()
	suite := base.GetSuite()
	suite.Require().NoError(err, "eventSender.communitiesManager.All")
	suite.Len(tcs2, 1, "Must have 1 community")

	// ControlNode creates a community and chat
	community := createTestCommunity(base, protobuf.CommunityPermissions_AUTO_ACCEPT)
	refreshMessengerResponses(base)

	// add events sender and member to the community
	advertiseCommunityTo(suite, community.ID(), base.GetControlNode(), base.GetEventSender())
	advertiseCommunityTo(suite, community.ID(), base.GetControlNode(), base.GetMember())

	request := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{eventsSenderAccountAddress},
		AirdropAddress:    eventsSenderAccountAddress,
	}
	joinCommunity(suite, community, base.GetControlNode(), base.GetEventSender(), request, accountPassword)
	refreshMessengerResponses(base)

	request = &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{aliceAccountAddress},
		AirdropAddress:    aliceAccountAddress,
	}
	joinCommunity(suite, community, base.GetControlNode(), base.GetMember(), request, accountPassword)
	refreshMessengerResponses(base)

	// grant permissions to the event sender
	grantPermission(suite, community, base.GetControlNode(), base.GetEventSender(), role)
	refreshMessengerResponses(base)

	return community
}

func createTestCommunity(base CommunityEventsTestsInterface, membershipType protobuf.CommunityPermissions_Access) *communities.Community {
	description := &requests.CreateCommunity{
		Membership:                  membershipType,
		Name:                        "status",
		Color:                       "#ffffff",
		Description:                 "status community description",
		PinMessageAllMembersEnabled: false,
	}
	response, err := base.GetControlNode().CreateCommunity(description, true)

	suite := base.GetSuite()
	suite.Require().NoError(err)
	suite.Require().NotNil(response)
	suite.Require().Len(response.Communities(), 1)
	suite.Require().Len(response.Chats(), 1)

	return response.Communities()[0]
}

func getModifiedCommunity(response *MessengerResponse, communityID string) (*communities.Community, error) {
	if len(response.Communities()) == 0 {
		return nil, errors.New("community not received")
	}

	var modifiedCommmunity *communities.Community = nil
	for _, c := range response.Communities() {
		if c.IDString() == communityID {
			modifiedCommmunity = c
		}
	}

	if modifiedCommmunity == nil {
		return nil, errors.New("couldn't find community in response")
	}

	return modifiedCommmunity, nil
}

func createCommunityChannel(base CommunityEventsTestsInterface, community *communities.Community, newChannel *protobuf.CommunityChat) string {
	checkChannelCreated := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == newChannel.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find created chat in response")
	}

	response, err := base.GetEventSender().CreateCommunityChat(community.ID(), newChannel)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkChannelCreated(response))
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].ChatsAdded, 1)
	var addedChatID string
	for addedChatID = range response.CommunityChanges[0].ChatsAdded {
		break
	}

	checkClientsReceivedAdminEvent(base, checkChannelCreated)

	return addedChatID
}

func editCommunityChannel(base CommunityEventsTestsInterface, community *communities.Community, editChannel *protobuf.CommunityChat, channelID string) {
	checkChannelEdited := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == editChannel.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find modified chat in response")
	}

	response, err := base.GetEventSender().EditCommunityChat(community.ID(), channelID, editChannel)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkChannelEdited(response))

	checkClientsReceivedAdminEvent(base, checkChannelEdited)
}

func deleteCommunityChannel(base CommunityEventsTestsInterface, community *communities.Community, channelID string) {
	checkChannelDeleted := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		if _, exists := modifiedCommmunity.Chats()[channelID]; exists {
			return errors.New("channel was not deleted")
		}

		return nil
	}

	response, err := base.GetEventSender().DeleteCommunityChat(community.ID(), channelID)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkChannelDeleted(response))

	checkClientsReceivedAdminEvent(base, checkChannelDeleted)
}

func createTestPermissionRequest(community *communities.Community, pType protobuf.CommunityTokenPermission_Type) *requests.CreateCommunityTokenPermission {
	return &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        pType,
		TokenCriteria: []*protobuf.TokenCriteria{
			{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{uint64(communitiesEventsTestChainID): communitiesEventsTestTokenAddress},
				Symbol:            "TEST",
				Amount:            "100",
				Decimals:          uint64(18),
			},
		},
	}
}

func createTokenPermission(base CommunityEventsTestsInterface, community *communities.Community, request *requests.CreateCommunityTokenPermission) (string, *requests.CreateCommunityTokenPermission) {
	response, err := base.GetEventSender().CreateCommunityTokenPermission(request)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].TokenPermissionsAdded, 1)

	addedPermission := func() *communities.CommunityTokenPermission {
		for _, permission := range response.CommunityChanges[0].TokenPermissionsAdded {
			return permission
		}
		return nil
	}()
	s.Require().NotNil(addedPermission)
	// Permission added by event must be in pending state
	s.Require().Equal(communities.TokenPermissionAdditionPending, addedPermission.State)

	responseHasApprovedTokenPermission := func(r *MessengerResponse) bool {
		if len(r.Communities()) == 0 {
			return false
		}

		receivedPermission := r.Communities()[0].TokenPermissionByID(addedPermission.Id)
		return receivedPermission != nil && receivedPermission.State == communities.TokenPermissionApproved
	}

	// Control node receives community event & approves it
	_, err = WaitOnMessengerResponse(base.GetControlNode(), responseHasApprovedTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)

	// Member receives updated community description
	_, err = WaitOnMessengerResponse(base.GetMember(), responseHasApprovedTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)

	// EventSender receives updated community description
	_, err = WaitOnMessengerResponse(base.GetEventSender(), responseHasApprovedTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)

	return addedPermission.Id, request
}

func createTestTokenPermission(base CommunityEventsTestsInterface, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) (string, *requests.CreateCommunityTokenPermission) {
	createTokenPermissionRequest := createTestPermissionRequest(community, pType)
	return createTokenPermission(base, community, createTokenPermissionRequest)
}

func editTokenPermission(base CommunityEventsTestsInterface, community *communities.Community, request *requests.EditCommunityTokenPermission) {
	s := base.GetSuite()

	response, err := base.GetEventSender().EditCommunityTokenPermission(request)
	s.Require().NoError(err)
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].TokenPermissionsModified, 1)

	editedPermission := response.CommunityChanges[0].TokenPermissionsModified[request.PermissionID]
	s.Require().NotNil(editedPermission)
	// Permission edited by event must be in pending state
	s.Require().Equal(communities.TokenPermissionUpdatePending, editedPermission.State)

	permissionSatisfyRequest := func(p *communities.CommunityTokenPermission) bool {
		return request.Type == p.Type &&
			request.TokenCriteria[0].Symbol == p.TokenCriteria[0].Symbol &&
			request.TokenCriteria[0].Amount == p.TokenCriteria[0].Amount &&
			request.TokenCriteria[0].Decimals == p.TokenCriteria[0].Decimals
	}
	s.Require().True(permissionSatisfyRequest(editedPermission))

	responseHasApprovedEditedTokenPermission := func(r *MessengerResponse) bool {
		if len(r.Communities()) == 0 {
			return false
		}

		receivedPermission := r.Communities()[0].TokenPermissionByID(editedPermission.Id)
		return receivedPermission != nil && receivedPermission.State == communities.TokenPermissionApproved &&
			permissionSatisfyRequest(receivedPermission)
	}

	// Control node receives community event & approves it
	_, err = WaitOnMessengerResponse(base.GetControlNode(), responseHasApprovedEditedTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)

	// Member receives updated community description
	_, err = WaitOnMessengerResponse(base.GetMember(), responseHasApprovedEditedTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)

	// EventSender receives updated community description
	_, err = WaitOnMessengerResponse(base.GetEventSender(), responseHasApprovedEditedTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)
}

func deleteTokenPermission(base CommunityEventsTestsInterface, community *communities.Community, request *requests.DeleteCommunityTokenPermission) {
	response, err := base.GetEventSender().DeleteCommunityTokenPermission(request)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].TokenPermissionsModified, 1)

	removedPermission := response.CommunityChanges[0].TokenPermissionsModified[request.PermissionID]
	s.Require().NotNil(removedPermission)
	// Permission removed by event must be in pending state
	s.Require().Equal(communities.TokenPermissionRemovalPending, removedPermission.State)

	responseHasNoTokenPermission := func(r *MessengerResponse) bool {
		if len(r.Communities()) == 0 {
			return false
		}

		return r.Communities()[0].TokenPermissionByID(removedPermission.Id) == nil
	}

	// Control node receives community event & approves it
	_, err = WaitOnMessengerResponse(base.GetControlNode(), responseHasNoTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)

	// Member receives updated community description
	_, err = WaitOnMessengerResponse(base.GetMember(), responseHasNoTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)

	// EventSender receives updated community description
	_, err = WaitOnMessengerResponse(base.GetEventSender(), responseHasNoTokenPermission, "community with approved permission not found")
	s.Require().NoError(err)
}

func assertCheckTokenPermissionCreated(s *suite.Suite, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) {
	permissions := community.TokenPermissionsByType(pType)
	s.Require().Len(permissions, 1)
	s.Require().Len(permissions[0].TokenCriteria, 1)
	s.Require().Equal(permissions[0].TokenCriteria[0].Type, protobuf.CommunityTokenType_ERC20)
	s.Require().Equal(permissions[0].TokenCriteria[0].Symbol, "TEST")
	s.Require().Equal(permissions[0].TokenCriteria[0].Amount, "100")
	s.Require().Equal(permissions[0].TokenCriteria[0].Decimals, uint64(18))
}

func setUpOnRequestCommunityAndRoles(base CommunityEventsTestsInterface, role protobuf.CommunityMember_Roles, additionalEventSenders []*Messenger) *communities.Community {
	tcs2, err := base.GetControlNode().communitiesManager.All()
	s := base.GetSuite()
	s.Require().NoError(err, "eventSender.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 community")

	// control node creates a community and chat
	community := createTestCommunity(base, protobuf.CommunityPermissions_MANUAL_ACCEPT)
	refreshMessengerResponses(base)

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), base.GetEventSender())
	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), base.GetMember())

	requestEventSender := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{eventsSenderAccountAddress},
		ENSName:           "eventSender",
		AirdropAddress:    eventsSenderAccountAddress,
	}

	joinOnRequestCommunity(s, community, base.GetControlNode(), base.GetEventSender(), requestEventSender)

	requestMember := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{aliceAccountAddress},
		ENSName:           "alice",
		AirdropAddress:    aliceAccountAddress,
	}
	joinOnRequestCommunity(s, community, base.GetControlNode(), base.GetMember(), requestMember)

	checkMemberJoined := func(response *MessengerResponse) error {
		return checkMemberJoinedToTheCommunity(response, base.GetMember().IdentityPublicKey())
	}

	waitOnMessengerResponse(s, checkMemberJoined, base.GetEventSender())

	// grant permissions to event sender
	grantPermission(s, community, base.GetControlNode(), base.GetEventSender(), role)
	checkPermissionGranted := func(response *MessengerResponse) error {
		return checkRolePermissionInResponse(response, base.GetEventSender().IdentityPublicKey(), role)
	}
	waitOnMessengerResponse(s, checkPermissionGranted, base.GetMember())

	for _, eventSender := range additionalEventSenders {
		advertiseCommunityTo(s, community.ID(), base.GetControlNode(), eventSender)
		joinOnRequestCommunity(s, community, base.GetControlNode(), eventSender, requestEventSender)

		grantPermission(s, community, base.GetControlNode(), eventSender, role)
		checkPermissionGranted = func(response *MessengerResponse) error {
			return checkRolePermissionInResponse(response, eventSender.IdentityPublicKey(), role)
		}
		waitOnMessengerResponse(s, checkPermissionGranted, base.GetMember())
		waitOnMessengerResponse(s, checkPermissionGranted, base.GetEventSender())
	}

	return community
}

func createCommunityCategory(base CommunityEventsTestsInterface, community *communities.Community, newCategory *requests.CreateCommunityCategory) string {
	checkCategoryCreated := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, category := range modifiedCommmunity.Categories() {
			if category.GetName() == newCategory.CategoryName {
				return nil
			}
		}

		return errors.New("couldn't find created Category in the response")
	}

	response, err := base.GetEventSender().CreateCommunityCategory(newCategory)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryCreated(response))
	s.Require().Len(response.Communities(), 1)
	s.Require().Len(response.CommunityChanges[0].CategoriesAdded, 1)

	var categoryID string
	for categoryID = range response.CommunityChanges[0].CategoriesAdded {
		break
	}

	checkClientsReceivedAdminEvent(base, checkCategoryCreated)

	return categoryID
}

func editCommunityCategory(base CommunityEventsTestsInterface, communityID string, editCategory *requests.EditCommunityCategory) {
	checkCategoryEdited := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, communityID)
		if err != nil {
			return err
		}

		for _, category := range modifiedCommmunity.Categories() {
			if category.GetName() == editCategory.CategoryName {
				return nil
			}
		}

		return errors.New("couldn't find edited Category in the response")
	}

	response, err := base.GetEventSender().EditCommunityCategory(editCategory)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryEdited(response))

	checkClientsReceivedAdminEvent(base, checkCategoryEdited)
}

func deleteCommunityCategory(base CommunityEventsTestsInterface, communityID string, deleteCategory *requests.DeleteCommunityCategory) {
	checkCategoryDeleted := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, communityID)
		if err != nil {
			return err
		}

		if _, exists := modifiedCommmunity.Chats()[deleteCategory.CategoryID]; exists {
			return errors.New("community was not deleted")
		}

		return nil
	}

	response, err := base.GetEventSender().DeleteCommunityCategory(deleteCategory)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryDeleted(response))

	checkClientsReceivedAdminEvent(base, checkCategoryDeleted)
}

func reorderCategory(base CommunityEventsTestsInterface, reorderRequest *requests.ReorderCommunityCategories) {
	checkCategoryReorder := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(reorderRequest.CommunityID))
		if err != nil {
			return err
		}

		category, exist := modifiedCommmunity.Categories()[reorderRequest.CategoryID]
		if !exist {
			return errors.New("couldn't find community category")
		}

		if int(category.Position) != reorderRequest.Position {
			return errors.New("category was not reordered")
		}

		return nil
	}

	response, err := base.GetEventSender().ReorderCommunityCategories(reorderRequest)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkCategoryReorder(response))

	checkClientsReceivedAdminEvent(base, checkCategoryReorder)
}

func reorderChannel(base CommunityEventsTestsInterface, reorderRequest *requests.ReorderCommunityChat) {
	checkChannelReorder := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(reorderRequest.CommunityID))
		if err != nil {
			return err
		}

		chat, exist := modifiedCommmunity.Chats()[reorderRequest.ChatID]
		if !exist {
			return errors.New("couldn't find community chat")
		}

		if int(chat.Position) != reorderRequest.Position {
			return errors.New("chat position was not reordered")
		}

		if chat.CategoryId != reorderRequest.CategoryID {
			return errors.New("chat category was not reordered")
		}

		return nil
	}

	response, err := base.GetEventSender().ReorderCommunityChat(reorderRequest)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkChannelReorder(response))

	checkClientsReceivedAdminEvent(base, checkChannelReorder)
}

func kickMember(base CommunityEventsTestsInterface, communityID types.HexBytes, pubkey string) {
	checkKicked := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(communityID))
		if err != nil {
			return err
		}

		if modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey) {
			return errors.New("alice was not kicked")
		}

		if len(modifiedCommmunity.PendingAndBannedMembers()) > 0 {
			return errors.New("alice was kicked and should not be presented in the pending list")
		}

		return nil
	}

	response, err := base.GetEventSender().RemoveUserFromCommunity(
		communityID,
		pubkey,
	)

	s := base.GetSuite()
	s.Require().NoError(err)

	// 1. event sender should get pending state for kicked member
	modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(communityID))
	s.Require().NoError(err)
	s.Require().True(modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey))
	s.Require().Equal(communities.CommunityMemberKickPending, modifiedCommmunity.PendingAndBannedMembers()[pubkey])

	// 2. wait for event as a sender
	waitOnMessengerResponse(s, func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(communityID))
		if err != nil {
			return err
		}

		if !modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey) {
			return errors.New("alice should not be not kicked (yet)")
		}

		if modifiedCommmunity.PendingAndBannedMembers()[pubkey] != communities.CommunityMemberKickPending {
			return errors.New("alice should be in the pending state")
		}

		return nil
	}, base.GetEventSender())

	// 3. wait for event as the community member and check we are still until control node gets it
	waitOnMessengerResponse(s, func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(communityID))
		if err != nil {
			return err
		}

		if !modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey) {
			return errors.New("alice should not be not kicked (yet)")
		}

		if len(modifiedCommmunity.PendingAndBannedMembers()) == 0 {
			return errors.New("alice should know about banned and pending members")
		}

		return nil
	}, base.GetMember())

	// 4. control node should handle event and actually kick member
	waitOnMessengerResponse(s, checkKicked, base.GetControlNode())

	// 5. event sender get removed member
	waitOnMessengerResponse(s, checkKicked, base.GetEventSender())

	// 6. member should be notified about actual removal
	waitOnMessengerResponse(s, checkKicked, base.GetMember())
}

func banMember(base CommunityEventsTestsInterface, banRequest *requests.BanUserFromCommunity) {
	pubkey := common.PubkeyToHex(&base.GetMember().identity.PublicKey)

	checkBanned := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(banRequest.CommunityID))
		if err != nil {
			return err
		}

		if modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey) {
			return errors.New("alice was not removed from the member list")
		}

		if !modifiedCommmunity.IsBanned(&base.GetMember().identity.PublicKey) {
			return errors.New("alice was not added to the banned list")
		}

		if modifiedCommmunity.PendingAndBannedMembers()[pubkey] != communities.CommunityMemberBanned {
			return errors.New("alice should be in the pending state")
		}

		return nil
	}

	response, err := base.GetEventSender().BanUserFromCommunity(context.Background(), banRequest)

	s := base.GetSuite()
	s.Require().NoError(err)

	// 1. event sender should get pending state for ban member
	modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(banRequest.CommunityID))
	s.Require().NoError(err)
	s.Require().True(modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey))
	s.Require().Equal(communities.CommunityMemberBanPending, modifiedCommmunity.PendingAndBannedMembers()[pubkey])

	// 2. wait for event as a sender
	waitOnMessengerResponse(s, func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(banRequest.CommunityID))
		if err != nil {
			return err
		}

		if !modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey) {
			return errors.New("alice should not be not banned (yet)")
		}

		if modifiedCommmunity.PendingAndBannedMembers()[pubkey] != communities.CommunityMemberBanPending {
			return errors.New("alice should be in the pending state")
		}

		return nil
	}, base.GetEventSender())

	// 3. wait for event as the community member and check we are still until control node gets it
	waitOnMessengerResponse(s, func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(banRequest.CommunityID))
		if err != nil {
			return err
		}

		if !modifiedCommmunity.HasMember(&base.GetMember().identity.PublicKey) {
			return errors.New("alice should not be not banned (yet)")
		}

		if len(modifiedCommmunity.PendingAndBannedMembers()) == 0 {
			return errors.New("alice should know about banned and pending members")
		}

		return nil
	}, base.GetMember())

	// 4. control node should handle event and actually ban member
	waitOnMessengerResponse(s, checkBanned, base.GetControlNode())

	// 5. event sender get banned member
	waitOnMessengerResponse(s, checkBanned, base.GetEventSender())

	// 6. member should be notified about actual removal
	waitOnMessengerResponse(s, checkBanned, base.GetMember())
}

func unbanMember(base CommunityEventsTestsInterface, unbanRequest *requests.UnbanUserFromCommunity) {
	pubkey := common.PubkeyToHex(&base.GetMember().identity.PublicKey)

	checkUnbanned := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(unbanRequest.CommunityID))
		if err != nil {
			return err
		}

		if modifiedCommmunity.IsBanned(&base.GetMember().identity.PublicKey) {
			return errors.New("alice was not unbanned")
		}

		if modifiedCommmunity.PendingAndBannedMembers()[pubkey] != communities.CommunityMemberBanned {
			return errors.New("alice should be in the pending state")
		}

		return nil
	}

	response, err := base.GetEventSender().UnbanUserFromCommunity(unbanRequest)

	s := base.GetSuite()
	s.Require().NoError(err)

	// 1. event sender should get pending state for unban member
	modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(unbanRequest.CommunityID))
	s.Require().NoError(err)
	s.Require().Equal(communities.CommunityMemberUnbanPending, modifiedCommmunity.PendingAndBannedMembers()[pubkey])

	// 2. wait for event as a sender
	waitOnMessengerResponse(s, func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(unbanRequest.CommunityID))
		if err != nil {
			return err
		}

		if modifiedCommmunity.PendingAndBannedMembers()[pubkey] != communities.CommunityMemberUnbanPending {
			return errors.New("alice should be in the pending state")
		}

		return nil
	}, base.GetEventSender())

	// 3. wait for event as the community member and check we are still until control node gets it
	waitOnMessengerResponse(s, func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, types.EncodeHex(unbanRequest.CommunityID))
		if err != nil {
			return err
		}

		if len(modifiedCommmunity.PendingAndBannedMembers()) == 0 {
			return errors.New("alice should know about banned and pending members")
		}

		return nil
	}, base.GetMember())

	// 4. control node should handle event and actually unban member
	waitOnMessengerResponse(s, checkUnbanned, base.GetControlNode())

	// 5. event sender get removed member
	waitOnMessengerResponse(s, checkUnbanned, base.GetEventSender())

	// 6. member should be notified about actual removal
	waitOnMessengerResponse(s, checkUnbanned, base.GetMember())
}

func controlNodeSendMessage(base CommunityEventsTestsInterface, inputMessage *common.Message) string {
	response, err := base.GetControlNode().SendChatMessage(context.Background(), inputMessage)

	s := base.GetSuite()
	s.Require().NoError(err)
	message := response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)
	messageID := message.ID

	response, err = WaitOnMessengerResponse(base.GetEventSender(), WaitMessageCondition, "messages not received")
	s.Require().NoError(err)
	message = response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)

	response, err = WaitOnMessengerResponse(base.GetMember(), WaitMessageCondition, "messages not received")
	s.Require().NoError(err)
	message = response.Messages()[0]
	s.Require().Equal(inputMessage.Text, message.Text)

	refreshMessengerResponses(base)

	return messageID
}

func deleteControlNodeMessage(base CommunityEventsTestsInterface, messageID string) {
	checkMessageDeleted := func(response *MessengerResponse) error {
		if len(response.RemovedMessages()) > 0 {
			return nil
		}
		return errors.New("message was not deleted")
	}

	response, err := base.GetEventSender().DeleteMessageAndSend(context.Background(), messageID)

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkMessageDeleted(response))

	waitOnMessengerResponse(s, checkMessageDeleted, base.GetMember())
	waitOnMessengerResponse(s, checkMessageDeleted, base.GetControlNode())
}

func pinControlNodeMessage(base CommunityEventsTestsInterface, pinnedMessage *common.PinMessage) {
	checkPinned := func(response *MessengerResponse) error {
		if len(response.Messages()) == 0 {
			return errors.New("no messages in the response")
		}

		if len(response.PinMessages()) > 0 {
			return nil
		}
		return errors.New("pin messages was not added")
	}

	response, err := base.GetEventSender().SendPinMessage(context.Background(), pinnedMessage)
	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().NoError(checkPinned(response))

	waitOnMessengerResponse(s, checkPinned, base.GetMember())
	waitOnMessengerResponse(s, checkPinned, base.GetControlNode())
}

func editCommunityDescription(base CommunityEventsTestsInterface, community *communities.Community) {
	expectedName := "edited community name"
	expectedColor := "#000000"
	expectedDescr := "edited community description"

	response, err := base.GetEventSender().EditCommunity(&requests.EditCommunity{
		CommunityID: community.ID(),
		CreateCommunity: requests.CreateCommunity{
			Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
			Name:        expectedName,
			Color:       expectedColor,
			Description: expectedDescr,
		},
	})

	checkCommunityEdit := func(response *MessengerResponse) error {
		if len(response.Communities()) == 0 {
			return errors.New("community not received")
		}

		rCommunities := response.Communities()
		if expectedName != rCommunities[0].Name() {
			return errors.New("incorrect community name")
		}

		if expectedColor != rCommunities[0].Color() {
			return errors.New("incorrect community color")
		}

		if expectedDescr != rCommunities[0].DescriptionText() {
			return errors.New("incorrect community description")
		}

		return nil
	}

	s := base.GetSuite()
	s.Require().NoError(err)
	s.Require().Nil(checkCommunityEdit(response))

	checkClientsReceivedAdminEvent(base, checkCommunityEdit)
}

func controlNodeCreatesCommunityPermission(base CommunityEventsTestsInterface, community *communities.Community, permissionRequest *requests.CreateCommunityTokenPermission) string {
	// control node creates permission
	response, err := base.GetControlNode().CreateCommunityTokenPermission(permissionRequest)
	s := base.GetSuite()
	s.Require().NoError(err)

	var tokenPermissionID string
	for id := range response.CommunityChanges[0].TokenPermissionsAdded {
		tokenPermissionID = id
	}
	s.Require().NotEqual(tokenPermissionID, "")

	ownerCommunity, err := base.GetControlNode().communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertCheckTokenPermissionCreated(s, ownerCommunity, permissionRequest.Type)

	// then, ensure event sender receives updated community
	resp, err := WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 &&
				len(r.Communities()[0].TokenPermissionsByType(permissionRequest.Type)) > 0 &&
				r.Communities()[0].HasPermissionToSendCommunityEvents()
		},
		"event sender did not receive community token permission",
	)
	s.Require().NoError(err)
	s.Require().NotNil(resp)
	eventSenderCommunity, err := base.GetEventSender().communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertCheckTokenPermissionCreated(s, eventSenderCommunity, permissionRequest.Type)
	s.Require().True(eventSenderCommunity.HasPermissionToSendCommunityEvents())

	return tokenPermissionID
}

func testCreateEditDeleteChannels(base CommunityEventsTestsInterface, community *communities.Community) {
	newChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat from the event sender",
			Emoji:       "",
			Description: "chat created by an event sender",
		},
	}

	newChatID := createCommunityChannel(base, community, newChat)

	newChat.Identity.DisplayName = "modified chat from event sender"
	editCommunityChannel(base, community, newChat, newChatID)
	deleteCommunityChannel(base, community, newChatID)
}

func testCreateEditDeleteBecomeMemberPermission(base CommunityEventsTestsInterface, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) {
	// first, create token permission
	tokenPermissionID, createTokenPermission := createTestTokenPermission(base, community, pType)

	createTokenPermission.TokenCriteria[0].Symbol = "UPDATED"
	createTokenPermission.TokenCriteria[0].Amount = "200"

	editTokenPermissionRequest := &requests.EditCommunityTokenPermission{
		PermissionID:                   tokenPermissionID,
		CreateCommunityTokenPermission: *createTokenPermission,
	}

	// then, event sender edits the permission
	editTokenPermission(base, community, editTokenPermissionRequest)

	deleteTokenPermissionRequest := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: tokenPermissionID,
	}

	// then, event sender deletes previously created token permission
	deleteTokenPermission(base, community, deleteTokenPermissionRequest)
}

func testAcceptMemberRequestToJoin(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger) {
	// set up additional user that will send request to join
	_, err := user.Start()

	s := base.GetSuite()

	s.Require().NoError(err)
	defer TearDownMessenger(s, user)

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID(), ENSName: "testName"}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	checkRequestToJoin := func(r *MessengerResponse) bool {
		if len(r.RequestsToJoinCommunity) == 0 {
			return false
		}
		for _, request := range r.RequestsToJoinCommunity {
			if request.ENSName == requestToJoin.ENSName {
				return true
			}
		}
		return false
	}
	// event sender receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		checkRequestToJoin,
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// control node receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		checkRequestToJoin,
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender has not accepted request yet
	eventSenderCommunity, err := base.GetEventSender().GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().False(eventSenderCommunity.HasMember(&user.identity.PublicKey))

	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	// we don't expect `user` to be a member already, because `eventSender` merely
	// forwards its accept decision to the control node
	s.Require().False(response.Communities()[0].HasMember(&user.identity.PublicKey))

	// at this point, the request to join is marked as accepted by GetEventSender node
	acceptedRequestsPending, err := base.GetEventSender().AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedRequestsPending, 1)
	s.Require().Equal(acceptedRequestsPending[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// user should not receive community admin event without being a member yet
	_, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"user did not receive community request to join response",
	)
	s.Require().Error(err)

	// control node receives community event with accepted membership request
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"control node did not receive community request to join response",
	)
	s.Require().NoError(err)

	// at this point, the request to join is marked as accepted by control node
	acceptedRequests, err := base.GetControlNode().AcceptedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	// we expect 3 here (1 event senders, 1 member + 1 from user)
	s.Require().Len(acceptedRequests, 3)
	s.Require().Equal(acceptedRequests[2].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// user receives updated community
	_, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"alice did not receive community request to join response",
	)
	s.Require().NoError(err)

	// event sender receives updated community
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"event sender did not receive community with the new member",
	)
	s.Require().NoError(err)

	// check control node notify event sender about accepting request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			acceptedRequests, err := base.GetEventSender().AcceptedRequestsToJoinForCommunity(community.ID())
			return err == nil && len(acceptedRequests) == 2 && (acceptedRequests[1].PublicKey == common.PubkeyToHex(&user.identity.PublicKey))
		},
		"no updates from control node",
	)

	s.Require().NoError(err)

	acceptedRequestsPending, err = base.GetEventSender().AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedRequestsPending, 0)
}

func testAcceptMemberRequestToJoinResponseSharedWithOtherEventSenders(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger, additionalEventSender *Messenger) {
	// set up additional user that will send request to join
	_, err := user.Start()

	s := base.GetSuite()

	s.Require().NoError(err)
	defer TearDownMessenger(s, user)

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 2 receives request to join
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 1 accepts request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	// event sender 2 receives decision of other event sender
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// at this point, the request to join is in accepted/pending state for event sender 2
	acceptedPendingRequests, err := additionalEventSender.AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedPendingRequests, 1)
	s.Require().Equal(acceptedPendingRequests[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// event sender 1 changes its mind and rejects the request
	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// event sender 2 receives updated decision of other event sender
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// at this point, the request to join is in declined/pending state for event sender 2
	rejectedPendingRequests, err := additionalEventSender.DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(rejectedPendingRequests, 1)
	s.Require().Equal(rejectedPendingRequests[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))
}

func testRejectMemberRequestToJoinResponseSharedWithOtherEventSenders(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger, additionalEventSender *Messenger) {
	// set up additional user that will send request to join
	_, err := user.Start()

	s := base.GetSuite()

	s.Require().NoError(err)
	defer TearDownMessenger(s, user)

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// event sender 2 receives request to join
	response, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// event sender 2 receives decision of other event sender
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// at this point, the request to join is in declined/pending state for event sender 2
	rejectedPendingRequests, err := additionalEventSender.DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(rejectedPendingRequests, 1)
	s.Require().Equal(rejectedPendingRequests[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// event sender 1 changes its mind and accepts the request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	// event sender 2 receives updated decision of other event sender
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// at this point, the request to join is in accepted/pending state for event sender 2
	acceptedPendingRequests, err := additionalEventSender.AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedPendingRequests, 1)
	s.Require().Equal(acceptedPendingRequests[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))
}

func testRejectMemberRequestToJoin(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger) {
	_, err := user.Start()

	s := base.GetSuite()
	s.Require().NoError(err)
	defer TearDownMessenger(s, user)

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// control node receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"control node did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// event sender has not accepted request yet
	eventSenderCommunity, err := base.GetEventSender().GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().False(eventSenderCommunity.HasMember(&user.identity.PublicKey))

	// event sender rejects request to join
	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = base.GetEventSender().DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)

	eventSenderCommunity, err = base.GetEventSender().GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().False(eventSenderCommunity.HasMember(&user.identity.PublicKey))

	requests, err := base.GetEventSender().DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().Len(requests, 1)
	s.Require().NoError(err)

	// control node receives event sender event and stores rejected request to join
	response, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool {
			requests, err := base.GetControlNode().DeclinedRequestsToJoinForCommunity(community.ID())
			s.Require().NoError(err)
			return len(response.Communities()) == 1 && len(requests) == 1
		},
		"control node did not receive community request to join update from event sender",
	)
	s.Require().NoError(err)
	s.Require().False(response.Communities()[0].HasMember(&user.identity.PublicKey))

	requests, err = base.GetControlNode().DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().Len(requests, 1)
	s.Require().NoError(err)

	// event sender receives updated community
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"event sender did not receive community update",
	)
	s.Require().NoError(err)

	// check control node notify event sender about declined request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			declinedRequests, err := base.GetEventSender().DeclinedRequestsToJoinForCommunity(community.ID())
			return err == nil && len(declinedRequests) == 1
		},
		"no updates from control node",
	)

	s.Require().NoError(err)

	declinedRequestsPending, err := base.GetEventSender().DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(declinedRequestsPending, 0)
}

func testControlNodeHandlesMultipleEventSenderRequestToJoinDecisions(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger, additionalEventSender *Messenger) {
	_, err := user.Start()

	s := base.GetSuite()
	s.Require().NoError(err)
	defer TearDownMessenger(s, user)

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	// event sender receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 2 receives request to join
	_, err = WaitOnMessengerResponse(
		additionalEventSender,
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// control node receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	// event sender 1 rejects request to join
	rejectRequestToJoin := &requests.DeclineRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = base.GetEventSender().DeclineRequestToJoinCommunity(rejectRequestToJoin)
	s.Require().NoError(err)
	// request to join is now marked as rejected pending for event sender 1
	rejectedPendingRequests, err := base.GetEventSender().DeclinedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(rejectedPendingRequests)
	s.Require().Len(rejectedPendingRequests, 1)

	// control node receives event sender 1's and 2's decision
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.RequestsToJoinCommunity) > 0 },
		"control node did not receive event senders decision",
	)
	s.Require().NoError(err)
	// request to join is now marked as rejected
	rejectedRequests, err := base.GetControlNode().DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(rejectedRequests)
	s.Require().Len(rejectedRequests, 1)

	// event sender 2 accepts request to join
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	_, err = additionalEventSender.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	// request to join is now marked as accepted pending for event sender 2
	acceptedPendingRequests, err := additionalEventSender.AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(acceptedPendingRequests)
	s.Require().Len(acceptedPendingRequests, 1)

	// control node now receives event sender 2's decision
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool { return len(r.Communities()) > 0 },
		"control node did not receive event senders decision",
	)
	s.Require().NoError(err)
	rejectedRequests, err = base.GetControlNode().DeclinedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(rejectedRequests)
	s.Require().Len(rejectedRequests, 1)
	// we expect user's request to join still to be rejected
	s.Require().Equal(rejectedRequests[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))
}

func testCreateEditDeleteCategories(base CommunityEventsTestsInterface, community *communities.Community) {
	newCategory := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "event-sender-category-name",
	}
	categoryID := createCommunityCategory(base, community, newCategory)

	editCategory := &requests.EditCommunityCategory{
		CommunityID:  community.ID(),
		CategoryID:   categoryID,
		CategoryName: "edited-event-sender-category-name",
	}

	editCommunityCategory(base, community.IDString(), editCategory)

	deleteCategory := &requests.DeleteCommunityCategory{
		CommunityID: community.ID(),
		CategoryID:  categoryID,
	}

	deleteCommunityCategory(base, community.IDString(), deleteCategory)
}

func testReorderChannelsAndCategories(base CommunityEventsTestsInterface, community *communities.Community) {
	newCategory := &requests.CreateCommunityCategory{
		CommunityID:  community.ID(),
		CategoryName: "event-sender-category-name",
	}
	_ = createCommunityCategory(base, community, newCategory)

	newCategory.CategoryName = "event-sender-category-name2"
	categoryID2 := createCommunityCategory(base, community, newCategory)

	chat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat from event-sender",
			Emoji:       "",
			Description: "chat created by an event-sender",
		},
	}

	chatID := createCommunityChannel(base, community, chat)

	reorderCommunityRequest := requests.ReorderCommunityCategories{
		CommunityID: community.ID(),
		CategoryID:  categoryID2,
		Position:    0,
	}

	reorderCategory(base, &reorderCommunityRequest)

	reorderChatRequest := requests.ReorderCommunityChat{
		CommunityID: community.ID(),
		CategoryID:  categoryID2,
		ChatID:      chatID,
		Position:    0,
	}

	reorderChannel(base, &reorderChatRequest)
}

func testEventSenderKickTheSameRole(base CommunityEventsTestsInterface, community *communities.Community) {
	// event sender tries to kick the member with the same role
	_, err := base.GetEventSender().RemoveUserFromCommunity(
		community.ID(),
		common.PubkeyToHex(&base.GetEventSender().identity.PublicKey),
	)

	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().EqualError(err, "not allowed to remove admin or owner")
}

func testEventSenderKickControlNode(base CommunityEventsTestsInterface, community *communities.Community) {
	// event sender tries to kick the control node
	_, err := base.GetEventSender().RemoveUserFromCommunity(
		community.ID(),
		common.PubkeyToHex(&base.GetControlNode().identity.PublicKey),
	)

	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().EqualError(err, "not allowed to remove admin or owner")
}

func testOwnerBanTheSameRole(base CommunityEventsTestsInterface, community *communities.Community) {
	_, err := base.GetEventSender().BanUserFromCommunity(
		context.Background(),
		&requests.BanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&base.GetEventSender().identity.PublicKey),
		},
	)

	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().EqualError(err, "not allowed to ban admin or owner")
}

func testOwnerBanControlNode(base CommunityEventsTestsInterface, community *communities.Community) {
	_, err := base.GetEventSender().BanUserFromCommunity(
		context.Background(),
		&requests.BanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&base.GetControlNode().identity.PublicKey),
		},
	)

	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().EqualError(err, "not allowed to ban admin or owner")
}

func testBanUnbanMember(base CommunityEventsTestsInterface, community *communities.Community) {
	// verify that event sender can't ban a control node
	_, err := base.GetEventSender().BanUserFromCommunity(
		context.Background(),
		&requests.BanUserFromCommunity{
			CommunityID: community.ID(),
			User:        common.PubkeyToHexBytes(&base.GetControlNode().identity.PublicKey),
		},
	)
	s := base.GetSuite()
	s.Require().Error(err)

	banRequest := &requests.BanUserFromCommunity{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(&base.GetMember().identity.PublicKey),
	}

	banMember(base, banRequest)

	unbanRequest := &requests.UnbanUserFromCommunity{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(&base.GetMember().identity.PublicKey),
	}

	unbanMember(base, unbanRequest)
}

func testDeleteAnyMessageInTheCommunity(base CommunityEventsTestsInterface, community *communities.Community) {
	chatID := community.ChatIDs()[0]

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "control node text"

	messageID := controlNodeSendMessage(base, inputMessage)

	deleteControlNodeMessage(base, messageID)
}

func testEventSenderPinMessage(base CommunityEventsTestsInterface, community *communities.Community) {
	s := base.GetSuite()
	s.Require().False(community.AllowsAllMembersToPinMessage())
	chatID := community.ChatIDs()[0]

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chatID
	inputMessage.ContentType = protobuf.ChatMessage_TEXT_PLAIN
	inputMessage.Text = "control node text"

	messageID := controlNodeSendMessage(base, inputMessage)

	pinnedMessage := common.NewPinMessage()
	pinnedMessage.MessageId = messageID
	pinnedMessage.ChatId = chatID
	pinnedMessage.Pinned = true

	pinControlNodeMessage(base, pinnedMessage)
}

func testMemberReceiveEventsWhenControlNodeOffline(base CommunityEventsTestsInterface, community *communities.Community) {
	// To simulate behavior when control node is offline, we will not use control node for listening new events
	// In this scenario member will reveive list of events

	s := base.GetSuite()
	member := base.GetMember()
	eventSender := base.GetEventSender()

	newAdminChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "chat from event sender",
			Emoji:       "",
			Description: "chat created by an event sender",
		},
	}

	checkChannelCreated := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == newAdminChat.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find created chat in response")
	}

	response, err := eventSender.CreateCommunityChat(community.ID(), newAdminChat)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelCreated(response))
	s.Require().Len(response.CommunityChanges, 1)
	s.Require().Len(response.CommunityChanges[0].ChatsAdded, 1)
	var addedChatID string
	for addedChatID = range response.CommunityChanges[0].ChatsAdded {
		break
	}

	waitOnMessengerResponse(s, checkChannelCreated, member)
	waitOnMessengerResponse(s, checkChannelCreated, eventSender)

	newAdminChat.Identity.DisplayName = "modified chat from event sender"

	checkChannelEdited := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, chat := range modifiedCommmunity.Chats() {
			if chat.GetIdentity().GetDisplayName() == newAdminChat.GetIdentity().GetDisplayName() {
				return nil
			}
		}

		return errors.New("couldn't find modified chat in response")
	}

	response, err = eventSender.EditCommunityChat(community.ID(), addedChatID, newAdminChat)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelEdited(response))

	waitOnMessengerResponse(s, checkChannelEdited, member)
	waitOnMessengerResponse(s, checkChannelEdited, eventSender)

	checkChannelDeleted := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		if _, exists := modifiedCommmunity.Chats()[addedChatID]; exists {
			return errors.New("channel was not deleted")
		}

		return nil
	}

	response, err = eventSender.DeleteCommunityChat(community.ID(), addedChatID)
	s.Require().NoError(err)
	s.Require().NoError(checkChannelDeleted(response))

	waitOnMessengerResponse(s, checkChannelDeleted, member)
	waitOnMessengerResponse(s, checkChannelDeleted, eventSender)
}

func testEventSenderCannotDeletePrivilegedCommunityPermission(base CommunityEventsTestsInterface, community *communities.Community,
	testPermissionType protobuf.CommunityTokenPermission_Type, rolePermissionType protobuf.CommunityTokenPermission_Type) {
	// Community should have eventSenderRole permission or eventSender will loose his role
	// after control node create a new community permission
	if testPermissionType != rolePermissionType {
		rolePermission := createTestPermissionRequest(community, rolePermissionType)
		controlNodeCreatesCommunityPermission(base, community, rolePermission)
	}

	permissionRequest := createTestPermissionRequest(community, testPermissionType)
	tokenPermissionID := controlNodeCreatesCommunityPermission(base, community, permissionRequest)

	deleteTokenPermission := &requests.DeleteCommunityTokenPermission{
		CommunityID:  community.ID(),
		PermissionID: tokenPermissionID,
	}

	// then event sender tries to delete permission which should fail
	response, err := base.GetEventSender().DeleteCommunityTokenPermission(deleteTokenPermission)
	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().Nil(response)
}

func testEventSenderCannotEditPrivilegedCommunityPermission(base CommunityEventsTestsInterface, community *communities.Community,
	testPermissionType protobuf.CommunityTokenPermission_Type, rolePermissionType protobuf.CommunityTokenPermission_Type) {

	// Community should have eventSenderRole permission or eventSender will loose his role
	// after control node create a new community permission
	if testPermissionType != rolePermissionType {
		rolePermission := createTestPermissionRequest(community, rolePermissionType)
		controlNodeCreatesCommunityPermission(base, community, rolePermission)
	}

	permissionRequest := createTestPermissionRequest(community, testPermissionType)
	tokenPermissionID := controlNodeCreatesCommunityPermission(base, community, permissionRequest)

	permissionRequest.TokenCriteria[0].Symbol = "UPDATED"
	permissionRequest.TokenCriteria[0].Amount = "200"

	permissionEditRequest := &requests.EditCommunityTokenPermission{
		PermissionID:                   tokenPermissionID,
		CreateCommunityTokenPermission: *permissionRequest,
	}

	// then, event sender tries to edit permission
	response, err := base.GetEventSender().EditCommunityTokenPermission(permissionEditRequest)
	s := base.GetSuite()
	s.Require().Error(err)
	s.Require().Nil(response)
}

func testAddAndSyncTokenFromControlNode(base CommunityEventsTestsInterface, community *communities.Community,
	privilegesLvl token.PrivilegesLevel) {
	tokenERC721 := createCommunityToken(community.IDString(), privilegesLvl)
	addCommunityTokenToCommunityTokensService(base, tokenERC721)

	s := base.GetSuite()

	_, err := base.GetControlNode().SaveCommunityToken(tokenERC721, nil)
	s.Require().NoError(err)

	err = base.GetControlNode().AddCommunityToken(tokenERC721.CommunityID, tokenERC721.ChainID, tokenERC721.Address)
	s.Require().NoError(err)

	tokens, err := base.GetEventSender().communitiesManager.GetAllCommunityTokens()
	s.Require().NoError(err)
	s.Require().Len(tokens, 0)

	checkTokenAdded := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		if privilegesLvl != token.CommunityLevel && len(modifiedCommmunity.TokenPermissions()) == 0 {
			return errors.New("Token permissions was not found")
		}

		for _, tokenMetadata := range modifiedCommmunity.CommunityTokensMetadata() {
			if tokenMetadata.Name == tokenERC721.Name {
				return nil
			}
		}

		return errors.New("Token was not found")
	}

	waitOnMessengerResponse(s, checkTokenAdded, base.GetMember())
	waitOnMessengerResponse(s, checkTokenAdded, base.GetEventSender())

	// check CommunityToken was added to the DB
	syncTokens, err := base.GetEventSender().communitiesManager.GetAllCommunityTokens()
	s.Require().NoError(err)
	s.Require().Len(syncTokens, 1)
	s.Require().Equal(syncTokens[0].PrivilegesLevel, privilegesLvl)

	// check CommunityToken was not added to the DB
	syncTokens, err = base.GetMember().communitiesManager.GetAllCommunityTokens()
	s.Require().NoError(err)
	s.Require().Len(syncTokens, 0)
}

func testAddAndSyncOwnerTokenFromControlNode(base CommunityEventsTestsInterface, community *communities.Community,
	privilegesLvl token.PrivilegesLevel) {
	tokenERC721 := createCommunityToken(community.IDString(), privilegesLvl)
	addCommunityTokenToCommunityTokensService(base, tokenERC721)

	s := base.GetSuite()

	_, err := base.GetControlNode().SaveCommunityToken(tokenERC721, nil)
	s.Require().NoError(err)

	err = base.GetControlNode().AddCommunityToken(tokenERC721.CommunityID, tokenERC721.ChainID, tokenERC721.Address)
	s.Require().NoError(err)

	tokens, err := base.GetEventSender().communitiesManager.GetAllCommunityTokens()
	s.Require().NoError(err)
	s.Require().Len(tokens, 0)

	// we only check that the community has been queued for validation
	checkTokenAdded := func(response *MessengerResponse) error {
		member := base.GetMember()
		communitiesToValidate, err := member.communitiesManager.CommunitiesToValidate()
		if err != nil {
			return err
		}
		if len(communitiesToValidate) == 0 || communitiesToValidate[community.IDString()] == nil {

			return errors.New("no communities to validate")
		}

		return nil
	}

	waitOnMessengerResponse(s, checkTokenAdded, base.GetMember())
}

func testEventSenderCannotCreatePrivilegedCommunityPermission(base CommunityEventsTestsInterface, community *communities.Community, pType protobuf.CommunityTokenPermission_Type) {
	permissionRequest := createTestPermissionRequest(community, pType)

	response, err := base.GetEventSender().CreateCommunityTokenPermission(permissionRequest)
	s := base.GetSuite()
	s.Require().Nil(response)
	s.Require().Error(err)
}

func createCommunityToken(communityID string, privilegesLevel token.PrivilegesLevel) *token.CommunityToken {
	return &token.CommunityToken{
		CommunityID:        communityID,
		TokenType:          protobuf.CommunityTokenType_ERC721,
		Address:            "0x123",
		Name:               "StatusToken",
		Symbol:             "STT",
		Description:        "desc",
		Supply:             &bigint.BigInt{Int: big.NewInt(123)},
		InfiniteSupply:     false,
		Transferable:       true,
		RemoteSelfDestruct: true,
		ChainID:            1,
		DeployState:        token.Deployed,
		Base64Image:        "ABCD",
		PrivilegesLevel:    privilegesLevel,
	}
}

func testAddAndSyncTokenFromEventSenderByControlNode(base CommunityEventsTestsInterface, community *communities.Community,
	privilegesLvl token.PrivilegesLevel) {
	tokenERC721 := createCommunityToken(community.IDString(), privilegesLvl)
	addCommunityTokenToCommunityTokensService(base, tokenERC721)

	s := base.GetSuite()

	_, err := base.GetEventSender().SaveCommunityToken(tokenERC721, nil)
	s.Require().NoError(err)

	err = base.GetEventSender().AddCommunityToken(tokenERC721.CommunityID, tokenERC721.ChainID, tokenERC721.Address)
	s.Require().NoError(err)

	tokens, err := base.GetControlNode().communitiesManager.GetAllCommunityTokens()
	s.Require().NoError(err)
	s.Require().Len(tokens, 0)

	checkTokenAdded := func(response *MessengerResponse) error {
		modifiedCommmunity, err := getModifiedCommunity(response, community.IDString())
		if err != nil {
			return err
		}

		for _, tokenMetadata := range modifiedCommmunity.CommunityTokensMetadata() {
			if tokenMetadata.Name == tokenERC721.Name {
				return nil
			}
		}

		return errors.New("Token was not found")
	}

	checkClientsReceivedAdminEvent(base, checkTokenAdded)

	// check event sender sent sync message to the control node
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool {
			tokens, err := base.GetControlNode().communitiesManager.GetAllCommunityTokens()
			return err == nil && len(tokens) == 1
		},
		"no token sync message from event sender",
	)

	s.Require().NoError(err)

	// check member did not receive sync message with the token
	_, err = WaitOnMessengerResponse(
		base.GetMember(),
		func(r *MessengerResponse) bool {
			tokens, err := base.GetMember().communitiesManager.GetAllCommunityTokens()
			return err == nil && len(tokens) == 1
		},
		"no token sync message from event sender",
	)

	s.Require().Error(err)
}

func testEventSenderAddTokenMasterAndOwnerToken(base CommunityEventsTestsInterface, community *communities.Community) {
	ownerToken := createCommunityToken(community.IDString(), token.OwnerLevel)
	addCommunityTokenToCommunityTokensService(base, ownerToken)

	s := base.GetSuite()

	_, err := base.GetEventSender().SaveCommunityToken(ownerToken, nil)
	s.Require().NoError(err)

	err = base.GetEventSender().AddCommunityToken(ownerToken.CommunityID, ownerToken.ChainID, ownerToken.Address)
	s.Require().Error(err, communities.ErrInvalidManageTokensPermission)

	tokenMasterToken := ownerToken
	tokenMasterToken.PrivilegesLevel = token.MasterLevel
	tokenMasterToken.Address = "0x124"

	_, err = base.GetEventSender().SaveCommunityToken(tokenMasterToken, nil)
	s.Require().NoError(err)

	err = base.GetEventSender().AddCommunityToken(ownerToken.CommunityID, ownerToken.ChainID, ownerToken.Address)
	s.Require().Error(err, communities.ErrInvalidManageTokensPermission)
}

func addCommunityTokenToCommunityTokensService(base CommunityEventsTestsInterface, token *token.CommunityToken) {
	data := &communitytokens.CollectibleContractData{
		TotalSupply:    token.Supply,
		Transferable:   token.Transferable,
		RemoteBurnable: token.RemoteSelfDestruct,
		InfiniteSupply: token.InfiniteSupply,
	}

	base.GetCollectiblesServiceMock().SetMockCollectibleContractData(uint64(token.ChainID), token.Address, data)
}

func testJoinedPrivilegedMemberReceiveRequestsToJoin(base CommunityEventsTestsInterface, community *communities.Community,
	bob *Messenger, newPrivilegedUser *Messenger, tokenPermissionType protobuf.CommunityTokenPermission_Type) {
	// create community permission
	rolePermission := createTestPermissionRequest(community, tokenPermissionType)
	controlNodeCreatesCommunityPermission(base, community, rolePermission)

	s := base.GetSuite()

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), bob)
	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), newPrivilegedUser)

	requestNewPrivilegedUser := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{eventsSenderAccountAddress},
		ENSName:           "newPrivilegedUser",
		AirdropAddress:    eventsSenderAccountAddress,
	}

	requestToJoinID := requestToJoinCommunity(s, base.GetControlNode(), newPrivilegedUser, requestNewPrivilegedUser)

	// accept join request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoinID}
	response, err := base.GetControlNode().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	updatedCommunity := response.Communities()[0]
	s.Require().NotNil(updatedCommunity)
	s.Require().True(updatedCommunity.HasMember(&newPrivilegedUser.identity.PublicKey))

	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 &&
				len(r.Communities()[0].TokenPermissionsByType(tokenPermissionType)) > 0 &&
				r.Communities()[0].HasPermissionToSendCommunityEvents()
		},
		"newPrivilegedUser did not receive privileged role",
	)

	s.Require().NoError(err)

	expectedLength := 3
	// newPrivilegedUser user should receive all requests to join with shared addresses from the control node
	waitAndCheckRequestsToJoin(s, newPrivilegedUser, expectedLength, community.ID(), tokenPermissionType == protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)

	// bob joins the community
	requestMember := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{bobAccountAddress},
		ENSName:           "bob",
		AirdropAddress:    bobAccountAddress,
	}

	bobRequestToJoinID := requestToJoinCommunity(s, base.GetControlNode(), bob, requestMember)

	// accept join request
	acceptRequestToJoin = &requests.AcceptRequestToJoinCommunity{ID: bobRequestToJoinID}
	_, err = base.GetControlNode().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)

	expectedLength = 4
	waitAndCheckRequestsToJoin(s, newPrivilegedUser, expectedLength, community.ID(), tokenPermissionType == protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func testMemberReceiveRequestsToJoinAfterGettingNewRole(base CommunityEventsTestsInterface, bob *Messenger, tokenPermissionType protobuf.CommunityTokenPermission_Type) {
	tcs2, err := base.GetControlNode().communitiesManager.All()
	s := base.GetSuite()
	s.Require().NoError(err, "eventSender.communitiesManager.All")
	s.Len(tcs2, 1, "Must have 1 community")

	// control node creates a community and chat
	community := createTestCommunity(base, protobuf.CommunityPermissions_MANUAL_ACCEPT)

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), base.GetEventSender())
	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), base.GetMember())
	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), bob)

	requestAlice := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{aliceAccountAddress},
		ENSName:           "alice",
		AirdropAddress:    aliceAccountAddress,
	}

	requestToJoinCommunity(s, base.GetControlNode(), base.GetMember(), requestAlice)

	requestBob := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{bobAccountAddress},
		ENSName:           "bob",
		AirdropAddress:    bobAccountAddress,
	}

	requestToJoinCommunity(s, base.GetControlNode(), bob, requestBob)

	requestEventSender := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{eventsSenderAccountAddress},
		ENSName:           "eventSender",
		AirdropAddress:    eventsSenderAccountAddress,
	}

	// event sender joins as simple user
	joinOnRequestCommunity(s, community, base.GetControlNode(), base.GetEventSender(), requestEventSender)

	// create community permission
	rolePermission := createTestPermissionRequest(community, tokenPermissionType)

	response, err := base.GetControlNode().CreateCommunityTokenPermission(rolePermission)
	s.Require().NoError(err)

	var tokenPermissionID string
	for id := range response.CommunityChanges[0].TokenPermissionsAdded {
		tokenPermissionID = id
	}
	s.Require().NotEqual(tokenPermissionID, "")

	ownerCommunity, err := base.GetControlNode().communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	assertCheckTokenPermissionCreated(s, ownerCommunity, rolePermission.Type)

	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 &&
				len(r.Communities()[0].TokenPermissionsByType(tokenPermissionType)) > 0 &&
				r.Communities()[0].HasPermissionToSendCommunityEvents()
		},
		"event sender did not receive privileged role",
	)

	s.Require().NoError(err)

	expectedLength := 3
	waitAndCheckRequestsToJoin(s, base.GetEventSender(), expectedLength, community.ID(), tokenPermissionType == protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER)
}

func waitAndCheckRequestsToJoin(s *suite.Suite, user *Messenger, expectedLength int, communityID types.HexBytes, checkRevealedAddresses bool) {
	_, err := WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool {
			requestsToJoin, err := user.communitiesManager.GetCommunityRequestsToJoinWithRevealedAddresses(communityID)
			if err != nil {
				return false
			}
			if len(requestsToJoin) != expectedLength {
				s.T().Log("invalid requests to join count:", len(requestsToJoin))
				return false
			}

			for _, request := range requestsToJoin {
				if request.PublicKey == common.PubkeyToHex(&user.identity.PublicKey) {
					if len(request.RevealedAccounts) != 1 {
						s.T().Log("our own requests to join must always have accounts revealed")
						return false
					}
				} else if checkRevealedAddresses {
					if len(request.RevealedAccounts) != 1 {
						s.T().Log("no accounts revealed")
						return false
					}
				} else {
					if len(request.RevealedAccounts) != 0 {
						s.T().Log("unexpected accounts revealed")
						return false
					}
				}
			}
			return true
		},
		"user did not receive all requests to join from the control node",
	)
	s.Require().NoError(err)
}

func testPrivilegedMemberAcceptsRequestToJoinAfterMemberLeave(base CommunityEventsTestsInterface, community *communities.Community, user *Messenger) {
	// set up additional user that will send request to join
	_, err := user.Start()

	s := base.GetSuite()

	s.Require().NoError(err)
	defer TearDownMessenger(s, user)

	advertiseCommunityTo(s, community.ID(), base.GetControlNode(), user)

	// user sends request to join
	requestToJoin := &requests.RequestToJoinCommunity{CommunityID: community.ID(), ENSName: "testName"}
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	sentRequest := response.RequestsToJoinCommunity[0]

	checkRequestToJoin := func(r *MessengerResponse) bool {
		if len(r.RequestsToJoinCommunity) == 0 {
			return false
		}
		for _, request := range r.RequestsToJoinCommunity {
			if request.ENSName == requestToJoin.ENSName {
				return true
			}
		}
		return false
	}
	// event sender receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		checkRequestToJoin,
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// control node receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		checkRequestToJoin,
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: sentRequest.ID}
	response, err = base.GetEventSender().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	// we don't expect `user` to be a member already, because `eventSender` merely
	// forwards its accept decision to the control node
	s.Require().False(response.Communities()[0].HasMember(&user.identity.PublicKey))

	// at this point, the request to join is marked as accepted by GetEventSender node
	acceptedRequestsPending, err := base.GetEventSender().AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedRequestsPending, 1)
	s.Require().Equal(acceptedRequestsPending[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// control node receives community event with accepted membership request
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"control node did not receive community request to join response",
	)
	s.Require().NoError(err)

	// at this point, the request to join is marked as accepted by control node
	acceptedRequests, err := base.GetControlNode().AcceptedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	// we expect 3 here (1 event senders, 1 member + 1 from user)
	s.Require().Len(acceptedRequests, 3)
	s.Require().Equal(acceptedRequests[2].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// user receives updated community
	_, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"alice did not receive community request to join response",
	)
	s.Require().NoError(err)

	// event sender receives updated community
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"event sender did not receive community with the new member",
	)
	s.Require().NoError(err)

	// check control node notify event sender about accepting request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			acceptedRequests, err := base.GetEventSender().AcceptedRequestsToJoinForCommunity(community.ID())
			return err == nil && len(acceptedRequests) == 2 && (acceptedRequests[1].PublicKey == common.PubkeyToHex(&user.identity.PublicKey))
		},
		"no updates from control node",
	)

	s.Require().NoError(err)

	acceptedRequestsPending, err = base.GetEventSender().AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedRequestsPending, 0)

	// user leaves the community
	response, err = user.LeaveCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	s.Require().False(response.Communities()[0].Joined())

	checkMemberLeave := func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0 && !r.Communities()[0].HasMember(&user.identity.PublicKey)
	}

	// check control node received member leave msg
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		checkMemberLeave,
		"control node did not receive member leave msg",
	)
	s.Require().NoError(err)

	// check event sender received member leave update from ControlNode
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		checkMemberLeave,
		"event sender did not receive member leave update",
	)
	s.Require().NoError(err)

	// user tries to rejoin again
	response, err = user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// event sender receives request to join
	response, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		checkRequestToJoin,
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	// control node receives request to join
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		checkRequestToJoin,
		"event sender did not receive community request to join",
	)
	s.Require().NoError(err)

	response, err = base.GetEventSender().AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)
	// we don't expect `user` to be a member already, because `eventSender` merely
	// forwards its accept decision to the control node
	s.Require().False(response.Communities()[0].HasMember(&user.identity.PublicKey))

	// at this point, the request to join is marked as accepted pending by GetEventSender node
	acceptedRequestsPending, err = base.GetEventSender().AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedRequestsPending, 1)
	s.Require().Equal(acceptedRequestsPending[0].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// control node receives community event with accepted membership request
	_, err = WaitOnMessengerResponse(
		base.GetControlNode(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"control node did not receive community request to join response",
	)
	s.Require().NoError(err)

	// at this point, the request to join is marked as accepted by control node
	acceptedRequests, err = base.GetControlNode().AcceptedRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	// we expect 3 here (1 event senders, 1 member + 1 from user)
	s.Require().Len(acceptedRequests, 3)
	s.Require().Equal(acceptedRequests[2].PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// user receives updated community
	_, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"user did not receive community request to join response",
	)
	s.Require().NoError(err)

	// event sender receives updated community
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"event sender did not receive community with the new member",
	)
	s.Require().NoError(err)

	// check control node notify event sender about accepting request to join
	_, err = WaitOnMessengerResponse(
		base.GetEventSender(),
		func(r *MessengerResponse) bool {
			acceptedRequests, err := base.GetEventSender().AcceptedRequestsToJoinForCommunity(community.ID())
			return err == nil && len(acceptedRequests) == 2 && (acceptedRequests[1].PublicKey == common.PubkeyToHex(&user.identity.PublicKey))
		},
		"no updates from control node",
	)

	s.Require().NoError(err)

	acceptedRequestsPending, err = base.GetEventSender().AcceptedPendingRequestsToJoinForCommunity(community.ID())
	s.Require().NoError(err)
	s.Require().Len(acceptedRequestsPending, 0)
}
