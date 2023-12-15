package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/status-im/status-go/appdatabase"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/t/helpers"

	mailserversDB "github.com/status-im/status-go/services/mailservers"
	waku2 "github.com/status-im/status-go/wakuv2"
)

const (
	localFleet        = "local-test-fleet-1"
	localMailserverID = "local-test-mailserver"
)

func TestMessengerStoreNodeRequestSuite(t *testing.T) {
	suite.Run(t, new(MessengerStoreNodeRequestSuite))
}

type MessengerStoreNodeRequestSuite struct {
	suite.Suite

	owner *Messenger
	bob   *Messenger

	wakuStoreNode *waku2.Waku
	ownerWaku     types.Waku
	bobWaku       types.Waku

	logger *zap.Logger
}

func (s *MessengerStoreNodeRequestSuite) newMessenger(shh types.Waku, logger *zap.Logger, mailserverAddress string) *Messenger {
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	mailserversSQLDb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)
	err = sqlite.Migrate(mailserversSQLDb) // migrate default
	s.Require().NoError(err)

	mailserversDatabase := mailserversDB.NewDB(mailserversSQLDb)
	err = mailserversDatabase.Add(mailserversDB.Mailserver{
		ID:      localMailserverID,
		Name:    localMailserverID,
		Address: mailserverAddress,
		Fleet:   localFleet,
	})
	s.Require().NoError(err)

	options := []Option{
		WithMailserversDatabase(mailserversDatabase),
		WithClusterConfig(params.ClusterConfig{
			Fleet: localFleet,
		}),
	}

	messenger, err := newMessengerWithKey(shh, privateKey, logger, options)

	s.Require().NoError(err)
	return messenger
}

func (s *MessengerStoreNodeRequestSuite) SetupTest() {
	s.logger = tt.MustCreateTestLogger()

	// Create store node

	storeNodeLogger := s.logger.With(zap.String("name", "store-node-waku"))
	s.wakuStoreNode = NewWakuV2(&s.Suite, storeNodeLogger, true, true)

	storeNodeListenAddresses := s.wakuStoreNode.ListenAddresses()
	s.Require().LessOrEqual(1, len(storeNodeListenAddresses))

	storeNodeAddress := storeNodeListenAddresses[0]
	s.logger.Info("store node ready", zap.String("address", storeNodeAddress))

	// Create community owner

	ownerWakuLogger := s.logger.With(zap.String("name", "owner-waku"))
	s.ownerWaku = gethbridge.NewGethWakuV2Wrapper(NewWakuV2(&s.Suite, ownerWakuLogger, true, false))

	ownerLogger := s.logger.With(zap.String("name", "owner"))
	s.owner = s.newMessenger(s.ownerWaku, ownerLogger, storeNodeAddress)

	// Create an independent user

	bobWakuLogger := s.logger.With(zap.String("name", "owner-waku"))
	s.bobWaku = gethbridge.NewGethWakuV2Wrapper(NewWakuV2(&s.Suite, bobWakuLogger, true, false))

	bobLogger := s.logger.With(zap.String("name", "bob"))
	s.bob = s.newMessenger(s.bobWaku, bobLogger, storeNodeAddress)
	s.bob.StartRetrieveMessagesLoop(time.Second, nil)

	// Connect owner to storenode so message is stored
	err := s.ownerWaku.DialPeer(storeNodeAddress)
	s.Require().NoError(err)
}

func (s *MessengerStoreNodeRequestSuite) TestRequestCommunityInfo() {
	WaitForAvailableStoreNode(&s.Suite, s.owner, time.Second)

	createCommunityRequest := &requests.CreateCommunity{
		Name:        "panda-lovers",
		Description: "we love pandas",
		Membership:  protobuf.CommunityPermissions_AUTO_ACCEPT,
		Color:       "#ff0000",
		Tags:        []string{"Web3"},
	}

	response, err := s.owner.CreateCommunity(createCommunityRequest, true)
	s.Require().NoError(err)
	s.Require().NotNil(response)
	s.Require().Len(response.Communities(), 1)

	community := response.Communities()[0]
	communityID := community.IDString()

	WaitForAvailableStoreNode(&s.Suite, s.bob, time.Second)

	request := FetchCommunityRequest{
		CommunityKey:    communityID,
		Shard:           nil,
		TryDatabase:     false,
		WaitForResponse: true,
	}

	fetchedCommunity, err := s.bob.FetchCommunity(&request)
	s.Require().NoError(err)
	s.Require().NotNil(fetchedCommunity)
	s.Require().Equal(communityID, fetchedCommunity.IDString())
}
