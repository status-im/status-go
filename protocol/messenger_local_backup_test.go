package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/accounts/accountsevent"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func TestMessengerLocalBackupSuite(t *testing.T) {
	suite.Run(t, new(MessengerBackupSuite))
}

type MessengerLocalBackupSuite struct {
	MessengerBaseTestSuite
}

func (s *MessengerBackupSuite) TestLocalBackup() {
	s.tmpdir = s.T().TempDir()
	backupOptions := []Option{
		WithLocalBackup(&params.BackupConfig{
			DataDir: filepath.Join(s.tmpdir, params.BackupsRelativePath),
		}),
	}
	const bob1DisplayName = "bobby"

	// Create bob1
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	bob1, err := newMessengerWithKey(s.shh, privateKey, s.logger, backupOptions)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, bob1)

	// Create bob2
	accountsFeed := &event.Feed{}
	backupOptions = append(backupOptions, WithAccountsFeed(accountsFeed))
	bob2, err := newMessengerWithKey(s.shh, bob1.identity, s.logger, backupOptions)
	s.Require().NoError(err)
	defer TearDownMessenger(&s.Suite, bob2)

	// Make sure there is no backup at first
	backupFile := filepath.Join(bob1.config.backupConfig.DataDir, "user_data.bkp")
	err = os.RemoveAll(backupFile)
	s.Require().NoError(err)

	// -------------------- CONTACTS --------------------
	// Create 2 contacts
	contact1Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID1 := types.EncodeHex(crypto.FromECDSAPub(&contact1Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID1})
	s.Require().NoError(err)

	contact2Key, err := crypto.GenerateKey()
	s.Require().NoError(err)
	contactID2 := types.EncodeHex(crypto.FromECDSAPub(&contact2Key.PublicKey))

	_, err = bob1.AddContact(context.Background(), &requests.AddContact{ID: contactID2})
	s.Require().NoError(err)

	s.Require().Len(bob1.Contacts(), 2)

	// Validate contacts on bob1
	actualContacts := bob1.Contacts()
	if actualContacts[0].ID == contactID1 {
		s.Require().Equal(actualContacts[0].ID, contactID1)
		s.Require().Equal(actualContacts[1].ID, contactID2)
	} else {
		s.Require().Equal(actualContacts[0].ID, contactID2)
		s.Require().Equal(actualContacts[1].ID, contactID1)
	}
	s.Require().Equal(ContactRequestStateSent, actualContacts[0].ContactRequestLocalState)
	s.Require().Equal(ContactRequestStateSent, actualContacts[1].ContactRequestLocalState)
	s.Require().True(actualContacts[0].added())
	s.Require().True(actualContacts[1].added())

	// Check that bob2 has no contacts
	s.Require().Len(bob2.Contacts(), 0)

	// -------------------- PROFILE SETTINGS --------------------
	// Set some profile settings on bob1
	bobProfileKp := accounts.GetProfileKeypairForTest(true, false, false)
	bobProfileKp.KeyUID = bob1.account.KeyUID
	bobProfileKp.Accounts[0].KeyUID = bob1.account.KeyUID

	err = bob1.settings.SaveOrUpdateKeypair(bobProfileKp)
	s.Require().NoError(err)

	err = bob1.SetDisplayName(bob1DisplayName)
	s.Require().NoError(err)
	bob1KeyUID := bob1.account.KeyUID
	imagesExpected := fmt.Sprintf(`[{"keyUid":"%s","type":"large","uri":"data:image/png;base64,iVBORw0KGgoAAAANSUg=","width":240,"height":300,"fileSize":1024,"resizeTarget":240,"clock":0},{"keyUid":"%s","type":"thumbnail","uri":"data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=","width":80,"height":80,"fileSize":256,"resizeTarget":80,"clock":0}]`,
		bob1KeyUID, bob1KeyUID)

	iis := images.SampleIdentityImages()
	s.Require().NoError(bob1.multiAccounts.StoreIdentityImages(bob1KeyUID, iis, false))

	bob1EnsUsernameDetail, err := bob1.saveEnsUsernameDetailProto(&protobuf.SyncEnsUsernameDetail{
		Clock:    1,
		Username: "bob1.eth",
		ChainId:  1,
		Removed:  false,
	})
	s.Require().NoError(err)

	profileShowcasePreferences := DummyProfileShowcasePreferences(false)
	err = bob1.SetProfileShowcasePreferences(profileShowcasePreferences, false)
	s.Require().NoError(err)

	// Validate profile settings on bob1
	storedBob1DisplayName, err := bob1.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().Equal(bob1DisplayName, storedBob1DisplayName)

	bob1Images, err := bob1.multiAccounts.GetIdentityImages(bob1KeyUID)
	s.Require().NoError(err)
	jBob1Images, err := json.Marshal(bob1Images)
	s.Require().NoError(err)
	s.Require().Equal(imagesExpected, string(jBob1Images))

	bob1EnsUsernameDetails, err := bob1.getEnsUsernameDetails()
	s.Require().NoError(err)
	s.Require().Equal(1, len(bob1EnsUsernameDetails))

	bob1ProfileShowcasePreferences, err := bob1.GetProfileShowcasePreferences()
	s.Require().NoError(err)
	s.Require().NotNil(bob1ProfileShowcasePreferences)
	s.Require().Greater(bob1ProfileShowcasePreferences.Clock, uint64(0))
	profileShowcasePreferences.Clock = bob1ProfileShowcasePreferences.Clock // override clock for simpler comparison
	s.Require().True(reflect.DeepEqual(profileShowcasePreferences, bob1ProfileShowcasePreferences))

	// Check bob2
	storedBob2DisplayName, err := bob2.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().Equal(DefaultProfileDisplayName, storedBob2DisplayName)

	var expectedEmpty []*images.IdentityImage
	bob2Images, err := bob2.multiAccounts.GetIdentityImages(bob1KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(expectedEmpty, bob2Images)

	bob2EnsUsernameDetails, err := bob2.getEnsUsernameDetails()
	s.Require().NoError(err)
	s.Require().Equal(0, len(bob2EnsUsernameDetails))

	bob2ProfileShowcasePreferences, err := bob2.GetProfileShowcasePreferences()
	s.Require().NoError(err)
	s.Require().NotNil(bob2ProfileShowcasePreferences)
	s.Require().Equal(uint64(0), bob2ProfileShowcasePreferences.Clock)
	s.Require().Len(bob2ProfileShowcasePreferences.Communities, 0)
	s.Require().Len(bob2ProfileShowcasePreferences.Accounts, 0)
	s.Require().Len(bob2ProfileShowcasePreferences.Collectibles, 0)
	s.Require().Len(bob2ProfileShowcasePreferences.VerifiedTokens, 0)
	s.Require().Len(bob2ProfileShowcasePreferences.UnverifiedTokens, 0)
	s.Require().Len(bob2ProfileShowcasePreferences.SocialLinks, 0)

	// -------------------- SETTINGS --------------------
	// Set some settings on bob1
	const (
		bob1Currency                  = "eur"
		bob1MessagesFromContactsOnly  = true
		bob1ProfilePicturesShowTo     = settings.ProfilePicturesShowToEveryone
		bob1ProfilePicturesVisibility = settings.ProfilePicturesVisibilityEveryone
		bob1Bio                       = "bio"
		bob1Mnemonic                  = ""
		bob1MnemonicRemoved           = true
		bob1PreferredName             = "talent"
		bob1UrlUnfUnfurlingMode       = settings.URLUnfurlingEnableAll
	)

	// Create bob1 and set fields which are supposed to be backed up to/fetched from waku
	err = bob1.settings.SaveSettingField(settings.Currency, bob1Currency)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.MessagesFromContactsOnly, bob1MessagesFromContactsOnly)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.ProfilePicturesShowTo, bob1ProfilePicturesShowTo)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.ProfilePicturesVisibility, bob1ProfilePicturesVisibility)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.Bio, bob1Bio)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.Mnemonic, bob1Mnemonic)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.PreferredName, bob1PreferredName)
	s.Require().NoError(err)
	err = bob1.settings.SaveSettingField(settings.URLUnfurlingMode, bob1UrlUnfUnfurlingMode)
	s.Require().NoError(err)

	// Check bob1
	s.Require().Equal(bob1DisplayName, storedBob1DisplayName)
	storedBob1Currency, err := bob1.settings.GetCurrency()
	s.Require().NoError(err)
	s.Require().Equal(bob1Currency, storedBob1Currency)
	storedBob1MessagesFromContactsOnly, err := bob1.settings.GetMessagesFromContactsOnly()
	s.Require().NoError(err)
	s.Require().Equal(bob1MessagesFromContactsOnly, storedBob1MessagesFromContactsOnly)
	storedBob1ProfilePicturesShowTo, err := bob1.settings.GetProfilePicturesShowTo()
	s.Require().NoError(err)
	s.Require().Equal(int64(bob1ProfilePicturesShowTo), storedBob1ProfilePicturesShowTo)
	storedBob1ProfilePicturesVisibility, err := bob1.settings.GetProfilePicturesVisibility()
	s.Require().NoError(err)
	s.Require().Equal(int(bob1ProfilePicturesVisibility), storedBob1ProfilePicturesVisibility)
	storedBob1Bio, err := bob1.settings.Bio()
	s.Require().NoError(err)
	s.Require().Equal(bob1Bio, storedBob1Bio)
	storedMnemonic, err := bob1.settings.Mnemonic()
	s.Require().NoError(err)
	s.Require().Equal(bob1Mnemonic, storedMnemonic)
	storedMnemonicRemoved, err := bob1.settings.MnemonicRemoved()
	s.Require().NoError(err)
	s.Require().Equal(bob1MnemonicRemoved, storedMnemonicRemoved)
	storedPreferredName, err := bob1.settings.GetPreferredUsername()
	s.NoError(err)
	s.Require().Equal(bob1PreferredName, storedPreferredName)
	storedBob1UrlUnfurlingMode, err := bob1.settings.URLUnfurlingMode()
	s.NoError(err)
	s.Require().Equal(int64(bob1UrlUnfUnfurlingMode), storedBob1UrlUnfurlingMode)

	// Check bob2
	s.Require().NotEqual(storedBob1DisplayName, storedBob2DisplayName)
	storedBob2Currency, err := bob2.settings.GetCurrency()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1Currency, storedBob2Currency)
	storedBob2MessagesFromContactsOnly, err := bob2.settings.GetMessagesFromContactsOnly()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1MessagesFromContactsOnly, storedBob2MessagesFromContactsOnly)
	storedBob2ProfilePicturesShowTo, err := bob2.settings.GetProfilePicturesShowTo()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1ProfilePicturesShowTo, storedBob2ProfilePicturesShowTo)
	storedBob2ProfilePicturesVisibility, err := bob2.settings.GetProfilePicturesVisibility()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1ProfilePicturesVisibility, storedBob2ProfilePicturesVisibility)
	storedBob2Bio, err := bob2.settings.Bio()
	s.Require().NoError(err)
	s.Require().NotEqual(storedBob1Bio, storedBob2Bio)
	storedBob2MnemonicRemoved, err := bob2.settings.MnemonicRemoved()
	s.Require().NoError(err)
	s.Require().Equal(false, storedBob2MnemonicRemoved)
	storedBob2PreferredName, err := bob2.settings.GetPreferredUsername()
	s.NoError(err)
	s.Require().Equal("", storedBob2PreferredName)
	storedBob2UrlUnfurlingMode, err := bob2.settings.URLUnfurlingMode()
	s.NoError(err)
	s.Require().Equal(int64(settings.URLUnfurlingAlwaysAsk), storedBob2UrlUnfurlingMode)

	//-------------------- COMMUNITIES --------------------
	// Create a community
	description := &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Name:        "status",
		Color:       "#ffffff",
		Description: "status community description",
	}

	// Create a community chat
	response, err := bob1.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	// Check bob2
	communities, err := bob2.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 0)

	// --------------------- LEFT COMMUNITY --------------------
	// Create another community
	description = &requests.CreateCommunity{
		Membership:  protobuf.CommunityPermissions_MANUAL_ACCEPT,
		Name:        "other-status",
		Color:       "#fffff4",
		Description: "other status community description",
	}

	response, err = bob1.CreateCommunity(description, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	newCommunity := response.Communities()[0]

	// Leave community
	response, err = bob1.LeaveCommunity(newCommunity.ID())
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// Check bob2
	communities, err = bob2.Communities()
	s.Require().NoError(err)
	s.Require().Len(communities, 0)

	// --------------------- WATCHONLY ACCOUTNS --------------------
	// Create watch-only accounts
	woAccounts := accounts.GetWatchOnlyAccountsForTest()
	err = bob1.settings.SaveOrUpdateAccounts(woAccounts, false)
	s.Require().NoError(err)

	// Validate watch-only accounts on bob1
	dbWoAccounts1, err := bob1.settings.GetActiveWatchOnlyAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbWoAccounts1))
	s.Require().True(haveSameElements(woAccounts, dbWoAccounts1, accounts.SameAccounts))

	// Setup bob2
	s.Require().NotNil(bob2.config.accountsFeed)
	ch := make(chan accountsevent.Event, 20)
	sub := bob2.config.accountsFeed.Subscribe(ch)

	// --------------------- CHATS --------------------
	// Create a group chat
	response, err = bob1.CreateGroupChatWithMembers(context.Background(), "group", []string{})
	s.NoError(err)
	s.Require().Len(response.Chats(), 1)

	ourGroupChat := response.Chats()[0]

	err = bob1.SaveChat(ourGroupChat)
	s.NoError(err)

	// Create a one-to-one chat
	alice := s.newMessenger()
	defer TearDownMessenger(&s.Suite, alice)

	ourOneOneChat := CreateOneToOneChat("Our 1TO1", &alice.identity.PublicKey, alice.getTimesource())
	err = bob1.SaveChat(ourOneOneChat)
	s.Require().NoError(err)

	// -------------------- BACKUP --------------------
	// Backup
	err = bob1.BackupDataLocally(context.Background())
	s.Require().NoError(err)

	// Import the backup file and process it
	response, err = bob2.ImportLocalBackupFile(backupFile)
	s.Require().NoError(err)
	s.Require().NotNil(response)

	// -------------------- VALIDATE BACKUP --------------------
	// Validate contacts on bob2
	s.Require().Len(response.Contacts, 2)
	s.Require().Len(bob2.Contacts(), 2)

	// Validate profile settings on bob2
	storedBob2DisplayName, err = bob2.settings.DisplayName()
	s.Require().NoError(err)
	s.Require().Equal(bob1DisplayName, storedBob2DisplayName)

	bob2Images, err = bob2.multiAccounts.GetIdentityImages(bob1KeyUID)
	s.Require().NoError(err)
	s.Require().Equal(2, len(bob2Images))
	s.Require().Equal(bob2Images[0].Payload, bob1Images[0].Payload)
	s.Require().Equal(bob2Images[1].Payload, bob1Images[1].Payload)

	bob2EnsUsernameDetails, err = bob2.getEnsUsernameDetails()
	s.Require().NoError(err)
	s.Require().Equal(1, len(bob2EnsUsernameDetails))
	s.Require().Equal(bob1EnsUsernameDetail, bob2EnsUsernameDetails[0])

	bob2ProfileShowcasePreferences, err = bob1.GetProfileShowcasePreferences()
	s.Require().NoError(err)
	s.Require().True(reflect.DeepEqual(bob1ProfileShowcasePreferences, bob2ProfileShowcasePreferences))

	// Validate settings on bob2
	storedBob2Currency, err = bob2.settings.GetCurrency()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1Currency, storedBob2Currency)
	storedBob2MessagesFromContactsOnly, err = bob2.settings.GetMessagesFromContactsOnly()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1MessagesFromContactsOnly, storedBob2MessagesFromContactsOnly)
	storedBob2ProfilePicturesShowTo, err = bob2.settings.GetProfilePicturesShowTo()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1ProfilePicturesShowTo, storedBob2ProfilePicturesShowTo)
	storedBob2ProfilePicturesVisibility, err = bob2.settings.GetProfilePicturesVisibility()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1ProfilePicturesVisibility, storedBob2ProfilePicturesVisibility)
	storedBob2Bio, err = bob2.settings.Bio()
	s.Require().NoError(err)
	s.Require().Equal(storedBob1Bio, storedBob2Bio)
	storedBob2MnemonicRemoved, err = bob2.settings.MnemonicRemoved()
	s.Require().NoError(err)
	s.Require().Equal(bob1MnemonicRemoved, storedBob2MnemonicRemoved)
	storedBob2PreferredName, err = bob2.settings.GetPreferredUsername()
	s.NoError(err)
	s.Require().Equal(bob1PreferredName, storedBob2PreferredName)
	s.Require().Equal(bob1PreferredName, bob2.account.Name)
	storedBob2UrlUnfurlingMode, err = bob2.settings.URLUnfurlingMode()
	s.NoError(err)
	s.Require().Equal(storedBob1UrlUnfurlingMode, storedBob2UrlUnfurlingMode)

	// Validate communities on bob2
	communities, err = bob2.JoinedCommunities()
	s.Require().NoError(err)
	s.Require().Len(communities, 1)

	// Validate watch-only accounts on bob2
	dbWoAccounts2, err := bob2.settings.GetActiveWatchOnlyAccounts()
	s.Require().NoError(err)
	s.Require().Equal(len(woAccounts), len(dbWoAccounts2))
	s.Require().True(haveSameElements(woAccounts, dbWoAccounts2, accounts.SameAccounts))
	// Check whether accounts added event is sent
	select {
	case <-time.After(1 * time.Second):
		s.Fail("Timed out waiting for accountsevent")
	case event := <-ch:
		switch event.Type {
		case accountsevent.EventTypeAdded:
			s.Require().Len(event.Accounts, 1)
			s.Require().Equal(common.Address(dbWoAccounts2[0].Address), event.Accounts[0])
		}
	}
	sub.Unsubscribe()

	// Validate chats on bob2
	// Group chat
	chat, ok := response.chats[ourGroupChat.ID]
	s.Require().True(ok)
	s.Require().Equal(ourGroupChat.Name, chat.Name)

	chat, ok = bob2.allChats.Load(ourGroupChat.ID)
	s.Require().True(ok)
	s.Require().Equal(ourGroupChat.Name, chat.Name)

	// One on one chat
	chat, ok = response.chats[ourOneOneChat.ID]
	s.Require().True(ok)
	s.Require().Equal("", chat.Name) // We set 1-1 chat names to "" because the name is not good

	chat, ok = bob2.allChats.Load(ourOneOneChat.ID)
	s.Require().True(ok)
	s.Require().True(chat.Active)
	s.Require().Equal("", chat.Name)
}
