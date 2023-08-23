package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"io/ioutil"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/collectibles"
	walletToken "github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

type AccountManagerMock struct {
	AccountsMap map[string]string
}

func (m *AccountManagerMock) GetVerifiedWalletAccount(db *accounts.Database, address, password string) (*account.SelectedExtKey, error) {
	return &account.SelectedExtKey{
		Address: types.HexToAddress(address),
	}, nil
}

func (m *AccountManagerMock) CanRecover(rpcParams account.RecoverParams, revealedAddress types.Address) (bool, error) {
	return true, nil
}

func (m *AccountManagerMock) Sign(rpcParams account.SignParams, verifiedAccount *account.SelectedExtKey) (result types.HexBytes, err error) {
	return types.HexBytes{}, nil
}

func (m *AccountManagerMock) DeleteAccount(address types.Address) error {
	return nil
}

type TokenManagerMock struct {
	Balances *map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big
}

func (m *TokenManagerMock) GetAllChainIDs() ([]uint64, error) {
	chainIDs := make([]uint64, 0, len(*m.Balances))
	for key := range *m.Balances {
		chainIDs = append(chainIDs, key)
	}
	return chainIDs, nil
}

func (m *TokenManagerMock) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, error) {
	time.Sleep(100 * time.Millisecond) // simulate response time
	return *m.Balances, nil
}

func (m *TokenManagerMock) UpsertCustom(token walletToken.Token) error {
	time.Sleep(100 * time.Millisecond) // simulate response time
	return nil
}

type CollectiblesServiceMock struct {
	Collectibles map[uint64]map[string]*collectibles.CollectibleContractData
	Assets       map[uint64]map[string]*collectibles.AssetContractData
}

func (c *CollectiblesServiceMock) GetCollectibleContractData(chainID uint64, contractAddress string) (*collectibles.CollectibleContractData, error) {
	collectibleContractData, dataExists := c.Collectibles[chainID][contractAddress]
	if dataExists {
		return collectibleContractData, nil
	}
	return nil, nil
}

func (c *CollectiblesServiceMock) GetAssetContractData(chainID uint64, contractAddress string) (*collectibles.AssetContractData, error) {
	assetsContractData, dataExists := c.Assets[chainID][contractAddress]
	if dataExists {
		return assetsContractData, nil
	}
	return nil, nil
}

func (c *CollectiblesServiceMock) SetMockCollectibleContractData(chainID uint64, contractAddress string, collectible *collectibles.CollectibleContractData) {
	if c.Collectibles == nil {
		c.Collectibles = make(map[uint64]map[string]*collectibles.CollectibleContractData)
	}
	c.Collectibles[chainID] = make(map[string]*collectibles.CollectibleContractData)
	c.Collectibles[chainID][contractAddress] = collectible
}

func (c *CollectiblesServiceMock) SetMockAssetContractData(chainID uint64, contractAddress string, assetData *collectibles.AssetContractData) {
	if c.Assets == nil {
		c.Assets = make(map[uint64]map[string]*collectibles.AssetContractData)
	}
	c.Assets[chainID] = make(map[string]*collectibles.AssetContractData)
	c.Assets[chainID][contractAddress] = assetData
}

func newMessenger(s *suite.Suite, shh types.Waku, logger *zap.Logger, password string, walletAddresses []string,
	mockedBalances *map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, collectiblesService collectibles.ServiceInterface) *Messenger {
	accountsManagerMock := &AccountManagerMock{}
	accountsManagerMock.AccountsMap = make(map[string]string)
	for _, walletAddress := range walletAddresses {
		accountsManagerMock.AccountsMap[walletAddress] = types.EncodeHex(crypto.Keccak256([]byte(password)))
	}

	tokenManagerMock := &TokenManagerMock{
		Balances: mockedBalances,
	}

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	messenger, err := newCommunitiesTestMessenger(shh, privateKey, logger, accountsManagerMock, tokenManagerMock, collectiblesService)
	s.Require().NoError(err)

	currentDistributorObj, ok := messenger.communitiesKeyDistributor.(*CommunitiesKeyDistributorImpl)
	s.Require().True(ok)
	messenger.communitiesKeyDistributor = &TestCommunitiesKeyDistributor{
		CommunitiesKeyDistributorImpl: *currentDistributorObj,
		subscriptions:                 map[chan *CommunityAndKeyActions]bool{},
		mutex:                         sync.RWMutex{},
	}

	// add wallet account with keypair
	for _, walletAddress := range walletAddresses {
		kp := accounts.GetProfileKeypairForTest(false, true, false)
		kp.Accounts[0].Address = types.HexToAddress(walletAddress)
		err := messenger.settings.SaveOrUpdateKeypair(kp)
		s.Require().NoError(err)
	}

	walletAccounts, err := messenger.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Len(walletAccounts, len(walletAddresses))
	for i := range walletAddresses {
		s.Require().Equal(walletAccounts[i].Type, accounts.AccountTypeGenerated)
	}
	return messenger
}

func newCommunitiesTestMessenger(shh types.Waku, privateKey *ecdsa.PrivateKey, logger *zap.Logger, accountsManager account.Manager,
	tokenManager communities.TokenManager, collectiblesService collectibles.ServiceInterface) (*Messenger, error) {
	tmpfile, err := ioutil.TempFile("", "accounts-tests-")
	if err != nil {
		return nil, err
	}
	madb, err := multiaccounts.InitializeDB(tmpfile.Name())
	if err != nil {
		return nil, err
	}

	acc := generator.NewAccount(privateKey, nil)
	iai := acc.ToIdentifiedAccountInfo("")

	walletDb, err := helpers.SetupTestMemorySQLDB(walletdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}

	appDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	if err != nil {
		return nil, err
	}

	options := []Option{
		WithCustomLogger(logger),
		WithDatabase(appDb),
		WithWalletDatabase(walletDb),
		WithMultiAccounts(madb),
		WithAccount(iai.ToMultiAccount()),
		WithDatasync(),
		WithTokenManager(tokenManager),
	}

	if collectiblesService != nil {
		options = append(options, WithCollectiblesService(collectiblesService))
	}

	m, err := NewMessenger(
		"Test",
		privateKey,
		&testNode{shh: shh},
		uuid.New().String(),
		nil,
		accountsManager,
		options...,
	)
	if err != nil {
		return nil, err
	}

	err = m.Init()
	if err != nil {
		return nil, err
	}

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
		LatestDerivedPath:         0,
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

	return m, nil
}

func createCommunity(s *suite.Suite, owner *Messenger) (*communities.Community, *Chat) {
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_NO_MEMBERSHIP,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create an community chat
	response, err := owner.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	s.Require().Len(response.Communities(), 1)
	community := response.Communities()[0]
	s.Require().True(community.Joined())
	s.Require().True(community.IsControlNode())

	s.Require().Len(response.CommunitiesSettings(), 1)
	communitySettings := response.CommunitiesSettings()[0]
	s.Require().Equal(communitySettings.CommunityID, community.IDString())
	s.Require().Equal(communitySettings.HistoryArchiveSupportEnabled, false)

	orgChat := &protobuf.CommunityChat{
		Permissions: &protobuf.CommunityPermissions{
			Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
		},
		Identity: &protobuf.ChatIdentity{
			DisplayName: "status-core",
			Emoji:       "😎",
			Description: "status-core community chat",
		},
	}

	response, err = owner.CreateCommunityChat(community.ID(), orgChat)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Chats(), 1)

	return community, response.Chats()[0]
}

func advertiseCommunityTo(s *suite.Suite, community *communities.Community, owner *Messenger, user *Messenger) {
	chat := CreateOneToOneChat(common.PubkeyToHex(&user.identity.PublicKey), &user.identity.PublicKey, user.transport)

	inputMessage := common.NewMessage()
	inputMessage.ChatId = chat.ID
	inputMessage.Text = "some text"
	inputMessage.CommunityID = community.IDString()

	err := owner.SaveChat(chat)
	s.Require().NoError(err)
	_, err = owner.SendChatMessage(context.Background(), inputMessage)
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

func joinCommunity(s *suite.Suite, community *communities.Community, owner *Messenger, user *Messenger, request *requests.RequestToJoinCommunity) {
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
		response, err := owner.RetrieveAll()
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

func joinOnRequestCommunity(s *suite.Suite, community *communities.Community, controlNode *Messenger, user *Messenger) {
	// Request to join the community
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	response, err := user.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity, 1)

	requestToJoin := response.RequestsToJoinCommunity[0]
	s.Require().Equal(requestToJoin.PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	response, err = WaitOnMessengerResponse(
		controlNode,
		func(r *MessengerResponse) bool {
			return len(r.RequestsToJoinCommunity) > 0
		},
		"control node did not receive community request to join",
	)
	s.Require().NoError(err)

	userRequestToJoin := response.RequestsToJoinCommunity[0]
	s.Require().Equal(userRequestToJoin.PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	// accept join request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoin.ID}
	response, err = controlNode.AcceptRequestToJoinCommunity(acceptRequestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	updatedCommunity := response.Communities()[0]
	s.Require().NotNil(updatedCommunity)
	s.Require().True(updatedCommunity.HasMember(&user.identity.PublicKey))

	// receive request to join response
	_, err = WaitOnMessengerResponse(
		user,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"user did not receive request to join response",
	)
	s.Require().NoError(err)

	userCommunity, err := user.GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(userCommunity.HasMember(&user.identity.PublicKey))

	_, err = WaitOnMessengerResponse(
		controlNode,
		func(r *MessengerResponse) bool {
			return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
		},
		"control node did not receive request to join response",
	)
	s.Require().NoError(err)
}

func sendChatMessage(s *suite.Suite, sender *Messenger, chatID string, text string) *common.Message {
	msg := &common.Message{
		ChatMessage: &protobuf.ChatMessage{
			ChatId:      chatID,
			ContentType: protobuf.ChatMessage_TEXT_PLAIN,
			Text:        text,
		},
	}

	_, err := sender.SendChatMessage(context.Background(), msg)
	s.Require().NoError(err)

	return msg
}

func grantPermission(s *suite.Suite, community *communities.Community, controlNode *Messenger, target *Messenger, role protobuf.CommunityMember_Roles) {
	responseAddRole, err := controlNode.AddRoleToMember(&requests.AddRoleToMember{
		CommunityID: community.ID(),
		User:        common.PubkeyToHexBytes(target.IdentityPublicKey()),
		Role:        role,
	})
	s.Require().NoError(err)
	s.Require().NoError(checkRolePermissionInResponse(responseAddRole, target.IdentityPublicKey(), role))

	response, err := WaitOnMessengerResponse(target, func(response *MessengerResponse) bool {
		if len(response.Communities()) == 0 {
			return false
		}

		err := checkRolePermissionInResponse(response, target.IdentityPublicKey(), role)

		return err == nil
	}, "community description changed message not received")
	s.Require().NoError(err)
	s.Require().NoError(checkRolePermissionInResponse(response, target.IdentityPublicKey(), role))
}

func checkRolePermissionInResponse(response *MessengerResponse, member *ecdsa.PublicKey, role protobuf.CommunityMember_Roles) error {
	if len(response.Communities()) == 0 {
		return errors.New("Response does not contain communities")
	}
	rCommunities := response.Communities()
	switch role {
	case protobuf.CommunityMember_ROLE_OWNER:
		if !rCommunities[0].IsMemberOwner(member) {
			return errors.New("Member without owner role")
		}
	case protobuf.CommunityMember_ROLE_ADMIN:
		if !rCommunities[0].IsMemberAdmin(member) {
			return errors.New("Member without admin role")
		}
	case protobuf.CommunityMember_ROLE_TOKEN_MASTER:
		if !rCommunities[0].IsMemberTokenMaster(member) {
			return errors.New("Member without token master role")
		}
	default:
		return errors.New("Can't check unknonw member role")
	}

	return nil
}

func checkMemberJoinedToTheCommunity(response *MessengerResponse, member *ecdsa.PublicKey) error {
	if len(response.Communities()) == 0 {
		return errors.New("No communities in the response")
	}

	if !response.Communities()[0].HasMember(member) {
		return errors.New("Member was not added to the community")
	}

	return nil
}

func waitOnCommunitiesEvent(user *Messenger, condition func(*communities.Subscription) bool) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		for {
			select {
			case sub, more := <-user.communitiesManager.Subscribe():
				if !more {
					errCh <- errors.New("channel closed when waiting for communities event")
					return
				}

				if condition(sub) {
					return
				}

			case <-time.After(500 * time.Millisecond):
				errCh <- errors.New("timed out when waiting for communities event")
				return
			}
		}
	}()

	return errCh
}
