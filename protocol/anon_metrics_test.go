// +build postgres
// TODO(samyoul) remove this `+build postgres` line ^^^ once docker testing in Jenkins is working good

// In order to run these tests, you must run a PostgreSQL database.
//
// Using Docker:
//   docker run -e POSTGRES_HOST_AUTH_METHOD=trust -d -p 5432:5432 postgres:9.6-alpine
//

package protocol

import (
	"context"
	"crypto/ecdsa"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	bindata "github.com/status-im/migrate/v4/source/go_bindata"

	appmetricsDB "github.com/status-im/status-go/appmetrics"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/postgres"
	"github.com/status-im/status-go/protocol/anonmetrics"
	"github.com/status-im/status-go/protocol/anonmetrics/migrations"
	"github.com/status-im/status-go/protocol/tt"
	"github.com/status-im/status-go/services/appmetrics"
	"github.com/status-im/status-go/waku"
)

func TestMessengerAnonMetricsSuite(t *testing.T) {
	suite.Run(t, new(MessengerAnonMetricsSuite))
}

type MessengerAnonMetricsSuite struct {
	suite.Suite
	alice *Messenger // client instance of Messenger
	bob   *Messenger // server instance of Messenger

	aliceKey *ecdsa.PrivateKey // private key for the alice instance of Messenger
	bobKey   *ecdsa.PrivateKey // private key for the bob instance of Messenger

	// If one wants to send messages between different instances of Messenger,
	// a single Waku service should be shared.
	shh    types.Waku
	logger *zap.Logger
}

func (s *MessengerAnonMetricsSuite) SetupSuite() {
	// ResetDefaultTestPostgresDB Required to completely reset the Postgres DB
	err := postgres.ResetDefaultTestPostgresDB()
	s.NoError(err)
}

func (s *MessengerAnonMetricsSuite) SetupTest() {
	var err error

	s.logger = tt.MustCreateTestLogger()

	// Setup Waku things
	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start())

	// Generate private keys for Alice and Bob
	s.aliceKey, err = crypto.GenerateKey()
	s.Require().NoError(err)

	s.bobKey, err = crypto.GenerateKey()
	s.Require().NoError(err)

	// Generate Alice Messenger as the client
	amcc := &anonmetrics.ClientConfig{
		ShouldSend:  true,
		SendAddress: &s.bobKey.PublicKey,
		Active:      anonmetrics.ActiveClientPhrase,
	}
	s.alice, err = newMessengerWithKey(s.shh, s.aliceKey, s.logger, []Option{WithAnonMetricsClientConfig(amcc)})
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	// Generate Bob Messenger as the Server
	amsc := &anonmetrics.ServerConfig{
		Enabled:     true,
		PostgresURI: postgres.DefaultTestURI,
		Active:      anonmetrics.ActiveServerPhrase,
	}
	s.bob, err = newMessengerWithKey(s.shh, s.bobKey, s.logger, []Option{WithAnonMetricsServerConfig(amsc)})
	s.Require().NoError(err)

	_, err = s.bob.Start()
	s.Require().NoError(err)
}

func (s *MessengerAnonMetricsSuite) TearDownTest() {
	// Down migrate the DB
	if s.bob.anonMetricsServer != nil {
		postgresMigration := bindata.Resource(migrations.AssetNames(), migrations.Asset)
		m, err := anonmetrics.MakeMigration(s.bob.anonMetricsServer.PostgresDB, postgresMigration)
		s.NoError(err)

		err = m.Down()
		s.NoError(err)
	}

	// Shutdown messengers
	s.NoError(s.alice.Shutdown())
	s.alice = nil
	s.NoError(s.bob.Shutdown())
	s.bob = nil
	_ = s.logger.Sync()
}

func (s *MessengerAnonMetricsSuite) TestReceiveAnonMetric() {
	// Create the appmetrics API to simulate incoming metrics from `status-react`
	ama := appmetrics.NewAPI(appmetricsDB.NewDB(s.alice.database))

	// Generate and store some metrics to Alice
	ams := appmetricsDB.GenerateMetrics(10)
	err := ama.SaveAppMetrics(context.Background(), ams)
	s.Require().NoError(err)

	// Check that we have what we stored
	amsdb, err := ama.GetAppMetrics(context.Background(), 100, 0)
	s.Require().NoError(err)
	s.Require().Len(amsdb.AppMetrics, 10)

	// Wait for messages to arrive at bob
	_, err = WaitOnMessengerResponse(
		s.bob,
		func(r *MessengerResponse) bool { return len(r.AnonymousMetrics) > 0 },
		"no anonymous metrics received",
	)
	s.Require().NoError(err)

	// Get app metrics from postgres DB
	bobMetrics, err := s.bob.anonMetricsServer.GetAppMetrics(100, 0)
	s.Require().NoError(err)
	s.Require().Len(bobMetrics, 5)

	// Check the values of received and stored metrics against the broadcast metrics
	for i, bobMetric := range bobMetrics {
		s.Require().True(bobMetric.CreatedAt.Equal(amsdb.AppMetrics[i].CreatedAt), "created_at values are equal")
		s.Require().Exactly(bobMetric.SessionID, amsdb.AppMetrics[i].SessionID, "session_id matched exactly")
		s.Require().Exactly(bobMetric.Value, amsdb.AppMetrics[i].Value, "value matches exactly")
		s.Require().Exactly(bobMetric.Event, amsdb.AppMetrics[i].Event, "event matches exactly")
		s.Require().Exactly(bobMetric.OS, amsdb.AppMetrics[i].OS, "operating system matches exactly")
		s.Require().Exactly(bobMetric.AppVersion, amsdb.AppMetrics[i].AppVersion, "app version matches exactly")
	}
}

// TestActivationIsOff tests if using the incorrect activation phrase for the anon metric client / server deactivates
// the client / server. This test can be removed when / if the anon metrics functionality is reintroduced / re-approved.
func (s *MessengerAnonMetricsSuite) TestActivationIsOff() {
	var err error

	// Check the set up messengers are in the expected state with the correct activation phrases
	s.NotNil(s.alice.anonMetricsClient)
	s.NotNil(s.bob.anonMetricsServer)

	// Generate Alice Messenger as the client with an incorrect phrase
	amcc := &anonmetrics.ClientConfig{
		ShouldSend:  true,
		SendAddress: &s.bobKey.PublicKey,
		Active:      "the wrong client phrase",
	}
	s.alice, err = newMessengerWithKey(s.shh, s.aliceKey, s.logger, []Option{WithAnonMetricsClientConfig(amcc)})
	s.NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.Nil(s.alice.anonMetricsClient)

	// Generate Alice Messenger as the client with an no activation phrase
	amcc = &anonmetrics.ClientConfig{
		ShouldSend:  true,
		SendAddress: &s.bobKey.PublicKey,
	}
	s.alice, err = newMessengerWithKey(s.shh, s.aliceKey, s.logger, []Option{WithAnonMetricsClientConfig(amcc)})
	s.NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	s.Nil(s.alice.anonMetricsClient)

	// Generate Bob Messenger as the Server with an incorrect phrase
	amsc := &anonmetrics.ServerConfig{
		Enabled:     true,
		PostgresURI: postgres.DefaultTestURI,
		Active:      "the wrong server phrase",
	}
	s.bob, err = newMessengerWithKey(s.shh, s.bobKey, s.logger, []Option{WithAnonMetricsServerConfig(amsc)})
	s.Require().NoError(err)

	s.Nil(s.bob.anonMetricsServer)

	// Generate Bob Messenger as the Server with no activation phrase
	amsc = &anonmetrics.ServerConfig{
		Enabled:     true,
		PostgresURI: postgres.DefaultTestURI,
	}
	s.bob, err = newMessengerWithKey(s.shh, s.bobKey, s.logger, []Option{WithAnonMetricsServerConfig(amsc)})
	s.Require().NoError(err)

	s.Nil(s.bob.anonMetricsServer)
}
