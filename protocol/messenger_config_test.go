package protocol

import (
	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/t/helpers"
)

func WithTestStoreNode(s *suite.Suite, id string, address string, fleet string, collectiblesServiceMock *CollectiblesServiceMock) Option {
	return func(c *config) error {
		sqldb, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
		s.Require().NoError(err)

		db := mailservers.NewDB(sqldb)
		err = db.Add(mailservers.Mailserver{
			ID:      id,
			Name:    id,
			Address: address,
			Fleet:   fleet,
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
