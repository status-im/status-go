package protocol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerProfileShowcaseSuite(t *testing.T) { // nolint: deadcode,unused
	suite.Run(t, new(TestMessengerProfileShowcase))
}

type TestMessengerProfileShowcase struct {
	MessengerBaseTestSuite
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

func (s *TestMessengerProfileShowcase) prepareShowcasePreferences() *ProfileShowcasePreferences {
	communityEntry1 := &ProfileShowcaseCommunityPreference{
		CommunityID:        "0x01312357798976434",
		ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
		Order:              10,
	}

	communityEntry2 := &ProfileShowcaseCommunityPreference{
		CommunityID:        "0x01312357798976535",
		ShowcaseVisibility: ProfileShowcaseVisibilityContacts,
		Order:              11,
	}

	communityEntry3 := &ProfileShowcaseCommunityPreference{
		CommunityID:        "0x01312353452343552",
		ShowcaseVisibility: ProfileShowcaseVisibilityIDVerifiedContacts,
		Order:              12,
	}

	accountEntry := &ProfileShowcaseAccountPreference{
		Address:            "0cx34662234",
		Name:               "Status Account",
		ColorID:            "blue",
		Emoji:              ">:-]",
		ShowcaseVisibility: ProfileShowcaseVisibilityEveryone,
		Order:              17,
	}

	collectibleEntry := &ProfileShowcaseCollectiblePreference{
		UID:                "0x12378534257568678487683576",
		CommunityID:        "0x01312357798976535",
		ShowcaseVisibility: ProfileShowcaseVisibilityIDVerifiedContacts,
		Order:              17,
	}

	assetEntry1 := &ProfileShowcaseAssetPreference{
		Symbol:             "ETH",
		ContractAddress:    "0xABCDEF123456789",
		CommunityID:        "",
		ShowcaseVisibility: ProfileShowcaseVisibilityNoOne,
		Order:              12,
	}

	assetEntry2 := &ProfileShowcaseAssetPreference{
		Symbol:             "DAI",
		ContractAddress:    "0x123456789ABCDEF",
		CommunityID:        "",
		ShowcaseVisibility: ProfileShowcaseVisibilityIDVerifiedContacts,
		Order:              17,
	}

	assetEntry3 := &ProfileShowcaseAssetPreference{
		Symbol:             "SNT",
		ContractAddress:    "0x1234ABCDEF56789",
		CommunityID:        "0x01312357798976535",
		ShowcaseVisibility: ProfileShowcaseVisibilityContacts,
		Order:              14,
	}

	return &ProfileShowcasePreferences{
		Communities:  []*ProfileShowcaseCommunityPreference{communityEntry1, communityEntry2, communityEntry3},
		Accounts:     []*ProfileShowcaseAccountPreference{accountEntry},
		Collectibles: []*ProfileShowcaseCollectiblePreference{collectibleEntry},
		Assets:       []*ProfileShowcaseAssetPreference{assetEntry1, assetEntry2, assetEntry3},
	}
}

func (s *TestMessengerProfileShowcase) TestSetAndGetProfileShowcasePreferences() {
	request := s.prepareShowcasePreferences()
	err := s.m.SetProfileShowcasePreferences(request)
	s.Require().NoError(err)

	// Restored preferences shoulf be same as stored
	response, err := s.m.GetProfileShowcasePreferences()
	s.Require().NoError(err)

	s.Require().Len(response.Communities, 3)
	s.Require().Equal(response.Communities[0], request.Communities[0])
	s.Require().Equal(response.Communities[1], request.Communities[1])
	s.Require().Equal(response.Communities[2], request.Communities[2])

	s.Require().Len(response.Accounts, 1)
	s.Require().Equal(response.Accounts[0], request.Accounts[0])

	s.Require().Len(response.Collectibles, 1)
	s.Require().Equal(response.Collectibles[0], request.Collectibles[0])

	s.Require().Len(response.Assets, 3)
	s.Require().Equal(response.Assets[0], request.Assets[0])
	s.Require().Equal(response.Assets[1], request.Assets[1])
	s.Require().Equal(response.Assets[2], request.Assets[2])
}

func (s *TestMessengerProfileShowcase) TestEncryptAndDecryptProfileShowcaseEntries() {
	// Add mutual contact
	theirMessenger := s.newMessenger()
	_, err := theirMessenger.Start()
	s.Require().NoError(err)
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
				Uid:         "dj31nk13nrjn312jrmi1mjjd",
				CommunityId: "0x12378534257568678487683576",
				Order:       0,
			},
		},
		Assets: []*protobuf.ProfileShowcaseAsset{
			&protobuf.ProfileShowcaseAsset{
				Symbol:          "ETH",
				CommunityId:     "0x01312357798976535235432345",
				ContractAddress: "",
				Order:           2,
			},
			&protobuf.ProfileShowcaseAsset{
				Symbol:          "DAI",
				CommunityId:     "",
				ContractAddress: "0x123456789ABCDEF",
				Order:           3,
			},
			&protobuf.ProfileShowcaseAsset{
				Symbol:          "SNT",
				CommunityId:     "0x12378534257568678487683576",
				ContractAddress: "0xABCDEF123456789",
				Order:           1,
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
		s.Require().Equal(entries.Collectibles[i].Uid, entriesBack.Collectibles[i].Uid)
		s.Require().Equal(entries.Collectibles[i].CommunityId, entriesBack.Collectibles[i].CommunityId)
		s.Require().Equal(entries.Collectibles[i].Order, entriesBack.Collectibles[i].Order)
	}

	s.Require().Equal(len(entries.Assets), len(entriesBack.Assets))
	for i := 0; i < len(entriesBack.Assets); i++ {
		s.Require().Equal(entries.Assets[i].Symbol, entriesBack.Assets[i].Symbol)
		s.Require().Equal(entries.Assets[i].CommunityId, entriesBack.Assets[i].CommunityId)
		s.Require().Equal(entries.Assets[i].ContractAddress, entriesBack.Assets[i].ContractAddress)
		s.Require().Equal(entries.Assets[i].Order, entriesBack.Assets[i].Order)
	}
}

func (s *TestMessengerProfileShowcase) TestShareShowcasePreferences() {
	// Set Display name to pass shouldPublishChatIdentity check
	profileKp := accounts.GetProfileKeypairForTest(true, false, false)
	profileKp.KeyUID = s.m.account.KeyUID
	profileKp.Accounts[0].KeyUID = s.m.account.KeyUID

	err := s.m.settings.SaveOrUpdateKeypair(profileKp)
	s.Require().NoError(err)

	err = s.m.SetDisplayName("bobby")
	s.Require().NoError(err)

	// Add mutual contact
	mutualContact := s.newMessenger()
	_, err = mutualContact.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, mutualContact)

	s.mutualContact(mutualContact)

	// Add identity verified contact
	verifiedContact := s.newMessenger()
	_, err = verifiedContact.Start()
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, verifiedContact)

	s.mutualContact(verifiedContact)
	s.verifiedContact(verifiedContact)

	// Save preferences to dispatch changes
	request := s.prepareShowcasePreferences()
	err = s.m.SetProfileShowcasePreferences(request)
	s.Require().NoError(err)

	// Get summarised profile data for mutual contact
	resp, err := WaitOnMessengerResponse(
		mutualContact,
		func(r *MessengerResponse) bool {
			return len(r.updatedProfileShowcases) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(resp.updatedProfileShowcases, 1)

	contactID := types.EncodeHex(crypto.FromECDSAPub(&s.m.identity.PublicKey))
	profileShowcase := resp.updatedProfileShowcases[contactID]

	s.Require().Len(profileShowcase.Communities, 2)

	// For everyone
	s.Require().Equal(profileShowcase.Communities[0].CommunityID, request.Communities[0].CommunityID)
	s.Require().Equal(profileShowcase.Communities[0].Order, request.Communities[0].Order)

	// For contacts
	s.Require().Equal(profileShowcase.Communities[1].CommunityID, request.Communities[1].CommunityID)
	s.Require().Equal(profileShowcase.Communities[1].Order, request.Communities[1].Order)

	s.Require().Len(profileShowcase.Accounts, 1)
	s.Require().Equal(profileShowcase.Accounts[0].Address, request.Accounts[0].Address)
	s.Require().Equal(profileShowcase.Accounts[0].Name, request.Accounts[0].Name)
	s.Require().Equal(profileShowcase.Accounts[0].ColorID, request.Accounts[0].ColorID)
	s.Require().Equal(profileShowcase.Accounts[0].Emoji, request.Accounts[0].Emoji)
	s.Require().Equal(profileShowcase.Accounts[0].Order, request.Accounts[0].Order)

	s.Require().Len(profileShowcase.Collectibles, 0)

	s.Require().Len(profileShowcase.Assets, 1)
	s.Require().Equal(profileShowcase.Assets[0].Symbol, request.Assets[2].Symbol)
	s.Require().Equal(profileShowcase.Assets[0].CommunityID, request.Assets[2].CommunityID)
	s.Require().Equal(profileShowcase.Assets[0].ContractAddress, request.Assets[2].ContractAddress)
	s.Require().Equal(profileShowcase.Assets[0].Order, request.Assets[2].Order)

	// Get summarised profile data for verified contact
	resp, err = WaitOnMessengerResponse(
		verifiedContact,
		func(r *MessengerResponse) bool {
			return len(r.updatedProfileShowcases) > 0
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(resp.updatedProfileShowcases, 1)

	// Here let's try synchronous
	profileShowcase, err = verifiedContact.GetProfileShowcaseForContact(contactID)
	s.Require().NoError(err)

	s.Require().Len(profileShowcase.Communities, 3)

	// For everyone
	s.Require().Equal(profileShowcase.Communities[0].CommunityID, request.Communities[0].CommunityID)
	s.Require().Equal(profileShowcase.Communities[0].Order, request.Communities[0].Order)

	// For contacts
	s.Require().Equal(profileShowcase.Communities[1].CommunityID, request.Communities[1].CommunityID)
	s.Require().Equal(profileShowcase.Communities[1].Order, request.Communities[1].Order)

	// For id verified
	s.Require().Equal(profileShowcase.Communities[2].CommunityID, request.Communities[2].CommunityID)
	s.Require().Equal(profileShowcase.Communities[2].Order, request.Communities[2].Order)

	s.Require().Len(profileShowcase.Accounts, 1)
	s.Require().Equal(profileShowcase.Accounts[0].Address, request.Accounts[0].Address)
	s.Require().Equal(profileShowcase.Accounts[0].Name, request.Accounts[0].Name)
	s.Require().Equal(profileShowcase.Accounts[0].ColorID, request.Accounts[0].ColorID)
	s.Require().Equal(profileShowcase.Accounts[0].Emoji, request.Accounts[0].Emoji)
	s.Require().Equal(profileShowcase.Accounts[0].Order, request.Accounts[0].Order)

	s.Require().Len(profileShowcase.Collectibles, 1)
	s.Require().Equal(profileShowcase.Collectibles[0].UID, request.Collectibles[0].UID)
	s.Require().Equal(profileShowcase.Collectibles[0].CommunityID, request.Collectibles[0].CommunityID)
	s.Require().Equal(profileShowcase.Collectibles[0].Order, request.Collectibles[0].Order)

	s.Require().Len(profileShowcase.Assets, 2)
	s.Require().Equal(profileShowcase.Assets[0].Symbol, request.Assets[2].Symbol)
	s.Require().Equal(profileShowcase.Assets[0].CommunityID, request.Assets[2].CommunityID)
	s.Require().Equal(profileShowcase.Assets[0].ContractAddress, request.Assets[2].ContractAddress)
	s.Require().Equal(profileShowcase.Assets[0].Order, request.Assets[2].Order)
	s.Require().Equal(profileShowcase.Assets[1].Symbol, request.Assets[1].Symbol)
	s.Require().Equal(profileShowcase.Assets[1].CommunityID, request.Assets[1].CommunityID)
	s.Require().Equal(profileShowcase.Assets[1].ContractAddress, request.Assets[1].ContractAddress)
	s.Require().Equal(profileShowcase.Assets[1].Order, request.Assets[1].Order)
}
