//go:build use_torrent

package protocol

import (
	"os"
	"time"

	"golang.org/x/exp/maps"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/communities/archive"
	archivetypes "github.com/status-im/status-go/internal/protocol/communities/archive/types"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/internal/protocol/requests"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func (s *MessengerCommunitiesTokenPermissionsSuite) TestImportDecryptedArchiveMessages() {
	// 1.1. Create community
	community, chat := s.createCommunity()

	// 1.2. Setup permissions
	communityPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_BECOME_MEMBER,
		TokenCriteria: []*protobuf.TokenCriteria{
			{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x124"},
				Symbol:            "TEST2",
				AmountInWei:       "100000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	channelPermission := &requests.CreateCommunityTokenPermission{
		CommunityID: community.ID(),
		Type:        protobuf.CommunityTokenPermission_CAN_VIEW_AND_POST_CHANNEL,
		ChatIds:     []string{chat.ID},
		TokenCriteria: []*protobuf.TokenCriteria{
			{
				Type:              protobuf.CommunityTokenType_ERC20,
				ContractAddresses: map[uint64]string{testChainID1: "0x124"},
				Symbol:            "TEST2",
				AmountInWei:       "200000000000000000000",
				Decimals:          uint64(18),
			},
		},
	}

	waitOnChannelKeyAdded := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		action, ok := sub.keyActions.ChannelKeysActions[chat.CommunityChatID()]
		if !ok || action.ActionType != communities.EncryptionKeyAdd {
			return false
		}
		_, ok = action.Members[crypto.PubkeyToHex(&s.owner.identity.PublicKey)]
		return ok
	})

	waitOnCommunityPermissionCreated := waitOnCommunitiesEvent(s.owner, func(sub *communities.Subscription) bool {
		return len(sub.Community.TokenPermissions()) == 2
	})

	response, err := s.owner.CreateCommunityTokenPermission(communityPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	response, err = s.owner.CreateCommunityTokenPermission(channelPermission)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community = response.Communities()[0]
	s.Require().True(community.HasTokenPermissions())
	s.Require().Len(community.TokenPermissions(), 2)

	err = <-waitOnCommunityPermissionCreated
	s.Require().NoError(err)
	s.Require().True(community.Encrypted())

	err = <-waitOnChannelKeyAdded
	s.Require().NoError(err)

	// 2. Owner: Send a message A
	messageText1 := RandomLettersString(10)
	message1 := s.sendChatMessage(s.owner, chat.ID, messageText1)

	// 2.2. Retrieve own message (to make it stored in the archive later)
	_, err = s.owner.RetrieveAll()
	s.Require().NoError(err)

	// 3. Owner: Create community archive
	const partition = 2 * time.Minute
	messageDate := time.UnixMilli(int64(message1.Timestamp))
	startDate := messageDate.Add(-time.Minute)
	endDate := messageDate.Add(time.Minute)
	topic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(chat.ID))
	communityCommonTopic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(community.UniversalChatID()))
	topics := []messagingtypes.ContentTopic{topic, communityCommonTopic}

	torrentConfig := params.TorrentConfig{
		Enabled:    true,
		DataDir:    os.TempDir() + "/archivedata",
		TorrentDir: os.TempDir() + "/torrents",
		Port:       0,
	}

	// Share archive directory between all users
	amc := &archivetypes.ArchiveManagerConfig{
		TorrentConfig: &torrentConfig,
	}

	s.owner.SetupArchiveManager(amc)
	s.bob.SetupArchiveManager(amc)

	s.owner.config.messengerSignalsHandler = &MessengerSignalsHandlerMock{}
	s.bob.config.messengerSignalsHandler = &MessengerSignalsHandlerMock{}

	archiveManager, ok := s.owner.archiveManager.(*archive.ArchiveManager)
	s.Require().True(ok)

	torrentBackend, err := archiveManager.GetTorrentBackend()
	s.Require().NoError(err)
	s.Require().NotNil(torrentBackend)

	archiveIDs, err := torrentBackend.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, community.Encrypted())
	s.Require().NoError(err)
	s.Require().Len(archiveIDs, 1)

	community, err = s.owner.GetCommunityByID(community.ID())
	s.Require().NoError(err)

	// 4. Bob: join community (satisfying membership, but not channel permissions)
	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, communityPermission.TokenCriteria[0])
	s.advertiseCommunityTo(community, s.bob)

	waitForKeysDistributedToBob := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		action := sub.keyActions.CommunityKeyAction
		if action.ActionType != communities.EncryptionKeySendToMembers {
			return false
		}
		_, ok := action.Members[s.bob.IdentityPublicKeyString()]
		return ok
	})

	s.joinCommunity(community, s.bob)

	err = <-waitForKeysDistributedToBob
	s.Require().NoError(err)

	// 5. Bob: Import community archive
	// The archive is successfully decrypted, but the message inside is not.
	// https://github.com/status-im/status-desktop/issues/13105 can be reproduced at this stage
	// by forcing `encryption.ErrHashRatchetGroupIDNotFound` in `ExtractMessagesFromHistoryArchive` after decryption here:
	// https://github.com/status-im/status-go/blob/6c82a6c2be7ebed93bcae3b9cf5053da3820de50/protocol/communities/manager.go#L4403

	// Ensure owner has archive

	archiveManager, ok = s.owner.archiveManager.(*archive.ArchiveManager)
	s.Require().True(ok)

	torrentBackend, err = archiveManager.GetTorrentBackend()
	s.Require().NoError(err)
	s.Require().NotNil(torrentBackend)

	archiveIndex, err := torrentBackend.LoadHistoryArchiveIndexFromFile(s.owner.identity, community.ID())
	s.Require().NoError(err)
	s.Require().Len(archiveIndex.Archives, 1)

	// Ensure bob has archive (because they share same local directory)
	archiveManager, ok = s.bob.archiveManager.(*archive.ArchiveManager)
	s.Require().True(ok)

	torrentBackend, err = archiveManager.GetTorrentBackend()
	s.Require().NoError(err)
	s.Require().NotNil(torrentBackend)

	archiveIndex, err = torrentBackend.LoadHistoryArchiveIndexFromFile(s.bob.identity, community.ID())
	s.Require().NoError(err)
	s.Require().Len(archiveIndex.Archives, 1)

	archiveHash := maps.Keys(archiveIndex.Archives)[0]

	// Save message archive ID as in
	// https://github.com/status-im/status-go/blob/6c82a6c2be7ebed93bcae3b9cf5053da3820de50/protocol/communities/manager.go#L4325-L4336
	err = s.bob.archiveManager.SaveMessageArchiveID(community.ID(), archiveHash)
	s.Require().NoError(err)

	// Import archive
	s.bob.importDelayer.once.Do(func() {
		close(s.bob.importDelayer.wait)
	})
	cancel := make(chan struct{})
	err = s.bob.importHistoryArchives(community.ID(), cancel, "")
	s.Require().NoError(err)

	// Ensure message1 wasn't imported, as it's encrypted, and we don't have access to the channel
	receivedMessage1, err := s.bob.MessageByID(message1.ID)
	s.Require().Nil(receivedMessage1)
	s.Require().Error(err)

	chatID := []byte(chat.ID)
	hashRatchetMessagesCount, err := s.bob.messaging.GetHashRatchetMessagesCountForGroup(chatID)
	s.Require().NoError(err)
	s.Require().Equal(1, hashRatchetMessagesCount)

	// Make bob satisfy channel criteria
	waitOnChannelKeyToBeDistributedToBob := s.waitOnKeyDistribution(func(sub *CommunityAndKeyActions) bool {
		action, ok := sub.keyActions.ChannelKeysActions[chat.CommunityChatID()]
		if !ok || action.ActionType != communities.EncryptionKeySendToMembers {
			return false
		}
		_, ok = action.Members[crypto.PubkeyToHex(&s.bob.identity.PublicKey)]
		return ok
	})

	s.makeAddressSatisfyTheCriteria(testChainID1, bobAddress, channelPermission.TokenCriteria[0])

	// force owner to reevaluate channel members
	// in production it will happen automatically, by periodic check
	err = s.owner.communitiesManager.ForceMembersReevaluation(community.ID())
	s.Require().NoError(err)

	err = <-waitOnChannelKeyToBeDistributedToBob
	s.Require().NoError(err)

	// Finally ensure that the message from archive was retrieved and decrypted
	response, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool {
			_, ok := r.messages[message1.ID]
			return ok
		},
		"no messages",
	)
	s.Require().NoError(err)
	s.Require().Len(response.Messages(), 1)
	s.Require().Equal(messageText1, response.Messages()[0].Text)
}
