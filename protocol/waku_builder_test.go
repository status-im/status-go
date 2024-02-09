package protocol

import (
	"database/sql"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/status-im/status-go/appdatabase"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common/shard"
	"github.com/status-im/status-go/t/helpers"
	waku2 "github.com/status-im/status-go/wakuv2"
)

type testWakuV2Config struct {
	logger                 *zap.Logger
	enableStore            bool
	useShardAsDefaultTopic bool
	clusterID              uint16
}

func NewWakuV2(s *suite.Suite, cfg testWakuV2Config) *waku2.Waku {
	wakuConfig := &waku2.Config{
		UseShardAsDefaultTopic: cfg.useShardAsDefaultTopic,
		ClusterID:              cfg.clusterID,
	}

	var db *sql.DB

	if cfg.enableStore {
		var err error
		db, err = helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
		s.Require().NoError(err)

		wakuConfig.EnableStore = true
		wakuConfig.StoreCapacity = 200
		wakuConfig.StoreSeconds = 200
	}

	wakuNode, err := waku2.New(
		"",
		"",
		wakuConfig,
		cfg.logger,
		db,
		nil,
		nil,
		nil)

	s.Require().NoError(err)

	err = wakuNode.Start()
	s.Require().NoError(err)

	return wakuNode
}

func CreateWakuV2Network(s *suite.Suite, parentLogger *zap.Logger, useShardAsDefaultTopic bool, nodeNames []string) []types.Waku {
	nodes := make([]*waku2.Waku, len(nodeNames))
	wrappers := make([]types.Waku, len(nodes))

	for i, name := range nodeNames {
		nodes[i] = NewWakuV2(s, testWakuV2Config{
			logger:                 parentLogger.Named("waku-" + name),
			enableStore:            false,
			useShardAsDefaultTopic: useShardAsDefaultTopic,
			clusterID:              shard.UndefinedShardValue, // FIXME: why it was 0 here?
		})
	}

	// Setup local network graph
	for i := 0; i < len(nodes); i++ {
		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			addrs := nodes[j].ListenAddresses()
			s.Require().Greater(len(addrs), 0)
			_, err := nodes[i].AddRelayPeer(addrs[0])
			s.Require().NoError(err)
			err = nodes[i].DialPeer(addrs[0])
			s.Require().NoError(err)
		}
	}
	for i, n := range nodes {
		wrappers[i] = gethbridge.NewGethWakuV2Wrapper(n)
	}
	return wrappers
}
