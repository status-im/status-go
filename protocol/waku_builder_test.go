package protocol

import (
	"crypto/ecdsa"
	"database/sql"

	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/wakuv2"
)

type testWakuV2Config struct {
	logger      *zap.Logger
	enableStore bool
	clusterID   uint16
	nodekey     []byte
}

func NewTestWakuV2(s *suite.Suite, cfg testWakuV2Config) *wakuv2.Waku {
	wakuConfig := &wakuv2.Config{
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

	wakuNode, err := wakuv2.New(
		nodeKey,
		wakuConfig,
		cfg.logger,
		db,
		nil,
		nil,
		nil)

	s.Require().NoError(err)

	err = wakuNode.Start()
	if cfg.enableStore {
		err := wakuNode.SubscribeToPubsubTopic(wakuv2.DefaultNonProtectedPubsubTopic(), nil)
		s.Require().NoError(err)
	}
	s.Require().NoError(err)

	return wakuNode
}
