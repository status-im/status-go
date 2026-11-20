package protocol

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/protobuf"
	"github.com/status-im/status-go/pkg/messaging"
	wakutypes "github.com/status-im/status-go/pkg/messaging/waku/types"
)

func TestStoreNodeRequestCommunityDescriptionFreshness(t *testing.T) {
	request := &storeNodeRequest{
		minimumDataClock: 10,
		config:           defaultStoreNodeRequestConfig(),
	}

	require.False(t, request.hasFoundCommunityDescription(10))
	require.True(t, request.hasFoundCommunityDescription(11))

	request.config = buildStoreNodeRequestConfig([]StoreNodeRequestOption{
		WithRequireNewerCommunityDescription(false),
	})
	require.True(t, request.hasFoundCommunityDescription(10))
}

func TestStoreNodeRequestPageCheckSuite(t *testing.T) {
	suite.Run(t, new(StoreNodeRequestPageCheckSuite))
}

type StoreNodeRequestPageCheckSuite struct {
	suite.Suite

	messagingEnv *messaging.TestMessagingEnvironment
	m            *Messenger
	communityKey *ecdsa.PrivateKey
}

func (s *StoreNodeRequestPageCheckSuite) SetupTest() {
	var err error
	s.messagingEnv, err = messaging.NewTestMessagingEnvironment()
	s.Require().NoError(err)
	s.Require().NoError(s.messagingEnv.Setup(s.T()))

	s.m, err = newTestMessenger(s.T(), s.messagingEnv, testMessengerConfig{})
	s.Require().NoError(err)

	s.communityKey, err = crypto.GenerateKey()
	s.Require().NoError(err)
}

func (s *StoreNodeRequestPageCheckSuite) TearDownTest() {
	releaseTestMessenger(s.T(), s.m)
}

func (s *StoreNodeRequestPageCheckSuite) communityID() string {
	return types.EncodeHex(crypto.CompressPubkey(&s.communityKey.PublicKey))
}

func (s *StoreNodeRequestPageCheckSuite) storeCommunity(clock uint64, memberCount int) {
	members := make(map[string]*protobuf.CommunityMember, memberCount)
	for i := 0; i < memberCount; i++ {
		members[fmt.Sprintf("0x04%0128x", i)] = &protobuf.CommunityMember{
			Roles: []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_NONE},
		}
	}

	description := &protobuf.CommunityDescription{
		Clock:   clock,
		Members: members,
	}
	controlNode, err := communities.New(communities.Config{
		MemberIdentity:       s.m.identity,
		PrivateKey:           s.communityKey,
		ControlNode:          &s.communityKey.PublicKey,
		ControlDevice:        true,
		ID:                   &s.communityKey.PublicKey,
		CommunityDescription: description,
	}, s.m.getTimesource(), nil, nil)
	s.Require().NoError(err)
	wrappedDescription, err := controlNode.ToProtocolMessageBytes()
	s.Require().NoError(err)

	// Stored the way a community received from the network is: not controlled by us.
	community, err := communities.New(communities.Config{
		MemberIdentity:                      s.m.identity,
		ControlNode:                         &s.communityKey.PublicKey,
		ID:                                  &s.communityKey.PublicKey,
		CommunityDescription:                description,
		CommunityDescriptionProtocolMessage: wrappedDescription,
	}, s.m.getTimesource(), nil, nil)
	s.Require().NoError(err)
	s.Require().NoError(s.m.communitiesManager.GetPersistence().SaveCommunity(community))
}

// fetchCommunity runs a community store node request whose store pages are
// played by pages, which gets the request's per-page check.
func (s *StoreNodeRequestPageCheckSuite) fetchCommunity(pages func(pageCheck func(int) (bool, uint64))) *communities.Community {
	s.messagingEnv.SetProcessMailserverBatchHook(func(ctx context.Context, batch wakutypes.MailserverBatch, pageLimit uint64, shouldProcessNextPage func(int) (bool, uint64), processEnvelopes bool) error {
		pages(shouldProcessNextPage)
		return nil
	})
	s.T().Cleanup(func() {
		s.messagingEnv.SetProcessMailserverBatchHook(nil)
	})

	community, _, err := s.m.storeNodeRequestsManager.FetchCommunity(context.Background(), s.communityID(), nil)
	s.Require().NoError(err)
	return community
}

func (s *StoreNodeRequestPageCheckSuite) TestPagesUntilCommunityIsStored() {
	community := s.fetchCommunity(func(pageCheck func(int) (bool, uint64)) {
		next, pageSize := pageCheck(1)
		s.True(next)
		s.Equal(uint64(defaultStoreNodeRequestPageSize), pageSize)

		s.storeCommunity(5, 10)

		next, _ = pageCheck(1)
		s.False(next)
	})

	s.Require().NotNil(community)
	s.Require().Equal(s.communityID(), community.IDString())
	s.Require().Equal(uint64(5), community.Clock())
}

func (s *StoreNodeRequestPageCheckSuite) TestPagesUntilStoredCommunityIsNewer() {
	s.storeCommunity(5, 10)

	community := s.fetchCommunity(func(pageCheck func(int) (bool, uint64)) {
		next, _ := pageCheck(1)
		s.True(next)

		s.storeCommunity(6, 10)

		next, _ = pageCheck(1)
		s.False(next)
	})

	s.Require().NotNil(community)
	s.Require().Equal(uint64(6), community.Clock())
}

func (s *StoreNodeRequestPageCheckSuite) TestReturnsNoCommunityWhenNoneIsNewer() {
	s.storeCommunity(5, 10)

	community := s.fetchCommunity(func(pageCheck func(int) (bool, uint64)) {
		next, _ := pageCheck(1)
		s.True(next)
	})

	s.Require().Nil(community)
}

// A request whose page check failed returns no community, also when an
// earlier page had found it.
func (s *StoreNodeRequestPageCheckSuite) TestReturnsNoCommunityWhenLaterPageFails() {
	s.messagingEnv.SetProcessMailserverBatchHook(func(ctx context.Context, batch wakutypes.MailserverBatch, pageLimit uint64, pageCheck func(int) (bool, uint64), processEnvelopes bool) error {
		s.storeCommunity(5, 10)
		next, _ := pageCheck(1)
		s.False(next)

		// The next page check cannot tell whether the community awaits validation.
		_, err := s.m.database.Exec("ALTER TABLE communities_validate_signer RENAME TO communities_validate_signer_gone")
		s.NoError(err)
		next, _ = pageCheck(1)
		s.False(next)
		return nil
	})
	s.T().Cleanup(func() {
		s.messagingEnv.SetProcessMailserverBatchHook(nil)
	})

	community, _, _ := s.m.storeNodeRequestsManager.FetchCommunity(context.Background(), s.communityID(), nil)
	s.Require().Nil(community)
}

// The error of a failed page check is the error of the request, also when
// the store node query itself ended well.
func (s *StoreNodeRequestPageCheckSuite) TestKeepsPageCheckErrorWhenQuerySucceeds() {
	s.messagingEnv.SetProcessMailserverBatchHook(func(ctx context.Context, batch wakutypes.MailserverBatch, pageLimit uint64, pageCheck func(int) (bool, uint64), processEnvelopes bool) error {
		// The page check cannot tell whether the community awaits validation.
		_, err := s.m.database.Exec("ALTER TABLE communities_validate_signer RENAME TO communities_validate_signer_gone")
		s.NoError(err)
		next, _ := pageCheck(1)
		s.False(next)
		return nil
	})
	s.T().Cleanup(func() {
		s.messagingEnv.SetProcessMailserverBatchHook(nil)
	})

	result, err := s.m.storeNodeRequestsManager.requestCommunity(context.Background(), s.communityID(), defaultStoreNodeRequestConfig())
	s.Require().NoError(err)
	s.Require().Nil(result.community)
	s.Require().ErrorContains(result.err, "failed to validate community by ID")
}

// A page that does not bring the awaited description must not cost a load of
// the stored community.
func (s *StoreNodeRequestPageCheckSuite) TestPageCheckDoesNotLoadStoredCommunity() {
	const pages = 20
	s.storeCommunity(5, 5000)

	var allocated uint64
	s.fetchCommunity(func(pageCheck func(int) (bool, uint64)) {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for i := 0; i < pages; i++ {
			next, _ := pageCheck(1)
			s.True(next)
		}
		runtime.ReadMemStats(&after)
		allocated = after.TotalAlloc - before.TotalAlloc
	})

	s.T().Logf("allocated per page check: %d bytes", allocated/pages)
	// Loading the community costs over 7 MB. Reading its clock costs the copy
	// of the stored description the database driver makes, 0.7 MB here; the
	// budget leaves room for what the rest of the process allocates meanwhile.
	s.Require().Less(allocated/pages, uint64(2*1024*1024))
}
