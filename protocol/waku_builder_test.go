package protocol

import (
	"crypto/ecdsa"
	"database/sql"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/appdatabase"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/t/helpers"
	waku2 "github.com/status-im/status-go/wakuv2"
)

type testWakuV2Config struct {
	logger      *zap.Logger
	enableStore bool
	clusterID   uint16
	nodekey     []byte
}

func NewTestWakuV2(s *suite.Suite, cfg testWakuV2Config) *waku2.Waku {
	wakuConfig := &waku2.Config{
		ClusterID:                cfg.clusterID,
		LightClient:              false,
		EnablePeerExchangeServer: true,
		EnablePeerExchangeClient: false,
		EnableDiscV5:             false,
	}

	var nodeKey *ecdsa.PrivateKey
	if len(cfg.nodekey) != 0 {
		nodeKey, _ = crypto.ToECDSA(cfg.nodekey)
	}

	var db *sql.DB
	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	s.Require().NoError(err)

	if cfg.enableStore {
		wakuConfig.EnableStore = true
		wakuConfig.StoreCapacity = 200
		wakuConfig.StoreSeconds = 200
	}

	wakuNode, err := waku2.New(
		nodeKey,
		"",
		wakuConfig,
		cfg.logger,
		db,
		nil,
		nil,
		nil)

	s.Require().NoError(err)

	err = wakuNode.Start()
	if cfg.enableStore {
		err := wakuNode.SubscribeToPubsubTopic(waku2.DefaultNonProtectedPubsubTopic(), nil)
		s.Require().NoError(err)
	}
	s.Require().NoError(err)

	return wakuNode
}

func CreateWakuV2Network(s *suite.Suite, parentLogger *zap.Logger, nodeNames []string) []types.Waku {
	nodes := make([]*waku2.Waku, len(nodeNames))
	wrappers := make([]types.Waku, len(nodes))

	for i, name := range nodeNames {
		nodes[i] = NewTestWakuV2(s, testWakuV2Config{
			logger:      parentLogger.Named("waku-" + name),
			enableStore: false,
			clusterID:   waku2.MainStatusShardCluster,
		})
	}

	// Setup local network graph
	for i := 0; i < len(nodes); i++ {
		for j := 0; j < len(nodes); j++ {
			if i == j {
				continue
			}

			addrs, err := nodes[j].ListenAddresses()
			s.Require().NoError(err)
			s.Require().Greater(len(addrs), 0)
			_, err = nodes[i].AddRelayPeer(addrs[0])
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
