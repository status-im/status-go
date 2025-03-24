package protocol

import (
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/waku/types"
)

func WithTestStoreNode(s *suite.Suite, id string, address multiaddr.Multiaddr, fleet string, collectiblesServiceMock *CollectiblesServiceMock) Option {
	return func(c *config) error {
		sqldb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
		s.Require().NoError(err)

		db := mailservers.NewDB(sqldb)
		err = db.Add(types.Mailserver{
			ID:    id,
			Name:  id,
			Addr:  &address,
			Fleet: fleet,
		})
		s.Require().NoError(err)

		c.mailserversDatabase = db
		c.clusterConfig = params.ClusterConfig{Fleet: fleet}
		c.communityTokensService = collectiblesServiceMock

		return nil
	}
}

func WithAutoRequestHistoricMessages(enabled bool) Option {
	return func(c *config) error {
		c.codeControlFlags.AutoRequestHistoricMessages = enabled
		return nil
	}
}

func WithCuratedCommunitiesUpdateLoop(enabled bool) Option {
	return func(c *config) error {
		c.codeControlFlags.CuratedCommunitiesUpdateLoopEnabled = enabled
		return nil
	}
}

func WithCommunityManagerOptions(options []communities.ManagerOption) Option {
	return func(c *config) error {
		c.communityManagerOptions = options
		return nil
	}
}

func WithStubOnlineChecker() Option {
	return func(c *config) error {
		c.onlineChecker = func() bool {
			return true
		}
		return nil
	}
}
