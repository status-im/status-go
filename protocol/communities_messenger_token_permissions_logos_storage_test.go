//go:build use_logos_storage
// +build use_logos_storage

package protocol

import (
	"context"
	"log"
	"time"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/pkg/messaging"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/archive"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

func (s *MessengerCommunitiesTokenPermissionsSuite) TestUploadDownloadLogosStorageHistoryArchives() {
	// The messengers used in the tests in this suite use the helper newTestMessenger (protocol/messenger_builder_test.go). In the config setup (config.complete), tmc.nodeConfig defaults to an empty params.NodeConfig{} unless the test overrides it. The default params.NodeConfig zero-value has all nested configs (including LogosStorageConfig.Enabled) set to false.

	// During newTestMessenger, the in-memory appDb is migrated and then sDB.CreateSettings(*config.appSettings, *config.nodeConfig) is called (messenger_builder_test.go (line 120)). If you don’t override config.nodeConfig beforehand, this writes LogosStorageConfig.Enabled = false into the node-config tables—mirroring what a brand-new install would do.

	// So TestImportDecryptedLogosStorageArchiveMessages starts from that baseline: the in-memory DB contains the node config seeded with LogosStorageConfig.Enabled false (unless you explicitly mutate it in the test).

	// Following the above reasoning, we read the LogosStorage config from the database and verify that LogosStorageConfig setting are what we expect them to be.

	ownerNodeCfgFromDB, err := s.owner.settings.GetNodeConfig()
	s.Require().NoError(err)
	s.Assert().Equal(
		s.nodeConfigs[s.owner.IdentityPublicKeyString()].LogosStorageConfig,
		ownerNodeCfgFromDB.LogosStorageConfig,
	)

	bobNodeCfgFromDB, err := s.bob.settings.GetNodeConfig()
	s.Require().NoError(err)
	s.Assert().Equal(
		s.nodeConfigs[s.bob.IdentityPublicKeyString()].LogosStorageConfig,
		bobNodeCfgFromDB.LogosStorageConfig,
	)

	// now to be specific
	s.Assert().False(ownerNodeCfgFromDB.LogosStorageConfig.Enabled)
	s.Assert().False(bobNodeCfgFromDB.LogosStorageConfig.Enabled)

	s.Require().NoError(s.owner.EnableLogosStorageCommunityHistoryArchiveProtocol(nil))
	s.Require().NoError(s.bob.EnableLogosStorageCommunityHistoryArchiveProtocol(nil))

	ownerNodeCfgFromDB2, err := s.owner.settings.GetNodeConfig()
	s.Require().NoError(err)
	bobNodeCfgFromDB2, err := s.bob.settings.GetNodeConfig()
	s.Require().NoError(err)

	s.Assert().True(ownerNodeCfgFromDB2.LogosStorageConfig.Enabled)
	s.Assert().True(bobNodeCfgFromDB2.LogosStorageConfig.Enabled)

	// get LogosStorage client for owner - cast to concrete type
	ownerArchiveManager, ok := s.owner.archiveManager.(*archive.ArchiveManager)
	s.Require().True(ok)

	ownerBackend, err := ownerArchiveManager.GetLogosStorageBackend()
	s.Require().NoError(err)
	s.Require().NotNil(ownerBackend)
	ownerClient := ownerBackend.GetClient()
	s.Require().NotNil(ownerClient)

	// get PeerId of the owner:
	ownerInfo, err := ownerClient.Debug()
	s.Require().NoError(err)
	s.Require().NotNil(ownerInfo)

	bobArchiveManager, ok := s.bob.archiveManager.(*archive.ArchiveManager)
	s.Require().True(ok)

	bobBackend, err := bobArchiveManager.GetLogosStorageBackend()
	s.Require().NoError(err)
	s.Require().NotNil(bobBackend)
	bobClient := bobBackend.GetClient()
	s.Require().NotNil(bobClient)

	err = bobClient.Connect(ownerInfo.ID, ownerInfo.Addrs)
	s.Require().NoError(err)

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

	log.Println("Message sent with ID:", message1.ID)

	// 3. Owner: Create community archive
	const partition = 2 * time.Minute
	messageDate := time.UnixMilli(int64(message1.Timestamp))
	startDate := messageDate.Add(-time.Minute)
	endDate := messageDate.Add(time.Minute)
	topic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(chat.ID))
	communityCommonTopic := messagingtypes.BytesToContentTopic(messaging.ToContentTopic(community.UniversalChatID()))
	topics := []messagingtypes.ContentTopic{topic, communityCommonTopic}

	s.owner.config.messengerSignalsHandler = &MessengerSignalsHandlerMock{}
	s.bob.config.messengerSignalsHandler = &MessengerSignalsHandlerMock{}

	// this will create archive and push it to LogosStorage
	archiveIDs, err := s.owner.archiveManager.CreateHistoryArchiveFromDB(community.ID(), topics, startDate, endDate, partition, community.Encrypted())
	s.Require().NoError(err)
	s.Require().Len(archiveIDs, 1)

	// Ensure owner has archive
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	indexCid, err := s.owner.communitiesManager.GetLastSeenArchiveLink(community.ID())
	s.Require().NoError(err)
	s.Require().NotEmpty(indexCid)
	archiveIndex, err := ownerBackend.LoadHistoryArchiveIndex(ctx, s.owner.identity, community.ID(), indexCid, true)
	s.Require().NoError(err)
	s.Require().Len(archiveIndex.Archives, 1)

	indexCid, err = s.owner.communitiesManager.GetLastSeenArchiveLink(community.ID())
	s.Require().NoError(err)

	// log
	s.T().Logf("LogosStorage archive OWNER index CID: %s", indexCid)

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

	// Let's trigger actual download from bob's LogosStorage node
	cancelChan := make(chan struct{})
	defer close(cancelChan)
	s.bob.importDelayer.once.Do(func() {
		close(s.bob.importDelayer.wait)
	})

	s.bob.ratchetNotFoundDelay = 1 * time.Second

	s.bob.downloadAndImportHistoryArchives(community.ID(), indexCid, cancelChan)

	// Ensure bob has archive
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	indexCid, err = s.bob.communitiesManager.GetLastSeenArchiveLink(community.ID())
	s.Require().NoError(err)
	s.Require().NotEmpty(indexCid)
	archiveIndex, err = bobBackend.LoadHistoryArchiveIndex(ctx, s.bob.identity, community.ID(), indexCid, true)
	s.Require().NoError(err)
	s.Require().Len(archiveIndex.Archives, 1)

	// log
	s.T().Logf("LogosStorage archive BOB index CID: %s", indexCid)

	// Ensure message1 wasn't imported, as it's encrypted, and we don't have access to the channel
	receivedMessage1, err := s.bob.MessageByID(message1.ID)
	s.Require().Nil(receivedMessage1)
	s.Require().Error(err)

	chatID := []byte(chat.ID)
	s.Require().Eventually(func() bool {
		count, err := s.bob.messaging.GetHashRatchetMessagesCountForGroup(chatID)
		s.Require().NoError(err)
		return count == 1
	}, 30*time.Second, 500*time.Millisecond)

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
