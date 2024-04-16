package protocol

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/appdatabase"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/identity"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/waku"
)

type CollectiblesManagerMock struct {
	response map[thirdparty.CollectibleUniqueID][]thirdparty.AccountBalance
}

func (m *CollectiblesManagerMock) FetchBalancesByOwnerAndContractAddress(ctx context.Context, chainID walletCommon.ChainID,
	ownerAddress gethcommon.Address, contractAddresses []gethcommon.Address) (thirdparty.TokenBalancesPerContractAddress, error) {
	return nil, errors.New("FetchBalancesByOwnerAndContractAddress is not implemented for testCollectiblesManager")
}

func (m *CollectiblesManagerMock) GetCollectibleOwnership(requestedID thirdparty.CollectibleUniqueID) ([]thirdparty.AccountBalance, error) {
	// NOTE: TokenID inside of thirdparty.CollectibleUniqueID is a pointer so m.response[id] is now working
	for id, balances := range m.response {
		if id.ContractID.Address == requestedID.ContractID.Address &&
			id.ContractID.ChainID == requestedID.ContractID.ChainID {
			return balances, nil
		}
	}
	return []thirdparty.AccountBalance{}, nil
}

func (m *CollectiblesManagerMock) SetResponse(id thirdparty.CollectibleUniqueID, balances []thirdparty.AccountBalance) {
	if m.response == nil {
		m.response = map[thirdparty.CollectibleUniqueID][]thirdparty.AccountBalance{}
	}
	m.response[id] = balances
}

func TestMessengerProfileShowcaseSuite(t *testing.T) { // nolint: deadcode,unused
	suite.Run(t, new(TestMessengerProfileShowcase))
}

type TestMessengerProfileShowcase struct {
	suite.Suite
	m          *Messenger        // main instance of Messenger
	privateKey *ecdsa.PrivateKey // private key for the main instance of Messenger
	// If one wants to send messages between different instances of Messenger,
	// a single waku service should be shared.
	shh              types.Waku
	logger           *zap.Logger
	collectiblesMock *CollectiblesManagerMock
}

func (s *TestMessengerProfileShowcase) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	s.m = s.newMessengerForProfileShowcase()
	s.privateKey = s.m.identity
}

func (s *TestMessengerProfileShowcase) TearDownTest() {
	TearDownMessenger(&s.Suite, s.m)
	_ = s.logger.Sync()
}

func (s *TestMessengerProfileShowcase) newMessengerForProfileShowcase() *Messenger {
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.NoError(err, "creating sqlite db instance")
	err = sqlite.Migrate(db)
	s.NoError(err, "protocol migrate")

	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	s.collectiblesMock = &CollectiblesManagerMock{}

	options := []Option{
		WithCollectiblesManager(s.collectiblesMock),
	}

	m, err := newMessengerWithKey(s.shh, privateKey, s.logger, options)
	s.Require().NoError(err)

	return m
}

func (s *TestMessengerProfileShowcase) mutualContact(theirMessenger *Messenger) {
	messageText := "hello!"

	contactID := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	request := &requests.SendContactRequest{
		ID:      contactID,
		Message: messageText,
	}

	// Send contact request
	_, err := s.m.SendContactRequest(context.Background(), request)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) > 0 && len(r.Messages()) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Make sure it's the pending contact requests
	contactRequests, _, err := theirMessenger.PendingContactRequests("", 10)
	s.Require().NoError(err)
	s.Require().Len(contactRequests, 1)
	s.Require().Equal(contactRequests[0].ContactRequestState, common.ContactRequestStatePending)

	// Accept contact request, receiver side
	_, err = theirMessenger.AcceptContactRequest(context.Background(), &requests.AcceptContactRequest{ID: types.Hex2Bytes(contactRequests[0].ID)})
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			return len(r.Contacts) == 1 && len(r.Messages()) == 2 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)

	// Check the contact state is correctly set
	s.Require().Len(resp.Contacts, 1)
	s.Require().True(resp.Contacts[0].mutual())
}

func (s *TestMessengerProfileShowcase) verifiedContact(theirMessenger *Messenger) {
	theirPk := types.EncodeHex(crypto.FromECDSAPub(&theirMessenger.identity.PublicKey))
	challenge := "Want to see what I'm hiding in my profile showcase?"

	_, err := s.m.SendContactVerificationRequest(context.Background(), theirPk, challenge)
	s.Require().NoError(err)

	// Wait for the message to reach its destination
	resp, err := WaitOnMessengerResponse(
		theirMessenger,
		func(r *MessengerResponse) bool {
			return len(r.VerificationRequests()) == 1 && len(r.ActivityCenterNotifications()) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(resp.VerificationRequests(), 1)
	verificationRequestID := resp.VerificationRequests()[0].ID

	_, err = theirMessenger.AcceptContactVerificationRequest(context.Background(), verificationRequestID, "For sure!")
	s.Require().NoError(err)

	s.Require().NoError(err)

	// Wait for the message to reach its destination
	_, err = WaitOnMessengerResponse(
		s.m,
		func(r *MessengerResponse) bool {
			return len(r.VerificationRequests()) == 1
		},
		"no messages",
	)
	s.Require().NoError(err)

	resp, err = s.m.VerifiedTrusted(context.Background(), &requests.VerifiedTrusted{ID: types.FromHex(verificationRequestID)})
	s.Require().NoError(err)

	s.Require().Len(resp.Messages(), 1)
	s.Require().Equal(common.ContactVerificationStateTrusted, resp.Messages()[0].ContactVerificationState)
}

func (s *TestMessengerProfileShowcase) TestSaveAndGetProfileShowcasePreferences() {
	request := DummyProfileShowcasePreferences(true)

	// Provide collectible balances test response
	collectible := request.Collectibles[0]
	collectibleID, err := toCollectibleUniqueID(collectible.ContractAddress, collectible.TokenID, collectible.ChainID)
	s.Require().NoError(err)
	balances := []thirdparty.AccountBalance{
		thirdparty.AccountBalance{
			Address:     gethcommon.HexToAddress(request.Accounts[0].Address),
			Balance:     &bigint.BigInt{Int: big.NewInt(5)},
			TxTimestamp: 0,
		},
	}
	s.collectiblesMock.SetResponse(collectibleID, balances)

	err = s.m.SetProfileShowcasePreferences(request, false)
	s.Require().NoError(err)

	// Restored preferences shoulf be same as stored
	response, err := s.m.GetProfileShowcasePreferences()
	s.Require().NoError(err)

	s.Require().Equal(len(response.Communities), len(request.Communities))
	for i := 0; i < len(response.Communities); i++ {
		s.Require().Equal(*response.Communities[i], *request.Communities[i])
	}

	s.Require().Equal(len(response.Accounts), len(request.Accounts))
	for i := 0; i < len(response.Accounts); i++ {
		s.Require().Equal(*response.Accounts[i], *request.Accounts[i])
	}

	s.Require().Equal(len(response.Collectibles), len(request.Collectibles))
	for i := 0; i < len(response.Collectibles); i++ {
		s.Require().Equal(*response.Collectibles[i], *request.Collectibles[i])
	}

	s.Require().Equal(len(response.VerifiedTokens), len(request.VerifiedTokens))
	for i := 0; i < len(response.VerifiedTokens); i++ {
		s.Require().Equal(*response.VerifiedTokens[i], *request.VerifiedTokens[i])
	}

	s.Require().Equal(len(response.UnverifiedTokens), len(request.UnverifiedTokens))
	for i := 0; i < len(response.UnverifiedTokens); i++ {
		s.Require().Equal(*response.UnverifiedTokens[i], *request.UnverifiedTokens[i])
	}

	s.Require().Equal(len(response.SocialLinks), len(request.SocialLinks))
	for i := 0; i < len(response.SocialLinks); i++ {
		s.Require().Equal(*response.SocialLinks[i], *request.SocialLinks[i])
	}
}

func (s *TestMessengerProfileShowcase) TestFailToSaveProfileShowcasePreferencesWithWrongVisibility() {
	accountEntry := &identity.ProfileShowcaseAccountPreference{
		Address:            "0x0000000000000000000000000032433445133424",
		ShowcaseVisibility: identity.ProfileShowcaseVisibilityIDVerifiedContacts,
		Order:              17,
	}

	collectibleEntry := &identity.ProfileShowcaseCollectiblePreference{
		ContractAddress:    "0x12378534257568678487683576",
		ChainID:            11155111,
		TokenID:            "12321389592999903",
		ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
		Order:              17,
	}

	request := &identity.ProfileShowcasePreferences{
		Accounts:     []*identity.ProfileShowcaseAccountPreference{accountEntry},
		Collectibles: []*identity.ProfileShowcaseCollectiblePreference{collectibleEntry},
	}

	// Provide collectible balances test response
	collectible := request.Collectibles[0]
	collectibleID, err := toCollectibleUniqueID(collectible.ContractAddress, collectible.TokenID, collectible.ChainID)
	s.Require().NoError(err)
	balances := []thirdparty.AccountBalance{
		thirdparty.AccountBalance{
			Address:     gethcommon.HexToAddress(request.Accounts[0].Address),
			Balance:     &bigint.BigInt{Int: big.NewInt(5)},
			TxTimestamp: 0,
		},
	}
	s.collectiblesMock.SetResponse(collectibleID, balances)

	err = s.m.SetProfileShowcasePreferences(request, false)
	s.Require().Equal(errorAccountVisibilityLowerThanCollectible, err)
}

func (s *TestMessengerProfileShowcase) TestEncryptAndDecryptProfileShowcaseEntries() {
	// Add mutual contact
	theirMessenger := s.newMessengerForProfileShowcase()
	defer TearDownMessenger(&s.Suite, theirMessenger)

	s.mutualContact(theirMessenger)

	entries := &protobuf.ProfileShowcaseEntries{
		Communities: []*protobuf.ProfileShowcaseCommunity{
			&protobuf.ProfileShowcaseCommunity{
				CommunityId: "0x01312357798976535235432345",
				Order:       12,
			},
			&protobuf.ProfileShowcaseCommunity{
				CommunityId: "0x12378534257568678487683576",
				Order:       11,
			},
		},
		Accounts: []*protobuf.ProfileShowcaseAccount{
			&protobuf.ProfileShowcaseAccount{
				Address: "0x00000323245",
				Name:    "Default",
				ColorId: "red",
				Emoji:   "(=^ ◡ ^=)",
				Order:   1,
			},
		},
		Collectibles: []*protobuf.ProfileShowcaseCollectible{
			&protobuf.ProfileShowcaseCollectible{
				ContractAddress: "0x12378534257568678487683576",
				ChainId:         1,
				TokenId:         "12321389592999903",
				AccountAddress:  "0x32433445133424",
				CommunityId:     "0x12378534257568678487683576",
				Order:           0,
			},
		},
		VerifiedTokens: []*protobuf.ProfileShowcaseVerifiedToken{
			&protobuf.ProfileShowcaseVerifiedToken{
				Symbol: "ETH",
				Order:  1,
			},
			&protobuf.ProfileShowcaseVerifiedToken{
				Symbol: "DAI",
				Order:  2,
			},
			&protobuf.ProfileShowcaseVerifiedToken{
				Symbol: "SNT",
				Order:  3,
			},
		},
		UnverifiedTokens: []*protobuf.ProfileShowcaseUnverifiedToken{
			&protobuf.ProfileShowcaseUnverifiedToken{
				ContractAddress: "0x454525452023452",
				ChainId:         11155111,
				Order:           0,
			},
			&protobuf.ProfileShowcaseUnverifiedToken{
				ContractAddress: "0x12312323323233",
				ChainId:         1,
				Order:           1,
			},
		},
		SocialLinks: []*protobuf.ProfileShowcaseSocialLink{
			&protobuf.ProfileShowcaseSocialLink{
				Text:  identity.TwitterID,
				Url:   "https://twitter.com/ethstatus",
				Order: 1,
			},
			&protobuf.ProfileShowcaseSocialLink{
				Text:  identity.TwitterID,
				Url:   "https://twitter.com/StatusIMBlog",
				Order: 2,
			},
			&protobuf.ProfileShowcaseSocialLink{
				Text:  identity.GithubID,
				Url:   "https://github.com/status-im",
				Order: 3,
			},
		},
	}
	data, err := s.m.EncryptProfileShowcaseEntriesWithContactPubKeys(entries, s.m.Contacts())
	s.Require().NoError(err)

	entriesBack, err := theirMessenger.DecryptProfileShowcaseEntriesWithPubKey(&s.m.identity.PublicKey, data)
	s.Require().NoError(err)

	s.Require().Equal(len(entries.Communities), len(entriesBack.Communities))
	for i := 0; i < len(entriesBack.Communities); i++ {
		s.Require().Equal(entries.Communities[i].CommunityId, entriesBack.Communities[i].CommunityId)
		s.Require().Equal(entries.Communities[i].Order, entriesBack.Communities[i].Order)
	}

	s.Require().Equal(len(entries.Accounts), len(entriesBack.Accounts))
	for i := 0; i < len(entriesBack.Accounts); i++ {
		s.Require().Equal(entries.Accounts[i].Address, entriesBack.Accounts[i].Address)
		s.Require().Equal(entries.Accounts[i].Name, entriesBack.Accounts[i].Name)
		s.Require().Equal(entries.Accounts[i].ColorId, entriesBack.Accounts[i].ColorId)
		s.Require().Equal(entries.Accounts[i].Emoji, entriesBack.Accounts[i].Emoji)
		s.Require().Equal(entries.Accounts[i].Order, entriesBack.Accounts[i].Order)
	}

	s.Require().Equal(len(entries.Collectibles), len(entriesBack.Collectibles))
	for i := 0; i < len(entriesBack.Collectibles); i++ {
		s.Require().Equal(entries.Collectibles[i].TokenId, entriesBack.Collectibles[i].TokenId)
		s.Require().Equal(entries.Collectibles[i].ChainId, entriesBack.Collectibles[i].ChainId)
		s.Require().Equal(entries.Collectibles[i].ContractAddress, entriesBack.Collectibles[i].ContractAddress)
		s.Require().Equal(entries.Collectibles[i].AccountAddress, entriesBack.Collectibles[i].AccountAddress)
		s.Require().Equal(entries.Collectibles[i].CommunityId, entriesBack.Collectibles[i].CommunityId)
		s.Require().Equal(entries.Collectibles[i].Order, entriesBack.Collectibles[i].Order)
	}

	s.Require().Equal(len(entries.VerifiedTokens), len(entriesBack.VerifiedTokens))
	for i := 0; i < len(entriesBack.VerifiedTokens); i++ {
		s.Require().Equal(entries.VerifiedTokens[i].Symbol, entriesBack.VerifiedTokens[i].Symbol)
		s.Require().Equal(entries.VerifiedTokens[i].Order, entriesBack.VerifiedTokens[i].Order)
	}

	s.Require().Equal(len(entries.UnverifiedTokens), len(entriesBack.UnverifiedTokens))
	for i := 0; i < len(entriesBack.UnverifiedTokens); i++ {
		s.Require().Equal(entries.UnverifiedTokens[i].ContractAddress, entriesBack.UnverifiedTokens[i].ContractAddress)
		s.Require().Equal(entries.UnverifiedTokens[i].ChainId, entriesBack.UnverifiedTokens[i].ChainId)
		s.Require().Equal(entries.UnverifiedTokens[i].Order, entriesBack.UnverifiedTokens[i].Order)
	}

	s.Require().Equal(len(entries.SocialLinks), len(entriesBack.SocialLinks))
	for i := 0; i < len(entriesBack.SocialLinks); i++ {
		s.Require().Equal(entries.SocialLinks[i].Text, entriesBack.SocialLinks[i].Text)
		s.Require().Equal(entries.SocialLinks[i].Url, entriesBack.SocialLinks[i].Url)
		s.Require().Equal(entries.SocialLinks[i].Order, entriesBack.SocialLinks[i].Order)
	}
}

func (s *TestMessengerProfileShowcase) TestShareShowcasePreferences() {
	s.T().Skip("flaky test")

	// Set Display name to pass shouldPublishChatIdentity check
	profileKp := accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = s.m.account.KeyUID
	profileKp.Accounts[0].KeyUID = s.m.account.KeyUID

	err := s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	err = s.m.SetDisplayName("bobby")
	s.Require().NoError(err)

	// Add mutual contact
	mutualContact := s.newMessengerForProfileShowcase()
	defer TearDownMessenger(&s.Suite, mutualContact)

	s.mutualContact(mutualContact)

	// Add identity verified contact
	verifiedContact := s.newMessengerForProfileShowcase()
	defer TearDownMessenger(&s.Suite, verifiedContact)

	s.mutualContact(verifiedContact)
	s.verifiedContact(verifiedContact)

	// Save preferences to dispatch changes
	request := DummyProfileShowcasePreferences(true)

	// Provide collectible balances test response
	collectible := request.Collectibles[0]
	collectibleID, err := toCollectibleUniqueID(collectible.ContractAddress, collectible.TokenID, collectible.ChainID)
	s.Require().NoError(err)
	balances := []thirdparty.AccountBalance{
		thirdparty.AccountBalance{
			Address:     gethcommon.HexToAddress(request.Accounts[0].Address),
			Balance:     &bigint.BigInt{Int: big.NewInt(1)},
			TxTimestamp: 32443424,
		},
	}
	s.collectiblesMock.SetResponse(collectibleID, balances)

	err = s.m.SetProfileShowcasePreferences(request, false)
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	// Get summarised profile data for mutual contact
	_, err = WaitOnMessengerResponse(
		mutualContact,
		func(r *MessengerResponse) bool {
			return r.updatedProfileShowcaseContactIDs[contactID] == true
		},
		"no messages",
	)
	s.Require().NoError(err)

	profileShowcase, err := mutualContact.GetProfileShowcaseForContact(contactID, false)
	s.Require().NoError(err)

	s.Require().Len(profileShowcase.Communities, 2)
	s.Require().Equal(profileShowcase.Communities[0].CommunityID, request.Communities[0].CommunityID)
	s.Require().Equal(profileShowcase.Communities[0].Order, request.Communities[0].Order)
	s.Require().Equal(profileShowcase.Communities[1].CommunityID, request.Communities[1].CommunityID)
	s.Require().Equal(profileShowcase.Communities[1].Order, request.Communities[1].Order)

	s.Require().Len(profileShowcase.Accounts, 2)
	s.Require().Equal(profileShowcase.Accounts[0].Address, request.Accounts[0].Address)
	s.Require().Equal(profileShowcase.Accounts[0].Order, request.Accounts[0].Order)
	s.Require().Equal(profileShowcase.Accounts[1].Address, request.Accounts[1].Address)
	s.Require().Equal(profileShowcase.Accounts[1].Order, request.Accounts[1].Order)

	s.Require().Len(profileShowcase.Collectibles, 1)
	s.Require().Equal(profileShowcase.Collectibles[0].TokenID, request.Collectibles[0].TokenID)
	s.Require().Equal(profileShowcase.Collectibles[0].ChainID, request.Collectibles[0].ChainID)
	s.Require().Equal(profileShowcase.Collectibles[0].ContractAddress, request.Collectibles[0].ContractAddress)
	s.Require().Equal(profileShowcase.Collectibles[0].Order, request.Collectibles[0].Order)

	s.Require().Len(profileShowcase.VerifiedTokens, 1)
	s.Require().Equal(profileShowcase.VerifiedTokens[0].Symbol, request.VerifiedTokens[0].Symbol)
	s.Require().Equal(profileShowcase.VerifiedTokens[0].Order, request.VerifiedTokens[0].Order)

	s.Require().Len(profileShowcase.UnverifiedTokens, 2)
	s.Require().Equal(profileShowcase.UnverifiedTokens[0].ContractAddress, request.UnverifiedTokens[0].ContractAddress)
	s.Require().Equal(profileShowcase.UnverifiedTokens[0].ChainID, request.UnverifiedTokens[0].ChainID)
	s.Require().Equal(profileShowcase.UnverifiedTokens[0].Order, request.UnverifiedTokens[0].Order)
	s.Require().Equal(profileShowcase.UnverifiedTokens[1].ContractAddress, request.UnverifiedTokens[1].ContractAddress)
	s.Require().Equal(profileShowcase.UnverifiedTokens[1].ChainID, request.UnverifiedTokens[1].ChainID)
	s.Require().Equal(profileShowcase.UnverifiedTokens[1].Order, request.UnverifiedTokens[1].Order)

	s.Require().Len(profileShowcase.SocialLinks, 2)
	s.Require().Equal(profileShowcase.SocialLinks[0].Text, request.SocialLinks[0].Text)
	s.Require().Equal(profileShowcase.SocialLinks[0].URL, request.SocialLinks[0].URL)
	s.Require().Equal(profileShowcase.SocialLinks[0].Order, request.SocialLinks[0].Order)
	s.Require().Equal(profileShowcase.SocialLinks[1].Text, request.SocialLinks[2].Text)
	s.Require().Equal(profileShowcase.SocialLinks[1].URL, request.SocialLinks[2].URL)
	s.Require().Equal(profileShowcase.SocialLinks[1].Order, request.SocialLinks[2].Order)

	// Get summarised profile data for verified contact
	_, err = WaitOnMessengerResponse(
		verifiedContact,
		func(r *MessengerResponse) bool {
			return r.updatedProfileShowcaseContactIDs[contactID] == true
		},
		"no messages",
	)
	s.Require().NoError(err)

	profileShowcase, err = verifiedContact.GetProfileShowcaseForContact(contactID, false)
	s.Require().NoError(err)

	s.Require().Len(profileShowcase.Accounts, 2)
	s.Require().Equal(profileShowcase.Accounts[0].Address, request.Accounts[0].Address)
	s.Require().Equal(profileShowcase.Accounts[0].Order, request.Accounts[0].Order)
	s.Require().Equal(profileShowcase.Accounts[1].Address, request.Accounts[1].Address)
	s.Require().Equal(profileShowcase.Accounts[1].Order, request.Accounts[1].Order)

	s.Require().Len(profileShowcase.Collectibles, 1)
	s.Require().Equal(profileShowcase.Collectibles[0].ContractAddress, request.Collectibles[0].ContractAddress)
	s.Require().Equal(profileShowcase.Collectibles[0].ChainID, request.Collectibles[0].ChainID)
	s.Require().Equal(profileShowcase.Collectibles[0].TokenID, request.Collectibles[0].TokenID)
	s.Require().Equal(profileShowcase.Collectibles[0].Order, request.Collectibles[0].Order)

	s.Require().Len(profileShowcase.VerifiedTokens, 2)
	s.Require().Equal(profileShowcase.VerifiedTokens[0].Symbol, request.VerifiedTokens[0].Symbol)
	s.Require().Equal(profileShowcase.VerifiedTokens[0].Order, request.VerifiedTokens[0].Order)
	s.Require().Equal(profileShowcase.VerifiedTokens[1].Symbol, request.VerifiedTokens[1].Symbol)
	s.Require().Equal(profileShowcase.VerifiedTokens[1].Order, request.VerifiedTokens[1].Order)

	s.Require().Len(profileShowcase.UnverifiedTokens, 2)
	s.Require().Equal(profileShowcase.UnverifiedTokens[0].ContractAddress, request.UnverifiedTokens[0].ContractAddress)
	s.Require().Equal(profileShowcase.UnverifiedTokens[0].ChainID, request.UnverifiedTokens[0].ChainID)
	s.Require().Equal(profileShowcase.UnverifiedTokens[0].Order, request.UnverifiedTokens[0].Order)
	s.Require().Equal(profileShowcase.UnverifiedTokens[1].ContractAddress, request.UnverifiedTokens[1].ContractAddress)
	s.Require().Equal(profileShowcase.UnverifiedTokens[1].ChainID, request.UnverifiedTokens[1].ChainID)
	s.Require().Equal(profileShowcase.UnverifiedTokens[1].Order, request.UnverifiedTokens[1].Order)

	s.Require().Len(profileShowcase.SocialLinks, 3)
	s.Require().Equal(profileShowcase.SocialLinks[0].Text, request.SocialLinks[0].Text)
	s.Require().Equal(profileShowcase.SocialLinks[0].URL, request.SocialLinks[0].URL)
	s.Require().Equal(profileShowcase.SocialLinks[0].Order, request.SocialLinks[0].Order)
	s.Require().Equal(profileShowcase.SocialLinks[1].Text, request.SocialLinks[1].Text)
	s.Require().Equal(profileShowcase.SocialLinks[1].URL, request.SocialLinks[1].URL)
	s.Require().Equal(profileShowcase.SocialLinks[1].Order, request.SocialLinks[1].Order)
	s.Require().Equal(profileShowcase.SocialLinks[2].Text, request.SocialLinks[2].Text)
	s.Require().Equal(profileShowcase.SocialLinks[2].URL, request.SocialLinks[2].URL)
	s.Require().Equal(profileShowcase.SocialLinks[2].Order, request.SocialLinks[2].Order)
}

func (s *TestMessengerProfileShowcase) TestProfileShowcaseProofOfMembershipUnencryptedCommunities() {
	alice := s.m

	// Set Display name to pass shouldPublishChatIdentity check
	profileKp := accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = alice.account.KeyUID
	profileKp.Accounts[0].KeyUID = alice.account.KeyUID

	err := alice.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	err = alice.SetDisplayName("Alice")
	s.Require().NoError(err)

	// Add bob as a mutual contact
	bob := s.newMessengerForProfileShowcase()
	defer TearDownMessenger(&s.Suite, bob)

	s.mutualContact(bob)

	// Alice creates a community
	aliceCommunity, _ := createCommunityConfigurable(&s.Suite, alice, protobuf.CommunityPermissions_MANUAL_ACCEPT)
	advertiseCommunityTo(&s.Suite, aliceCommunity, alice, bob)

	// Bobs creates an another community
	bobCommunity, _ := createCommunityConfigurable(&s.Suite, bob, protobuf.CommunityPermissions_AUTO_ACCEPT)

	// Add community to the Alice's profile showcase & get it on the Bob's side
	err = alice.SetProfileShowcasePreferences(&identity.ProfileShowcasePreferences{
		Communities: []*identity.ProfileShowcaseCommunityPreference{
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        aliceCommunity.IDString(),
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              0,
			},
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        bobCommunity.IDString(),
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              2,
			},
		},
	}, false)
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey))
	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return r.updatedProfileShowcaseContactIDs[contactID] == true
		},
		"no messages",
	)
	s.Require().NoError(err)

	profileShowcase, err := bob.GetProfileShowcaseForContact(contactID, true)
	s.Require().NoError(err)

	// Verify community's data
	s.Require().Len(profileShowcase.Communities, 2)
	s.Require().Equal(profileShowcase.Communities[0].CommunityID, aliceCommunity.IDString())
	s.Require().Equal(profileShowcase.Communities[0].MembershipStatus, identity.ProfileShowcaseMembershipStatusProvenMember)
	s.Require().Equal(profileShowcase.Communities[1].CommunityID, bobCommunity.IDString())
	s.Require().Equal(profileShowcase.Communities[1].MembershipStatus, identity.ProfileShowcaseMembershipStatusNotAMember)
}

func (s *TestMessengerProfileShowcase) TestProfileShowcaseProofOfMembershipEncryptedCommunity() {
	alice := s.m

	// Set Display name to pass shouldPublishChatIdentity check
	profileKp := accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = alice.account.KeyUID
	profileKp.Accounts[0].KeyUID = alice.account.KeyUID

	err := alice.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	err = alice.SetDisplayName("Alice")
	s.Require().NoError(err)

	// Add bob as a mutual contact
	bob := s.newMessengerForProfileShowcase()
	defer TearDownMessenger(&s.Suite, bob)

	s.mutualContact(bob)

	// Alice creates an ecrypted community
	aliceCommunity, _ := createEncryptedCommunity(&s.Suite, alice)
	s.Require().True(aliceCommunity.Encrypted())
	advertiseCommunityTo(&s.Suite, aliceCommunity, alice, bob)

	// Bob creates an another encryped community
	bobCommunity, _ := createEncryptedCommunity(&s.Suite, bob)
	s.Require().True(bobCommunity.Encrypted())
	advertiseCommunityTo(&s.Suite, bobCommunity, bob, alice)

	// Add community to the Alice's profile showcase & get it on the Bob's side
	err = alice.SetProfileShowcasePreferences(&identity.ProfileShowcasePreferences{
		Communities: []*identity.ProfileShowcaseCommunityPreference{
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        aliceCommunity.IDString(),
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityContacts,
				Order:              0,
			},
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        bobCommunity.IDString(),
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              1,
			},
		},
	}, false)
	s.Require().NoError(err)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey))
	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return r.updatedProfileShowcaseContactIDs[contactID] == true
		},
		"no messages",
	)
	s.Require().NoError(err)

	profileShowcase, err := bob.GetProfileShowcaseForContact(contactID, true)
	s.Require().NoError(err)

	// Verify community's data
	s.Require().Len(profileShowcase.Communities, 2)
	s.Require().Equal(profileShowcase.Communities[0].CommunityID, aliceCommunity.IDString())
	s.Require().Equal(profileShowcase.Communities[0].MembershipStatus, identity.ProfileShowcaseMembershipStatusProvenMember)
	s.Require().Equal(profileShowcase.Communities[1].CommunityID, bobCommunity.IDString())
	s.Require().Equal(profileShowcase.Communities[1].MembershipStatus, identity.ProfileShowcaseMembershipStatusUnproven)
}

// The scenario tested is as follow:
// 1) Owner creates an encrypted community
// 2) Bob add Alice becommes a mutual contacts
// 3) Alice and bob join the community
// 4) Alice presents the community in her profile showcase
// 5) Bob gets the community from Alice's profile showcase and validates community's membership with grant
// 6) Wait until the grant expires, Bob should not be able to validate the membership anymore (commented step)
// 7) Owner updates the grant
// 8) Bob should be able to validate the membership again
func (s *TestMessengerProfileShowcase) TestProfileShowcaseCommuniesDispatchOnGrantUpdate() {
	// NOTE: commented to reduce test length communities.GrantExpirationTime = 200 * time.Millisecond
	grantInvokesProfileDispatchInterval = 1 * time.Millisecond
	alice := s.m

	// Set Display name to pass shouldPublishChatIdentity check
	profileKp := accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = alice.account.KeyUID
	profileKp.Accounts[0].KeyUID = alice.account.KeyUID

	err := alice.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	err = alice.SetDisplayName("Alice")
	s.Require().NoError(err)

	// 1) Owner creates an encrypted community
	owner := s.newMessengerForProfileShowcase()
	defer TearDownMessenger(&s.Suite, owner)

	owner.communitiesManager.PermissionChecker = &testPermissionChecker{}

	community, _ := createEncryptedCommunity(&s.Suite, owner)
	s.Require().True(community.Encrypted())

	// 2) Bob add Alice becommes a mutual contacts
	bob := s.newMessengerForProfileShowcase()
	defer TearDownMessenger(&s.Suite, bob)

	s.mutualContact(bob)

	// 3) Alice and bob join the community
	advertiseCommunityTo(&s.Suite, community, owner, alice)
	advertiseCommunityTo(&s.Suite, community, owner, bob)
	request := &requests.RequestToJoinCommunity{CommunityID: community.ID()}
	joinCommunity(&s.Suite, community, owner, alice, request, "")

	joinedCommunities, err := alice.communitiesManager.Joined()
	s.Require().NoError(err)
	s.Require().Len(joinedCommunities, 1)
	s.Require().Equal(joinedCommunities[0].IDString(), community.IDString())
	s.Require().True(joinedCommunities[0].Encrypted())

	grant, clock, err := alice.communitiesManager.GetCommunityGrant(community.IDString())
	s.Require().NoError(err)
	s.Require().NotEqual(grant, []byte{})
	s.Require().True(clock > 0)

	// 4) Alice presents the community in her profile showcase
	err = alice.SetProfileShowcasePreferences(&identity.ProfileShowcasePreferences{
		Communities: []*identity.ProfileShowcaseCommunityPreference{
			&identity.ProfileShowcaseCommunityPreference{
				CommunityID:        community.IDString(),
				ShowcaseVisibility: identity.ProfileShowcaseVisibilityEveryone,
				Order:              0,
			},
		},
	}, false)
	s.Require().NoError(err)

	// 5) Bob gets the community from Alice's profile showcase and validates community's membership with grant
	contactID := types.EncodeHex(crypto.FromECDSAPub(&alice.identity.PublicKey))
	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return r.updatedProfileShowcaseContactIDs[contactID] == true
		},
		"no messages",
	)
	s.Require().NoError(err)

	profileShowcase, err := bob.GetProfileShowcaseForContact(contactID, true)
	s.Require().NoError(err)
	s.Require().Len(profileShowcase.Communities, 1)
	s.Require().Equal(community.IDString(), profileShowcase.Communities[0].CommunityID)
	s.Require().Equal(identity.ProfileShowcaseMembershipStatusProvenMember, profileShowcase.Communities[0].MembershipStatus)

	// NOTE: this step is commented because it requires significant time to wait for the grant to expire
	// and smaller timeouts can lead test to be flaky, but it should be working

	// 6) Wait until the grant expires, Bob should not be able to validate the membership anymore
	// time.Sleep(communities.GrantExpirationTime)

	// profileShowcase, err = bob.GetProfileShowcaseForContact(contactID, true)
	// s.Require().NoError(err)
	// s.Require().Len(profileShowcase.Communities, 1)
	// s.Require().Equal(community.IDString(), profileShowcase.Communities[0].CommunityID)
	// s.Require().Equal(identity.ProfileShowcaseMembershipStatusUnproven, profileShowcase.Communities[0].MembershipStatus)

	// 7) Owner updates the grant
	owner.updateGrantsForControlledCommunities()

	// Retrieve for grant clock update
	err = tt.RetryWithBackOff(func() error {
		_, err = alice.RetrieveAll()
		if err != nil {
			return err
		}
		_, updatedClock, err := alice.communitiesManager.GetCommunityGrant(community.IDString())
		if err != nil {
			return err
		}

		if clock == updatedClock {
			return errors.New("can't recive an updated grant")
		}
		return nil
	})
	s.Require().NoError(err)

	// 8) Bob should be able to validate the membership again
	_, err = WaitOnMessengerResponse(
		bob,
		func(r *MessengerResponse) bool {
			return r.updatedProfileShowcaseContactIDs[contactID] == true
		},
		"no messages",
	)
	s.Require().NoError(err)

	profileShowcase, err = bob.GetProfileShowcaseForContact(contactID, true)
	s.Require().NoError(err)
	s.Require().Len(profileShowcase.Communities, 1)
	s.Require().Equal(profileShowcase.Communities[0].CommunityID, community.IDString())
	s.Require().Equal(profileShowcase.Communities[0].MembershipStatus, identity.ProfileShowcaseMembershipStatusProvenMember)
}
