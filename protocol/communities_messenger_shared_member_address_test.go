package protocol

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	hexutil "github.com/ethereum/go-ethereum/common/hexutil"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/tt"
)

func TestMessengerCommunitiesSharedMemberAddressSuite(t *testing.T) {
	suite.Run(t, new(MessengerCommunitiesSharedMemberAddressSuite))
}

type MessengerCommunitiesSharedMemberAddressSuite struct {
	suite.Suite
	owner *Messenger
	bob   *Messenger
	alice *Messenger

	ownerWaku types.Waku
	bobWaku   types.Waku
	aliceWaku types.Waku

	logger *zap.Logger

	mockedBalances          map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big // chainID, account, token, balance
	collectiblesServiceMock *CollectiblesServiceMock
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) SetupTest() {
	// Initialize with nil to avoid panics in TearDownTest
	s.owner = nil
	s.bob = nil
	s.alice = nil
	s.ownerWaku = nil
	s.bobWaku = nil
	s.aliceWaku = nil

	communities.SetValidateInterval(300 * time.Millisecond)
	s.collectiblesServiceMock = &CollectiblesServiceMock{}

	s.resetMockedBalances()

	s.logger = tt.MustCreateTestLogger()

	wakuNodes := CreateWakuV2Network(&s.Suite, s.logger, []string{"owner", "bob", "alice"})

	s.ownerWaku = wakuNodes[0]
	s.owner = s.newMessenger(ownerPassword, []string{ownerAddress}, s.ownerWaku, "owner", []Option{})

	s.bobWaku = wakuNodes[1]
	s.bob = s.newMessenger(bobPassword, []string{bobAddress}, s.bobWaku, "bob", []Option{})
	s.bob.EnableBackedupMessagesProcessing()

	s.aliceWaku = wakuNodes[2]
	s.alice = s.newMessenger(alicePassword, []string{aliceAddress1, aliceAddress2}, s.aliceWaku, "alice", []Option{})

	_, err := s.owner.Start()
	s.Require().NoError(err)
	_, err = s.bob.Start()
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TearDownTest() {
	TearDownMessenger(&s.Suite, s.owner)
	TearDownMessenger(&s.Suite, s.bob)
	TearDownMessenger(&s.Suite, s.alice)
	if s.ownerWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.ownerWaku).Stop())
	}
	if s.bobWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.bobWaku).Stop())
	}
	if s.aliceWaku != nil {
		s.Require().NoError(gethbridge.GetGethWakuV2From(s.aliceWaku).Stop())
	}
	_ = s.logger.Sync()
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) newMessenger(password string, walletAddresses []string, waku types.Waku, name string, extraOptions []Option) *Messenger {
	communityManagerOptions := []communities.ManagerOption{
		communities.WithAllowForcingCommunityMembersReevaluation(true),
	}
	extraOptions = append(extraOptions, WithCommunityManagerOptions(communityManagerOptions))

	return newTestCommunitiesMessenger(&s.Suite, waku, testCommunitiesMessengerConfig{
		testMessengerConfig: testMessengerConfig{
			logger:       s.logger.Named(name),
			extraOptions: extraOptions,
		},
		password:            password,
		walletAddresses:     walletAddresses,
		mockedBalances:      &s.mockedBalances,
		collectiblesService: s.collectiblesServiceMock,
	})
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) joinCommunity(community *communities.Community, user *Messenger, password string, addresses []string) {
	s.joinCommunityWithAirdropAddress(community, user, password, addresses, "")
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) joinCommunityWithAirdropAddress(community *communities.Community, user *Messenger, password string, addresses []string, airdropAddress string) {
	passwdHash := types.EncodeHex(crypto.Keccak256([]byte(password)))
	if airdropAddress == "" && len(addresses) > 0 {
		airdropAddress = addresses[0]
	}

	request := &requests.RequestToJoinCommunity{CommunityID: community.ID(), AddressesToReveal: addresses, AirdropAddress: airdropAddress}
	joinCommunity(&s.Suite, community, s.owner, user, request, passwdHash)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) checkRevealedAccounts(communityID types.HexBytes, user *Messenger, expectedAccounts []*protobuf.RevealedAccount) {
	revealedAccounts, err := user.communitiesManager.GetRevealedAddresses(communityID, s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Equal(revealedAccounts, expectedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) makeAddressSatisfyTheCriteria(chainID uint64, address string, criteria *protobuf.TokenCriteria) {
	walletAddress := gethcommon.HexToAddress(address)
	contractAddress := gethcommon.HexToAddress(criteria.ContractAddresses[chainID])
	balance, ok := new(big.Int).SetString(criteria.AmountInWei, 10)
	s.Require().True(ok)

	s.mockedBalances[chainID][walletAddress][contractAddress] = (*hexutil.Big)(balance)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) resetMockedBalances() {
	s.mockedBalances = make(map[uint64]map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1] = make(map[gethcommon.Address]map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(aliceAddress1)] = make(map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(aliceAddress2)] = make(map[gethcommon.Address]*hexutil.Big)
	s.mockedBalances[testChainID1][gethcommon.HexToAddress(bobAddress)] = make(map[gethcommon.Address]*hexutil.Big)
}

func createTokenMasterTokenCriteria() *protobuf.TokenCriteria {
	return &protobuf.TokenCriteria{
		ContractAddresses: map[uint64]string{testChainID1: "0x123"},
		Type:              protobuf.CommunityTokenType_ERC20,
		Symbol:            "STT",
		Name:              "Status Test Token",
		AmountInWei:       "10000000000000000000",
		Decimals:          18,
	}
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) createEditSharedAddressesRequest(communityID types.HexBytes) *requests.EditSharedAddresses {
	request := &requests.EditSharedAddresses{CommunityID: communityID, AddressesToReveal: []string{aliceAddress2}, AirdropAddress: aliceAddress2}

	signingParams, err := s.alice.GenerateJoiningCommunityRequestsForSigning(common.PubkeyToHex(&s.alice.identity.PublicKey), communityID, request.AddressesToReveal)
	s.Require().NoError(err)

	passwdHash := types.EncodeHex(crypto.Keccak256([]byte(alicePassword)))
	for i := range signingParams {
		signingParams[i].Password = passwdHash
	}
	signatures, err := s.alice.SignData(signingParams)
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

	return request
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestJoinedCommunityMembersSharedAddress() {
	community, _ := createCommunity(&s.Suite, s.owner)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)

	s.joinCommunity(community, s.alice, alicePassword, []string{})
	s.joinCommunity(community, s.bob, bobPassword, []string{})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(3, community.MembersCount())

	// Check owner's DB for revealed accounts
	for pubKey := range community.Members() {
		if pubKey != common.PubkeyToHex(&s.owner.identity.PublicKey) {
			revealedAccounts, err := s.owner.communitiesManager.GetRevealedAddresses(community.ID(), pubKey)
			s.Require().NoError(err)
			switch pubKey {
			case common.PubkeyToHex(&s.alice.identity.PublicKey):
				s.Require().Len(revealedAccounts, 2)
				s.Require().Equal(revealedAccounts[0].Address, aliceAddress1)
				s.Require().Equal(revealedAccounts[1].Address, aliceAddress2)
				s.Require().Equal(true, revealedAccounts[0].IsAirdropAddress)
			case common.PubkeyToHex(&s.bob.identity.PublicKey):
				s.Require().Len(revealedAccounts, 1)
				s.Require().Equal(revealedAccounts[0].Address, bobAddress)
				s.Require().Equal(true, revealedAccounts[0].IsAirdropAddress)
			default:
				s.Require().Fail("pubKey does not match expected keys")
			}
		}
	}

	// Check Bob's DB for revealed accounts
	revealedAccountsInBobsDB, err := s.bob.communitiesManager.GetRevealedAddresses(community.ID(), common.PubkeyToHex(&s.bob.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().Len(revealedAccountsInBobsDB, 1)
	s.Require().Equal(revealedAccountsInBobsDB[0].Address, bobAddress)
	s.Require().Equal(true, revealedAccountsInBobsDB[0].IsAirdropAddress)

	// Check Alices's DB for revealed accounts
	revealedAccountsInAlicesDB, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().NoError(err)
	s.Require().Len(revealedAccountsInAlicesDB, 2)
	s.Require().Equal(revealedAccountsInAlicesDB[0].Address, aliceAddress1)
	s.Require().Equal(revealedAccountsInAlicesDB[1].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccountsInAlicesDB[0].IsAirdropAddress)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestJoinedCommunityMembersSelectedSharedAddress() {
	community, _ := createCommunity(&s.Suite, s.owner)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)

	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress2})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(2, community.MembersCount())

	alicePubkey := common.PubkeyToHex(&s.alice.identity.PublicKey)

	// Check Alice's DB for revealed accounts
	revealedAccountsInAlicesDB, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(revealedAccountsInAlicesDB, 1)
	s.Require().Equal(revealedAccountsInAlicesDB[0].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccountsInAlicesDB[0].IsAirdropAddress)

	// Check owner's DB for revealed accounts
	s.checkRevealedAccounts(community.ID(), s.owner, revealedAccountsInAlicesDB)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestJoinedCommunityMembersMultipleSelectedSharedAddresses() {
	community, _ := createCommunity(&s.Suite, s.owner)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)

	s.joinCommunityWithAirdropAddress(community, s.alice, alicePassword, []string{aliceAddress1, aliceAddress2}, aliceAddress2)

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	s.Require().Equal(2, community.MembersCount())

	alicePubkey := common.PubkeyToHex(&s.alice.identity.PublicKey)

	// Check Alice's DB for revealed accounts
	revealedAccountsInAlicesDB, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(revealedAccountsInAlicesDB, 2)
	s.Require().Equal(revealedAccountsInAlicesDB[0].Address, aliceAddress1)
	s.Require().Equal(revealedAccountsInAlicesDB[1].Address, aliceAddress2)
	s.Require().Equal(true, revealedAccountsInAlicesDB[1].IsAirdropAddress)

	// Check owner's DB for revealed accounts
	s.checkRevealedAccounts(community.ID(), s.owner, revealedAccountsInAlicesDB)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestEditSharedAddresses() {
	community, _ := createCommunity(&s.Suite, s.owner)
	alicePubkey := common.PubkeyToHex(&s.alice.identity.PublicKey)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	community, err := s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)
	s.Require().Equal(2, community.MembersCount())

	aliceExpectedRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(aliceExpectedRevealedAccounts, 1)
	s.Require().Equal(aliceExpectedRevealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, aliceExpectedRevealedAccounts[0].IsAirdropAddress)

	s.checkRevealedAccounts(community.ID(), s.owner, aliceExpectedRevealedAccounts)

	request := s.createEditSharedAddressesRequest(community.ID())

	response, err := s.alice.EditSharedAddressesForCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	aliceExpectedRevealedAccounts, err = s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(aliceExpectedRevealedAccounts, 1)
	s.Require().Equal(aliceExpectedRevealedAccounts[0].Address, aliceAddress2)
	s.Require().Equal(true, aliceExpectedRevealedAccounts[0].IsAirdropAddress)

	// check that owner received revealed address
	_, err = WaitOnMessengerResponse(s.owner, func(r *MessengerResponse) bool {
		revealedAccounts, err := s.owner.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
		s.Require().NoError(err)
		s.Require().Len(revealedAccounts, 1)
		return revealedAccounts[0].Address == aliceAddress2

	}, "owned did not receive alice shared address")
	s.Require().NoError(err)

	s.checkRevealedAccounts(community.ID(), s.owner, aliceExpectedRevealedAccounts)

	// check that we filter out outdated edit shared addresses events
	community, err = s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	aliceClock := community.Description().Members[s.alice.IdentityPublicKeyString()].LastUpdateClock
	s.Require().Greater(aliceClock, uint64(1))

	editMsg := &protobuf.CommunityEditSharedAddresses{
		Clock:            aliceClock - 1,
		CommunityId:      community.ID(),
		RevealedAccounts: aliceExpectedRevealedAccounts,
	}

	state := &ReceivedMessageState{
		CurrentMessageState: &CurrentMessageState{
			PublicKey: s.alice.IdentityPublicKey(),
		},
	}

	err = s.owner.HandleCommunityEditSharedAddresses(state, editMsg, nil)
	s.Require().Error(err, communities.ErrEditSharedAddressesRequestOutdated)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestTokenMasterReceivesEditedSharedAddresses() {
	community, _ := createCommunity(&s.Suite, s.owner)

	alicePubkey := s.alice.IdentityPublicKeyString()

	tokenCriteria := createTokenMasterTokenCriteria()

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	// check bob has TM role
	community, err = s.bob.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	checkRoleBasedOnThePermissionType(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, &s.bob.identity.PublicKey, community)

	aliceRevealedAccounts, err := s.bob.GetRevealedAccounts(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(aliceRevealedAccounts, 1)

	request := s.createEditSharedAddressesRequest(community.ID())

	response, err := s.alice.EditSharedAddressesForCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	expectedAliceRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress2)
	s.Require().Equal(true, expectedAliceRevealedAccounts[0].IsAirdropAddress)

	// check that owner received revealed address
	_, err = WaitOnMessengerResponse(s.owner, func(r *MessengerResponse) bool {
		revealedAccounts, err := s.owner.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
		s.Require().NoError(err)
		s.Require().Len(revealedAccounts, 1)
		return revealedAccounts[0].Address == aliceAddress2

	}, "owned did not receive alice shared address")
	s.Require().NoError(err)

	s.checkRevealedAccounts(community.ID(), s.owner, expectedAliceRevealedAccounts)

	// check that bob as a token master received revealed address
	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		revealedAccounts, err := s.bob.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
		s.Require().NoError(err)
		s.Require().Len(revealedAccounts, 1)
		return revealedAccounts[0].Address == aliceAddress2
	}, "user not accepted")
	s.Require().NoError(err)
	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestSharedAddressesReturnsRevealedAccount() {
	community, _ := createCommunity(&s.Suite, s.owner)

	permissionRequest := requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL,
		TokenCriteria: []*protobuf.TokenCriteria{
			&protobuf.TokenCriteria{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x123"},
				Symbol:            "TEST",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	response, err := s.owner.CreateCommunityTokenPermission(&permissionRequest)
	s.Require().NoError(err)
	s.Require().Len(response.Communities(), 1)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)

	s.joinCommunity(community, s.alice, alicePassword, []string{})

	revealedAccounts, err := s.alice.GetRevealedAccounts(community.ID(), common.PubkeyToHex(&s.alice.identity.PublicKey))
	s.Require().NoError(err)

	revealedAddressesMap := make(map[string]struct{}, len(revealedAccounts))
	for _, acc := range revealedAccounts {
		revealedAddressesMap[acc.Address] = struct{}{}
	}

	s.Require().Len(revealedAddressesMap, 2)
	s.Require().Contains(revealedAddressesMap, aliceAddress1)
	s.Require().Contains(revealedAddressesMap, aliceAddress2)

	sharedAddresses, err := s.alice.getSharedAddresses(community.ID(), []string{})
	s.Require().NoError(err)
	s.Require().Len(sharedAddresses, 2)

	sharedAddressesMap := make(map[string]struct{}, len(sharedAddresses))
	for _, acc := range sharedAddresses {
		sharedAddressesMap[acc.String()] = struct{}{}
	}

	s.Require().Len(sharedAddressesMap, 2)
	s.Require().Contains(sharedAddressesMap, aliceAddress1)
	s.Require().Contains(sharedAddressesMap, aliceAddress2)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestResendSharedAddressesOnBackupRestore() {
	community, _ := createCommunity(&s.Suite, s.owner)

	// bob joins the community
	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	currentBobSharedAddresses, err := s.bob.GetRevealedAccounts(community.ID(), s.bob.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(currentBobSharedAddresses, 1)

	requestID := communities.CalculateRequestID(s.bob.IdentityPublicKeyString(), community.ID())
	err = s.bob.communitiesManager.RemoveRequestToJoinRevealedAddresses(requestID)
	s.Require().NoError(err)

	emptySharedAddresses, err := s.bob.GetRevealedAccounts(community.ID(), s.bob.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(emptySharedAddresses, 0)

	// Simulate backup creation and handling backup message
	// As a result, bob sends request to resend encryption keys to the owner
	clock, _ := s.bob.getLastClockWithRelatedChat()

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	backupMessage, err := s.bob.backupCommunity(community, clock)
	s.Require().NoError(err)

	err = s.bob.HandleBackup(s.bob.buildMessageState(), backupMessage, nil)
	s.Require().NoError(err)

	// Owner will receive the request for addresses and send them back to Bob
	response, err := WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			_, _ = s.owner.RetrieveAll()
			return len(r.requestsToJoinCommunity) > 0
		},
		"request to join not received",
	)
	s.Require().NoError(err)

	requestToJoin, ok := response.requestsToJoinCommunity[requestID.String()]
	s.Require().Equal(true, ok)
	s.Require().Equal(currentBobSharedAddresses, requestToJoin.RevealedAccounts)

	currentBobSharedAddresses, err = s.bob.GetRevealedAccounts(community.ID(), s.bob.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(currentBobSharedAddresses, 1)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestTokenMasterReceivesMembersSharedAddressesOnBackupRestore() {
	community, _ := createCommunity(&s.Suite, s.owner)

	tokenCriteria := createTokenMasterTokenCriteria()

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	expectedAliceRevealedAccounts, err := s.alice.GetRevealedAccounts(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress1)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	// check bob has TM role
	community, err = s.bob.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	checkRoleBasedOnThePermissionType(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, &s.bob.identity.PublicKey, community)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)

	// remove alice revealed addresses
	aliceRequestID := communities.CalculateRequestID(s.alice.IdentityPublicKeyString(), community.ID())
	err = s.bob.communitiesManager.RemoveRequestToJoinRevealedAddresses(aliceRequestID)
	s.Require().NoError(err)

	emptySharedAddresses, err := s.bob.GetRevealedAccounts(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(emptySharedAddresses, 0)
	s.Require().NotEqual(emptySharedAddresses, expectedAliceRevealedAccounts)

	// Simulate backup creation and handling backup message
	// As a result, bob sends request to resend encryption keys to the owner
	clock, _ := s.bob.getLastClockWithRelatedChat()

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	backupMessage, err := s.bob.backupCommunity(community, clock)
	s.Require().NoError(err)

	err = s.bob.HandleBackup(s.bob.buildMessageState(), backupMessage, nil)
	s.Require().NoError(err)

	// Owner will receive the request for addresses and send requests to join with revealed
	// addresses to token master
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			_, _ = s.owner.RetrieveAll()
			if len(r.requestsToJoinCommunity) == 0 {
				return false
			}

			for _, requestToJoin := range r.requestsToJoinCommunity {
				if requestToJoin.PublicKey == s.alice.IdentityPublicKeyString() {
					return true
				}
			}

			return false
		},
		"alice request to join with revealed addresses not received",
	)
	s.Require().NoError(err)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestTokenMasterReceivedRevealedAddressesFromJoinedMember() {
	community, _ := createCommunity(&s.Suite, s.owner)

	tokenCriteria := createTokenMasterTokenCriteria()

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 1)

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	// check bob has TM role
	community, err = s.bob.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	checkRoleBasedOnThePermissionType(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, &s.bob.identity.PublicKey, community)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	// check that bob received revealed address
	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.RequestsToJoinCommunity()) == 1 && r.RequestsToJoinCommunity()[0].PublicKey == s.alice.IdentityPublicKeyString()
	}, "user not accepted")
	s.Require().NoError(err)

	expectedAliceRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, expectedAliceRevealedAccounts[0].IsAirdropAddress)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestTokenMasterJoinedToCommunityAndReceivedRevealedAddresses() {
	community, _ := createCommunity(&s.Suite, s.owner)

	tokenCriteria := createTokenMasterTokenCriteria()

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 1)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	community, err = s.bob.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	checkRoleBasedOnThePermissionType(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, &s.bob.identity.PublicKey, community)

	expectedAliceRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, expectedAliceRevealedAccounts[0].IsAirdropAddress)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestMemberReceivedSharedAddressOnGettingTokenMasterRole() {
	community, _ := createCommunity(&s.Suite, s.owner)

	community, err := s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 0)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	tokenCriteria := createTokenMasterTokenCriteria()

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	// wait for owner to send sync message for bob, who got a TM role
	waitOnOwnerSendSyncMessage := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return sub.CommunityPrivilegedMemberSyncMessage != nil &&
			sub.CommunityPrivilegedMemberSyncMessage.CommunityPrivilegedUserSyncMessage.Type == protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN &&
			len(sub.CommunityPrivilegedMemberSyncMessage.Receivers) == 1 &&
			sub.CommunityPrivilegedMemberSyncMessage.Receivers[0].Equal(&s.bob.identity.PublicKey)
	})

	_, err = s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	err = <-waitOnOwnerSendSyncMessage
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.Communities()) == 1 && r.Communities()[0].IsTokenMaster()
	}, "bob didn't receive token master role")
	s.Require().NoError(err)

	expectedAliceRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, expectedAliceRevealedAccounts[0].IsAirdropAddress)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestTokenMasterReceivesAccountsAfterPendingRequestToJoinApproval() {
	community, _ := createOnRequestCommunity(&s.Suite, s.owner)

	tokenCriteria := createTokenMasterTokenCriteria()

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 1)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)

	aliceArray64Bytes := common.HashPublicKey(&s.alice.identity.PublicKey)
	aliceSignature := append([]byte{0}, aliceArray64Bytes...)
	aliceRequest := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{aliceAddress1},
		AirdropAddress:    aliceAddress1,
		Signatures:        []types.HexBytes{aliceSignature},
	}

	aliceRequestToJoinID := requestToJoinCommunity(&s.Suite, s.owner, s.alice, aliceRequest)

	bobArray64Bytes := common.HashPublicKey(&s.bob.identity.PublicKey)
	bobSignature := append([]byte{0}, bobArray64Bytes...)
	bobRequest := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{bobAddress},
		AirdropAddress:    bobAddress,
		Signatures:        []types.HexBytes{bobSignature},
	}

	joinOnRequestCommunity(&s.Suite, community, s.owner, s.bob, bobRequest)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.IsMemberTokenMaster(&s.bob.identity.PublicKey))

	_, err = s.owner.AcceptRequestToJoinCommunity(&requests.AcceptRequestToJoinCommunity{ID: aliceRequestToJoinID})
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		if r.RequestsToJoinCommunity() != nil {
			for _, request := range r.RequestsToJoinCommunity() {
				if request.PublicKey == s.alice.IdentityPublicKeyString() && request.State == communities.RequestToJoinStateAccepted {
					return true
				}
			}
		}
		return false
	}, "bob didn't receive accepted Alice request to join")
	s.Require().NoError(err)

	expectedAliceRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, expectedAliceRevealedAccounts[0].IsAirdropAddress)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestMemberReceivesPendingRequestToJoinAfterAfterGettingTokenMasterRole() {
	community, _ := createOnRequestCommunity(&s.Suite, s.owner)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)

	aliceArray64Bytes := common.HashPublicKey(&s.alice.identity.PublicKey)
	aliceSignature := append([]byte{0}, aliceArray64Bytes...)
	aliceRequest := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{aliceAddress1},
		AirdropAddress:    aliceAddress1,
		Signatures:        []types.HexBytes{aliceSignature},
	}

	aliceRequestToJoinID := requestToJoinCommunity(&s.Suite, s.owner, s.alice, aliceRequest)

	bobArray64Bytes := common.HashPublicKey(&s.bob.identity.PublicKey)
	bobSignature := append([]byte{0}, bobArray64Bytes...)
	bobRequest := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{bobAddress},
		AirdropAddress:    bobAddress,
		Signatures:        []types.HexBytes{bobSignature},
	}

	joinOnRequestCommunity(&s.Suite, community, s.owner, s.bob, bobRequest)

	tokenCriteria := createTokenMasterTokenCriteria()

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	// wait for owner to send sync message for bob, who got a TM role
	waitOnOwnerSendSyncMessage := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return sub.CommunityPrivilegedMemberSyncMessage != nil &&
			sub.CommunityPrivilegedMemberSyncMessage.CommunityPrivilegedUserSyncMessage.Type == protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN &&
			len(sub.CommunityPrivilegedMemberSyncMessage.Receivers) == 1 &&
			sub.CommunityPrivilegedMemberSyncMessage.Receivers[0].Equal(&s.bob.identity.PublicKey)
	})

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	err = <-waitOnOwnerSendSyncMessage
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.Communities()) == 1 && r.Communities()[0].IsTokenMaster()
	}, "bob didn't receive token master role")
	s.Require().NoError(err)

	_, err = s.owner.AcceptRequestToJoinCommunity(&requests.AcceptRequestToJoinCommunity{ID: aliceRequestToJoinID})
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.RequestsToJoinCommunity()) > 0 &&
			r.RequestsToJoinCommunity()[0].PublicKey == s.alice.IdentityPublicKeyString() &&
			r.RequestsToJoinCommunity()[0].State == communities.RequestToJoinStateAccepted

	}, "bob didn't receive accepted Alice request to join")
	s.Require().NoError(err)

	expectedAliceRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, expectedAliceRevealedAccounts[0].IsAirdropAddress)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestHandlingOutdatedPrivilegedUserSyncMessages() {
	community, _ := createCommunity(&s.Suite, s.owner)

	tokenCriteria := createTokenMasterTokenCriteria()

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 1)

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	// check bob has TM role
	community, err = s.bob.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	checkRoleBasedOnThePermissionType(protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER, &s.bob.identity.PublicKey, community)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	// check that bob received revealed address
	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.RequestsToJoinCommunity()) == 1 && r.RequestsToJoinCommunity()[0].PublicKey == s.alice.IdentityPublicKeyString()
	}, "user not accepted")
	s.Require().NoError(err)

	// handle outdated CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN msg
	expectedAliceRequestToJoin, err := s.bob.communitiesManager.GetRequestToJoinByPkAndCommunityID(s.alice.IdentityPublicKey(), community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(expectedAliceRequestToJoin)

	bobRequestToJoin, err := s.bob.communitiesManager.GetRequestToJoinByPkAndCommunityID(s.bob.IdentityPublicKey(), community.ID())
	s.Require().NoError(err)
	s.Require().NotNil(bobRequestToJoin)

	invalidAliceSyncRtj := expectedAliceRequestToJoin.ToSyncProtobuf()
	invalidAliceSyncRtj.RevealedAccounts = bobRequestToJoin.RevealedAccounts
	invalidAliceSyncRtj.EnsName = "corrupted"
	invalidAliceSyncRtj.State = uint64(communities.RequestToJoinStatePending)

	syncMsg := &protobuf.CommunityPrivilegedUserSyncMessage{
		Type:               protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN,
		CommunityId:        community.ID(),
		SyncRequestsToJoin: []*protobuf.SyncCommunityRequestsToJoin{invalidAliceSyncRtj},
	}

	state := &ReceivedMessageState{
		CurrentMessageState: &CurrentMessageState{
			PublicKey: community.PublicKey(),
		},
	}

	err = s.bob.HandleCommunityPrivilegedUserSyncMessage(state, syncMsg, nil)
	s.Require().NoError(err)

	aliceRtj, err := s.bob.communitiesManager.GetRequestToJoinByPkAndCommunityID(s.alice.IdentityPublicKey(), community.ID())
	s.Require().NoError(err)
	s.Require().Equal(aliceRtj, expectedAliceRequestToJoin)

	// handle outdated CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ACCEPT_REQUEST_TO_JOIN msg

	invalidAliceCommunityRtj := aliceRtj.ToCommunityRequestToJoinProtobuf()
	invalidAliceCommunityRtj.RevealedAccounts = bobRequestToJoin.RevealedAccounts
	invalidAliceCommunityRtj.EnsName = "corrupted"

	syncMsg.Type = protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ACCEPT_REQUEST_TO_JOIN
	syncMsg.RequestToJoin = map[string]*protobuf.CommunityRequestToJoin{
		s.alice.IdentityPublicKeyString(): invalidAliceCommunityRtj,
	}

	err = s.bob.HandleCommunityPrivilegedUserSyncMessage(state, syncMsg, nil)
	s.Require().NoError(err)

	aliceRtj, err = s.bob.communitiesManager.GetRequestToJoinByPkAndCommunityID(s.alice.IdentityPublicKey(), community.ID())
	s.Require().NoError(err)
	s.Require().Equal(aliceRtj, expectedAliceRequestToJoin)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestMemberReceivedEditedSharedAddressOnGettingTokenMasterRole() {
	community, _ := createCommunity(&s.Suite, s.owner)

	alicePubkey := s.alice.IdentityPublicKeyString()

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	request := s.createEditSharedAddressesRequest(community.ID())

	response, err := s.alice.EditSharedAddressesForCommunity(request)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	expectedAliceRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress2)
	s.Require().Equal(true, expectedAliceRevealedAccounts[0].IsAirdropAddress)

	// check that owner received edited shared adresses
	_, err = WaitOnMessengerResponse(s.owner, func(r *MessengerResponse) bool {
		revealedAccounts, err := s.owner.communitiesManager.GetRevealedAddresses(community.ID(), alicePubkey)
		s.Require().NoError(err)
		s.Require().Len(revealedAccounts, 1)
		return revealedAccounts[0].Address == aliceAddress2

	}, "owned did not receive alice shared address")
	s.Require().NoError(err)

	s.checkRevealedAccounts(community.ID(), s.owner, expectedAliceRevealedAccounts)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	tokenCriteria := createTokenMasterTokenCriteria()

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	// wait for owner to send sync message for bob, who got a TM role
	waitOnOwnerSendSyncMessage := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return sub.CommunityPrivilegedMemberSyncMessage != nil &&
			sub.CommunityPrivilegedMemberSyncMessage.CommunityPrivilegedUserSyncMessage.Type == protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN &&
			len(sub.CommunityPrivilegedMemberSyncMessage.Receivers) == 1 &&
			sub.CommunityPrivilegedMemberSyncMessage.Receivers[0].Equal(&s.bob.identity.PublicKey)
	})

	_, err = s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	err = <-waitOnOwnerSendSyncMessage
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.Communities()) == 1 && r.Communities()[0].IsTokenMaster()
	}, "bob didn't receive token master role")
	s.Require().NoError(err)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestMemberReceivesAccountsOnRoleChangeFromAdminToTokenMaster() {
	community, _ := createCommunity(&s.Suite, s.owner)

	alicePublicKey := s.alice.IdentityPublicKeyString()

	adminTokenCriteria := &protobuf.TokenCriteria{
		ContractAddresses: map[uint64]string{testChainID1: "0x125"},
		Type:              protobuf.CommunityTokenType_ERC20,
		Symbol:            "STT",
		Name:              "Status Test Token",
		AmountInWei:       "10000000000000000000",
		Decimals:          18,
	}
	tokenMasterTokenCriteria := createTokenMasterTokenCriteria()

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenMasterTokenCriteria},
	})
	s.Require().NoError(err)

	_, err = s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_ADMIN,
		TokenCriteria: []*protobuf.TokenCriteria{adminTokenCriteria},
	})
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 2)

	// make bob satisfy the admin criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, adminTokenCriteria)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)
	s.joinCommunity(community, s.bob, bobPassword, []string{bobAddress})

	// check bob has admin role
	community, err = s.bob.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	checkRoleBasedOnThePermissionType(protobuf.CommunityTokenPermission_BECOME_ADMIN, &s.bob.identity.PublicKey, community)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)
	s.joinCommunity(community, s.alice, alicePassword, []string{aliceAddress1})

	expectedAliceRevealedAccounts, err := s.alice.communitiesManager.GetRevealedAddresses(community.ID(), alicePublicKey)
	s.Require().NoError(err)
	s.Require().Len(expectedAliceRevealedAccounts, 1)
	s.Require().Equal(expectedAliceRevealedAccounts[0].Address, aliceAddress1)
	s.Require().Equal(true, expectedAliceRevealedAccounts[0].IsAirdropAddress)

	// check that bob received alice request to join without revealed accounts
	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.RequestsToJoinCommunity()) == 1 &&
			r.RequestsToJoinCommunity()[0].PublicKey == s.alice.IdentityPublicKeyString() &&
			len(r.RequestsToJoinCommunity()[0].RevealedAccounts) == 0
	}, "alice request to join was not delivered to admin bob")
	s.Require().NoError(err)

	emptyAliceAccounts, err := s.bob.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(emptyAliceAccounts, 0)

	// make bob satisfy TokenMaster criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenMasterTokenCriteria)

	// wait for owner to send sync message for bob, who got a TM role
	waitOnOwnerSendSyncMessage := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return sub.CommunityPrivilegedMemberSyncMessage != nil &&
			sub.CommunityPrivilegedMemberSyncMessage.CommunityPrivilegedUserSyncMessage.Type == protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN &&
			len(sub.CommunityPrivilegedMemberSyncMessage.Receivers) == 1 &&
			sub.CommunityPrivilegedMemberSyncMessage.Receivers[0].Equal(&s.bob.identity.PublicKey)
	})

	err = s.owner.communitiesManager.ForceMembersReevaluation(community.ID())
	s.Require().NoError(err)

	err = <-waitOnOwnerSendSyncMessage
	s.Require().NoError(err)

	// check that bob received alice request to join with revealed accounts
	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		revealedAccounts, err := s.bob.communitiesManager.GetRevealedAddresses(community.ID(), alicePublicKey)
		s.Require().NoError(err)
		return len(revealedAccounts) > 0
	}, "alice request to join was not delivered to token master bob")
	s.Require().NoError(err)

	s.checkRevealedAccounts(community.ID(), s.bob, expectedAliceRevealedAccounts)
}

func (s *MessengerCommunitiesSharedMemberAddressSuite) TestOwnerRejectAndAcceptAliceRequestToJoin() {
	s.T().Skip("flaky test")

	community, _ := createOnRequestCommunity(&s.Suite, s.owner)
	s.Require().False(community.AutoAccept())

	tokenCriteria := createTokenMasterTokenCriteria()

	// make bob satisfy the Token Master criteria
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, tokenCriteria)

	_, err := s.owner.CreateCommunityTokenPermission(&requests.CreateCommunityTokenPermission{
		CommunityID:   community.ID(),
		Type:          protobuf.CommunityTokenPermission_BECOME_TOKEN_MASTER,
		TokenCriteria: []*protobuf.TokenCriteria{tokenCriteria},
	})
	s.Require().NoError(err)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().Len(community.TokenPermissions(), 1)

	advertiseCommunityTo(&s.Suite, community, s.owner, s.bob)

	aliceArray64Bytes := common.HashPublicKey(&s.alice.identity.PublicKey)
	aliceSignature := append([]byte{0}, aliceArray64Bytes...)
	aliceRequest := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{aliceAddress1},
		AirdropAddress:    aliceAddress1,
		Signatures:        []types.HexBytes{aliceSignature},
	}

	bobArray64Bytes := common.HashPublicKey(&s.bob.identity.PublicKey)
	bobSignature := append([]byte{0}, bobArray64Bytes...)
	bobRequest := &requests.RequestToJoinCommunity{
		CommunityID:       community.ID(),
		AddressesToReveal: []string{bobAddress},
		AirdropAddress:    bobAddress,
		Signatures:        []types.HexBytes{bobSignature},
	}

	joinOnRequestCommunity(&s.Suite, community, s.owner, s.bob, bobRequest)

	community, err = s.owner.communitiesManager.GetByID(community.ID())
	s.Require().NoError(err)
	s.Require().True(community.IsMemberTokenMaster(&s.bob.identity.PublicKey))

	advertiseCommunityTo(&s.Suite, community, s.owner, s.alice)

	aliceRequestToJoinID := requestToJoinCommunity(&s.Suite, s.owner, s.alice, aliceRequest)

	// check that bob received alice request to join without revealed accounts due to pending state
	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.RequestsToJoinCommunity()) == 1 &&
			r.RequestsToJoinCommunity()[0].State == communities.RequestToJoinStatePending &&
			r.RequestsToJoinCommunity()[0].PublicKey == s.alice.IdentityPublicKeyString()
	}, "alice pending request to join was not delivered to token master bob")
	s.Require().NoError(err)

	// request to join was not approved, bob should not have alice revealed addresses
	aliceRevealedAccounts, err := s.bob.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(aliceRevealedAccounts, 0)

	_, err = s.owner.DeclineRequestToJoinCommunity(&requests.DeclineRequestToJoinCommunity{ID: aliceRequestToJoinID})
	s.Require().NoError(err)

	// check that bob received owner decline sync msg
	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.RequestsToJoinCommunity()) == 1 &&
			r.RequestsToJoinCommunity()[0].State == communities.RequestToJoinStateDeclined &&
			r.RequestsToJoinCommunity()[0].PublicKey == s.alice.IdentityPublicKeyString()
	}, "alice declined request to join was not delivered to token master bob")
	s.Require().NoError(err)

	// request to join was declined, bob should not have alice revealed addresses
	aliceRevealedAccounts, err = s.bob.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(aliceRevealedAccounts, 0)

	_, err = s.owner.AcceptRequestToJoinCommunity(&requests.AcceptRequestToJoinCommunity{ID: aliceRequestToJoinID})
	s.Require().NoError(err)

	_, err = WaitOnMessengerResponse(s.bob, func(r *MessengerResponse) bool {
		return len(r.RequestsToJoinCommunity()) == 1 &&
			r.RequestsToJoinCommunity()[0].State == communities.RequestToJoinStateAccepted &&
			r.RequestsToJoinCommunity()[0].PublicKey == s.alice.IdentityPublicKeyString()
	}, "bob didn't receive accepted Alice request to join")
	s.Require().NoError(err)

	// request to join was accepted, bob should have alice revealed addresses
	aliceRevealedAccounts, err = s.bob.communitiesManager.GetRevealedAddresses(community.ID(), s.alice.IdentityPublicKeyString())
	s.Require().NoError(err)
	s.Require().Len(aliceRevealedAccounts, 1)
}
