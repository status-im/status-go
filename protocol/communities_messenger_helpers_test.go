package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/stretchr/testify/suite"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	walletToken "github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/wallettypes"
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
	// mock signature
	bytesArray := []byte(rpcParams.Address)
	bytesArray = append(bytesArray, []byte(rpcParams.Password)...)
	bytesArray = common.Shake256(bytesArray)
	return append([]byte{0}, bytesArray...), nil
}

func (m *AccountManagerMock) DeleteAccount(address types.Address) error {
	return nil
}

type TokenManagerMock struct {
	Balances *communities.BalancesByChain
}

func (m *TokenManagerMock) GetAllChainIDs() ([]uint64, error) {
	chainIDs := make([]uint64, 0, len(*m.Balances))
	for key := range *m.Balances {
		chainIDs = append(chainIDs, key)
	}
	return chainIDs, nil
}

func (m *TokenManagerMock) getBalanceBasedOnParams(accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big {
	retBalances := make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)

	for _, chainId := range chainIDs {
		if _, exists := retBalances[chainId]; !exists {
			retBalances[chainId] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
		}
		if storedAccounts, exists := (*m.Balances)[chainId]; exists {
			for _, account := range accounts {
				if _, exists := retBalances[chainId][account]; !exists {
					retBalances[chainId][account] = make(map[gethcommon.Address]*hexutil.Big)
				}
				if storedTokenAddresses, exists := storedAccounts[account]; exists {
					for _, tokenAddress := range tokenAddresses {
						if _, exists := retBalances[chainId][account][tokenAddress]; !exists {
							retBalances[chainId][account] = make(map[gethcommon.Address]*hexutil.Big)
						}

						if balance, exists := storedTokenAddresses[tokenAddress]; exists {
							retBalances[chainId][account][tokenAddress] = balance
						}
					}
				}
			}
		}
	}

	return retBalances
}

func (m *TokenManagerMock) GetBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, error) {
	time.Sleep(100 * time.Millisecond) // simulate response time
	return m.getBalanceBasedOnParams(accounts, tokenAddresses, chainIDs), nil
}

func (m *TokenManagerMock) GetCachedBalancesByChain(ctx context.Context, accounts, tokenAddresses []gethcommon.Address, chainIDs []uint64) (map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big, error) {
	time.Sleep(100 * time.Millisecond) // simulate response time
	return m.getBalanceBasedOnParams(accounts, tokenAddresses, chainIDs), nil
}

func (m *TokenManagerMock) FindOrCreateTokenByAddress(ctx context.Context, chainID uint64, address gethcommon.Address) *walletToken.Token {
	time.Sleep(100 * time.Millisecond) // simulate response time
	return nil
}

type CollectiblesManagerMock struct {
	Collectibles                 *communities.CollectiblesByChain
	collectibleOwnershipResponse map[string][]thirdparty.AccountBalance
}

func (m *CollectiblesManagerMock) FetchCachedBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID,
	ownerAddress gethcommon.Address, contractAddresses []gethcommon.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	return m.FetchBalancesByOwnerAndContractAddress(ctx, chainID, ownerAddress, contractAddresses)
}

func (m *CollectiblesManagerMock) FetchBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID,
	ownerAddress gethcommon.Address, contractAddresses []gethcommon.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	ret := make(thirdparty.TokenBalancesPerContractAddress)
	accountsBalances, ok := (*m.Collectibles)[uint64(chainID)]
	if !ok {
		return ret, nil
	}

	balances, ok := accountsBalances[ownerAddress]
	if !ok {
		return ret, nil
	}

	for _, contractAddress := range contractAddresses {
		balance, ok := balances[contractAddress]
		if ok {
			ret[contractAddress] = balance
		}
	}

	return ret, nil
}

func (m *CollectiblesManagerMock) GetCollectibleOwnership(requestedID thirdparty.CollectibleUniqueID) ([]thirdparty.AccountBalance, error) {
	for id, balances := range m.collectibleOwnershipResponse {
		if id == requestedID.HashKey() {
			return balances, nil
		}
	}
	return []thirdparty.AccountBalance{}, nil
}

func (m *CollectiblesManagerMock) FetchCollectibleOwnersByContractAddress(ctx context.Context, chainID walletCommon.ChainID, contractAddress gethcommon.Address) (*thirdparty.CollectibleContractOwnership, error) {
	ret := &thirdparty.CollectibleContractOwnership{
		ContractAddress: contractAddress,
		Owners:          []thirdparty.CollectibleOwner{},
	}
	accountsBalances, ok := (*m.Collectibles)[uint64(chainID)]
	if !ok {
		return ret, nil
	}

	for wallet, balances := range accountsBalances {
		balance, ok := balances[contractAddress]
		if ok {
			ret.Owners = append(ret.Owners, thirdparty.CollectibleOwner{
				OwnerAddress:  wallet,
				TokenBalances: balance,
			})
		}
	}

	return ret, nil
}

func (m *CollectiblesManagerMock) SetCollectibleOwnershipResponse(id thirdparty.CollectibleUniqueID, balances []thirdparty.AccountBalance) {
	if m.collectibleOwnershipResponse == nil {
		m.collectibleOwnershipResponse = map[string][]thirdparty.AccountBalance{}
	}
	m.collectibleOwnershipResponse[id.HashKey()] = balances
}

type CollectiblesServiceMock struct {
	Collectibles map[uint64]map[string]*communities.CollectibleContractData
	Assets       map[uint64]map[string]*communities.AssetContractData
	Signers      map[string]string
}

func (c *CollectiblesServiceMock) SetSignerPubkeyForCommunity(communityID []byte, signerPubKey string) {
	if c.Signers == nil {
		c.Signers = make(map[string]string)
	}
	c.Signers[types.EncodeHex(communityID)] = signerPubKey
}

func (c *CollectiblesServiceMock) SetSignerPubKey(ctx context.Context, chainID uint64, contractAddress string, txArgs wallettypes.SendTxArgs, password string, newSignerPubKey string) (string, error) {
	return "", nil
}

func (c *CollectiblesServiceMock) GetCollectibleContractData(chainID uint64, contractAddress string) (*communities.CollectibleContractData, error) {
	collectibleContractData, dataExists := c.Collectibles[chainID][contractAddress]
	if dataExists {
		return collectibleContractData, nil
	}
	return nil, nil
}

func (c *CollectiblesServiceMock) GetAssetContractData(chainID uint64, contractAddress string) (*communities.AssetContractData, error) {
	assetsContractData, dataExists := c.Assets[chainID][contractAddress]
	if dataExists {
		return assetsContractData, nil
	}
	return nil, nil
}

func (c *CollectiblesServiceMock) SetMockCollectibleContractData(chainID uint64, contractAddress string, collectible *communities.CollectibleContractData) {
	if c.Collectibles == nil {
		c.Collectibles = make(map[uint64]map[string]*communities.CollectibleContractData)
	}
	if _, ok := c.Collectibles[chainID]; !ok {
		c.Collectibles[chainID] = make(map[string]*communities.CollectibleContractData)
	}
	c.Collectibles[chainID][contractAddress] = collectible
}

func (c *CollectiblesServiceMock) SetMockCommunityTokenData(token *token.CommunityToken) {
	if c.Collectibles == nil {
		c.Collectibles = make(map[uint64]map[string]*communities.CollectibleContractData)
	}

	data := &communities.CollectibleContractData{
		TotalSupply:    token.Supply,
		Transferable:   token.Transferable,
		RemoteBurnable: token.RemoteSelfDestruct,
		InfiniteSupply: token.InfiniteSupply,
	}

	c.SetMockCollectibleContractData(uint64(token.ChainID), token.Address, data)
}

func (c *CollectiblesServiceMock) SafeGetSignerPubKey(ctx context.Context, chainID uint64, communityID string) (string, error) {
	if c.Signers == nil {
		c.Signers = make(map[string]string)
	}
	return c.Signers[communityID], nil
}

func (c *CollectiblesServiceMock) SetMockAssetContractData(chainID uint64, contractAddress string, assetData *communities.AssetContractData) {
	if c.Assets == nil {
		c.Assets = make(map[uint64]map[string]*communities.AssetContractData)
	}
	c.Assets[chainID] = make(map[string]*communities.AssetContractData)
	c.Assets[chainID][contractAddress] = assetData
}

func (c *CollectiblesServiceMock) DeploymentSignatureDigest(chainID uint64, addressFrom string, communityID string) ([]byte, error) {
	return gethcommon.Hex2Bytes("ccbb375343347491706cf4b43796f7b96ccc89c9e191a8b78679daeba1684ec7"), nil
}

func (s *CollectiblesServiceMock) ProcessCommunityTokenAction(message *protobuf.CommunityTokenAction) error {
	return nil
}

type testCommunitiesMessengerConfig struct {
	testMessengerConfig

	nodeConfig  *params.NodeConfig
	appSettings *settings.Settings

	password            string
	walletAddresses     []string
	mockedBalances      *communities.BalancesByChain
	collectiblesService communities.CommunityTokensServiceInterface
	collectiblesManager communities.CollectiblesManager
}

func (tcmc *testCommunitiesMessengerConfig) complete() error {
	err := tcmc.testMessengerConfig.complete()
	if err != nil {
		return err
	}

	if tcmc.nodeConfig == nil {
		tcmc.nodeConfig = defaultTestCommunitiesMessengerNodeConfig()
	}
	if tcmc.appSettings == nil {
		tcmc.appSettings = defaultTestCommunitiesMessengerSettings()
	}

	return nil
}

func defaultTestCommunitiesMessengerNodeConfig() *params.NodeConfig {
	return &params.NodeConfig{
		NetworkID: 10,
		DataDir:   "test",
	}
}
func defaultTestCommunitiesMessengerSettings() *settings.Settings {
	networks := json.RawMessage("{}")
	return &settings.Settings{
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
}

func newTestCommunitiesMessenger(s *suite.Suite, waku types.Waku, config testCommunitiesMessengerConfig) *Messenger {
	err := config.complete()
	s.Require().NoError(err)

	accountsManagerMock := &AccountManagerMock{}
	accountsManagerMock.AccountsMap = make(map[string]string)
	for _, walletAddress := range config.walletAddresses {
		accountsManagerMock.AccountsMap[walletAddress] = types.EncodeHex(crypto.Keccak256([]byte(config.password)))
	}

	tokenManagerMock := &TokenManagerMock{
		Balances: config.mockedBalances,
	}

	options := []Option{
		WithAccountManager(accountsManagerMock),
		WithTokenManager(tokenManagerMock),
		WithCollectiblesManager(config.collectiblesManager),
		WithCommunityTokensService(config.collectiblesService),
		WithAppSettings(*config.appSettings, *config.nodeConfig),
	}

	config.extraOptions = append(config.extraOptions, options...)

	messenger, err := newTestMessenger(waku, config.testMessengerConfig)
	s.Require().NoError(err)

	currentDistributorObj, ok := messenger.communitiesKeyDistributor.(*CommunitiesKeyDistributorImpl)
	s.Require().True(ok)
	messenger.communitiesKeyDistributor = &TestCommunitiesKeyDistributor{
		CommunitiesKeyDistributorImpl: *currentDistributorObj,
		subscriptions:                 map[chan *CommunityAndKeyActions]bool{},
		mutex:                         sync.RWMutex{},
	}

	// add wallet account with keypair
	for _, walletAddress := range config.walletAddresses {
		kp := accounts.GetProfileKeypairForTest(false, true, false)
		kp.Accounts[0].Address = types.HexToAddress(walletAddress)
		err := messenger.settings.SaveOrUpdateKeypair(kp)
		s.Require().NoError(err)
	}

	walletAccounts, err := messenger.settings.GetActiveAccounts()
	s.Require().NoError(err)
	s.Require().Len(walletAccounts, len(config.walletAddresses))
	for i := range config.walletAddresses {
		s.Require().Equal(walletAccounts[i].Type, accounts.AccountTypeGenerated)
	}
	return messenger
}

func createEncryptedCommunity(s *suite.Suite, owner *Messenger) (*communities.Community, *Chat) {
	community, chat := createCommunityConfigurable(s, owner, protobuf.CommunityPermissions_AUTO_ACCEPT)
	// Add community permission
	_, err := owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{{
			ContractAddresses: map[uint64]string{3: "0x933"},
			Type:              protobuf.CommunityTokenType_ERC20,
			Symbol:            "STT",
			Name:              "Status Test Token",
			AmountInWei:       "10000000000000000000",
			Decimals:          18,
		}},
	})
	s.Require().NoError(err)

	// Add channel permission
	response, err := owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_CAN_VIEW_CHANNEL,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				ContractAddresses: map[uint64]string{3: "0x933"},
				Type:              protobuf.CommunityTokenType_ERC20,
				Symbol:            "STT",
				Name:              "Status Test Token",
				AmountInWei:       "10000000000000000000",
				Decimals:          18,
			},
		},
		ChatIds: []string{chat.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)
	community = response.Communities()[0]
	s.Require().True(community.Encrypted())
	s.Require().True(community.ChannelEncrypted(chat.CommunityChatID()))

	return community, chat

}

func createCommunity(s *suite.Suite, owner *Messenger) (*communities.Community, *Chat) {
	return createCommunityConfigurable(s, owner, protobuf.CommunityPermissions_AUTO_ACCEPT)
}

func createOnRequestCommunity(s *suite.Suite, owner *Messenger) (*communities.Community, *Chat) {
	return createCommunityConfigurable(s, owner, protobuf.CommunityPermissions_MANUAL_ACCEPT)
}

func createCommunityConfigurable(s *suite.Suite, owner *Messenger, permission protobuf.CommunityPermissions_Access) (*communities.Community, *Chat) {
	description := &requests.CreateCommunity{
		Membership:  permission,
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
	s.Require().Len(community.Chats(), 1)

	s.Require().Len(response.CommunitiesSettings(), 1)
	communitySettings := response.CommunitiesSettings()[0]
	s.Require().Equal(communitySettings.CommunityID, community.IDString())
	s.Require().Equal(communitySettings.HistoryArchiveSupportEnabled, false)

	return community, response.Chats()[0]
}

func advertiseCommunityTo(s *suite.Suite, community *communities.Community, owner *Messenger, user *Messenger) {
	// Create wrapped (Signed) community data.
	wrappedCommunity, err := community.ToProtocolMessageBytes()
	s.Require().NoError(err)

	// Unwrap signer (Admin) data at user side.
	signer, description, err := communities.UnwrapCommunityDescriptionMessage(wrappedCommunity)
	s.Require().NoError(err)

	// Handle community data state at receiver side
	messageState := user.buildMessageState()
	messageState.CurrentMessageState = &CurrentMessageState{}
	messageState.CurrentMessageState.PublicKey = &user.identity.PublicKey
	err = user.handleCommunityDescription(messageState, signer, description, wrappedCommunity, nil, community.Shard().Protobuffer())
	s.Require().NoError(err)
}

func createRequestToJoinCommunity(s *suite.Suite, communityID types.HexBytes, user *Messenger, password string, addresses []string) *requests.RequestToJoinCommunity {
	airdropAddress := ""
	if len(addresses) > 0 {
		airdropAddress = addresses[0]
	}

	request := &requests.RequestToJoinCommunity{
		CommunityID:       communityID,
		AddressesToReveal: addresses,
		AirdropAddress:    airdropAddress}

	if password != "" {
		signingParams, err := user.GenerateJoiningCommunityRequestsForSigning(common.PubkeyToHex(&user.identity.PublicKey), communityID, request.AddressesToReveal)
		s.Require().NoError(err)

		for i := range signingParams {
			signingParams[i].Password = password
		}
		signatures, err := user.SignData(signingParams)
		s.Require().NoError(err)

		updateAddresses := len(request.AddressesToReveal) == 0
		if updateAddresses {
			request.AddressesToReveal = make([]string, len(signingParams))
		}
		for i := range signingParams {
			request.AddressesToReveal[i] = signingParams[i].Address
			request.Signatures = append(request.Signatures, types.FromHex(signatures[i]))
		}
		if updateAddresses {
			request.AirdropAddress = request.AddressesToReveal[0]
		}
	}

	return request
}

func joinCommunity(s *suite.Suite, communityID types.HexBytes, controlNode *Messenger, user *Messenger, password string, addresses []string) {
	requestToJoin := createRequestToJoinCommunity(s, communityID, user, password, addresses)
	response, err := user.RequestToJoinCommunity(requestToJoin)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)
	s.Require().Len(response.ActivityCenterNotifications(), 1)

	notification := response.ActivityCenterNotifications()[0]
	s.Require().NotNil(notification)
	s.Require().Equal(notification.Type, ActivityCenterNotificationTypeCommunityRequest)
	s.Require().Equal(notification.MembershipStatus, ActivityCenterMembershipStatusPending)

	// Retrieve and accept join request
	_, err = WaitOnMessengerResponse(controlNode, func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey)
	}, "control node did accept user request to join")
	s.Require().NoError(err)

	// Retrieve join request response
	_, err = WaitOnMessengerResponse(user, func(r *MessengerResponse) bool {
		return len(r.Communities()) > 0 && r.Communities()[0].HasMember(&user.identity.PublicKey) &&
			// Note: 'handleCommunityRequestToJoinResponse' does not return RequestToJoin with revealed addresses
			checkRequestToJoinInResponse(r, user, communities.RequestToJoinStateAccepted, 0)
	}, "user not accepted")
	s.Require().NoError(err)
}

func requestToJoinCommunity(s *suite.Suite, controlNode *Messenger, user *Messenger, request *requests.RequestToJoinCommunity) types.HexBytes {
	response, err := user.RequestToJoinCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.RequestsToJoinCommunity(), 1)

	requestToJoin := response.RequestsToJoinCommunity()[0]
	s.Require().Equal(requestToJoin.PublicKey, common.PubkeyToHex(&user.identity.PublicKey))

	_, err = WaitOnMessengerResponse(
		controlNode,
		func(r *MessengerResponse) bool {
			return checkRequestToJoinInResponse(r, user, communities.RequestToJoinStatePending, 1)
		},
		"control node did not receive community request to join",
	)
	s.Require().NoError(err)

	return requestToJoin.ID
}

func joinOnRequestCommunity(s *suite.Suite, communityID types.HexBytes, controlNode *Messenger, user *Messenger, password string, addresses []string) {
	s.Require().NotEmpty(password)
	s.Require().NotEmpty(addresses)
	s.Require().NotEmpty(communityID)
	request := createRequestToJoinCommunity(s, communityID, user, password, addresses)
	// Request to join the community
	requestToJoinID := requestToJoinCommunity(s, controlNode, user, request)

	// accept join request
	acceptRequestToJoin := &requests.AcceptRequestToJoinCommunity{ID: requestToJoinID}
	response, err := controlNode.AcceptRequestToJoinCommunity(acceptRequestToJoin)
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

	userCommunity, err := user.GetCommunityByID(communityID)
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
		subscription := user.communitiesManager.Subscribe()

		for {
			select {
			case sub, more := <-subscription:
				if !more {
					errCh <- errors.New("channel closed when waiting for communities event")
					return
				}
				if condition(sub) {
					return
				}

			case <-time.After(5 * time.Second):
				errCh <- errors.New("timed out when waiting for communities event")
				return
			}
		}
	}()

	return errCh
}

func makeAddressSatisfyTheCriteria(s *suite.Suite, mockedBalances communities.BalancesByChain, mockedCollectibles communities.CollectiblesByChain,
	chainID uint64, address string, criteria *protobuf.TokenCriteria) {

	walletAddress := gethcommon.HexToAddress(address)
	contractAddress := gethcommon.HexToAddress(criteria.ContractAddresses[chainID])

	switch criteria.Type {
	case protobuf.CommunityTokenType_ERC20:
		balance, ok := new(big.Int).SetString(criteria.AmountInWei, 10)
		s.Require().True(ok)

		if _, exists := mockedBalances[chainID]; !exists {
			mockedBalances[chainID] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
		}

		if _, exists := mockedBalances[chainID][walletAddress]; !exists {
			mockedBalances[chainID][walletAddress] = make(map[gethcommon.Address]*hexutil.Big)
		}

		mockedBalances[chainID][walletAddress][contractAddress] = (*hexutil.Big)(balance)

	case protobuf.CommunityTokenType_ERC721:
		amount, err := strconv.ParseUint(criteria.AmountInWei, 10, 32)
		s.Require().NoError(err)

		balances := []thirdparty.TokenBalance{}
		for i := uint64(0); i < amount; i++ {
			balances = append(balances, thirdparty.TokenBalance{
				TokenID: &bigint.BigInt{
					Int: new(big.Int).SetUint64(i + 1),
				},
				Balance: &bigint.BigInt{
					Int: new(big.Int).SetUint64(1),
				},
			})
		}

		if _, exists := mockedCollectibles[chainID]; !exists {
			mockedCollectibles[chainID] = make(map[gethcommon.Address]thirdparty.TokenBalancesPerContractAddress)
		}

		if _, exists := mockedCollectibles[chainID][walletAddress]; !exists {
			mockedCollectibles[chainID][walletAddress] = make(thirdparty.TokenBalancesPerContractAddress)
		}

		mockedCollectibles[chainID][walletAddress][contractAddress] = balances

	case protobuf.CommunityTokenType_ENS:
		// not implemented
	}
}

func checkRequestToJoinInResponse(r *MessengerResponse, member *Messenger, state communities.RequestToJoinState, accountsCount int) bool {
	for _, request := range r.RequestsToJoinCommunity() {
		if request.PublicKey == member.IdentityPublicKeyString() &&
			request.State == state &&
			accountsCount == len(request.RevealedAccounts) {
			return true
		}
	}
	return false
}
