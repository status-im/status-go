// In order to run these tests, you must run a PostgreSQL database.
//
// Using Docker:
//   docker run --name anonmetrics-db -e POSTGRES_USER=anonmetrics -e POSTGRES_PASSWORD=mysecretpassword -e POSTGRES_DB=anonmetrics -d -p 5432:5432 postgres:9.6-alpine
//

package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	appmetricsDB "github.com/status-im/status-go/appmetrics"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/anonmetrics"
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

func (s *MessengerAnonMetricsSuite) resetAnonMetricsDB(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM app_metrics")
	return err
}

func (s *MessengerAnonMetricsSuite) SetupTest() {
	var err error

	s.logger = tt.MustCreateTestLogger()

	// Setup Waku things
	config := waku.DefaultConfig
	config.MinimumAcceptedPoW = 0
	shh := waku.New(&config, s.logger)
	s.shh = gethbridge.NewGethWakuWrapper(shh)
	s.Require().NoError(shh.Start(nil))

	// Generate private keys for Alice and Bob
	s.aliceKey, err = crypto.GenerateKey()
	s.Require().NoError(err)

	s.bobKey, err = crypto.GenerateKey()
	s.Require().NoError(err)

	// Generate Alice Messenger as the client
	amcc := &anonmetrics.ClientConfig{
		ShouldSend:  true,
		SendAddress: &s.bobKey.PublicKey,
	}
	s.alice, err = newMessengerWithKey(s.shh, s.aliceKey, s.logger, []Option{WithAnonMetricsClientConfig(amcc)})
	s.Require().NoError(err)
	_, err = s.alice.Start()
	s.Require().NoError(err)

	// Generate Bob Messenger as the Server
	amsc := &anonmetrics.ServerConfig{
		Enabled:     true,
		PostgresURI: "postgres://anonmetrics:mysecretpassword@127.0.0.1:5432/anonmetrics?sslmode=disable",
	}
	s.bob, err = newMessengerWithKey(s.shh, s.bobKey, s.logger, []Option{WithAnonMetricsServerConfig(amsc)})
	s.Require().NoError(err)

	err = s.resetAnonMetricsDB(s.bob.anonMetricsServer.PostgresDB)
	s.Require().NoError(err)

	_, err = s.bob.Start()
	s.Require().NoError(err)
}

func (s *MessengerAnonMetricsSuite) TearDownTest() {
	s.Require().NoError(s.alice.Shutdown())
	s.Require().NoError(s.bob.Shutdown())
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
	s.Require().Len(amsdb, 10)

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
	for _, bobMetric := range bobMetrics {
		s.Require().True(bobMetric.CreatedAt.Equal(amsdb[0].CreatedAt), "created_at values are equal")
		s.Require().Exactly(bobMetric.SessionID, amsdb[0].SessionID, "session_id matched exactly")
		s.Require().Exactly(bobMetric.Value, amsdb[0].Value, "value matches exactly")
		s.Require().Exactly(bobMetric.Event, amsdb[0].Event, "event matches exactly")
		s.Require().Exactly(bobMetric.OS, amsdb[0].OS, "operating system matches exactly")
		s.Require().Exactly(bobMetric.AppVersion, amsdb[0].AppVersion, "app version matches exactly")
	}
}
