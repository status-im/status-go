package wakuv2

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/timesource"
	"github.com/status-im/status-go/wakuv2"
	wakuv2common "github.com/status-im/status-go/wakuv2/common"
)

type Service struct {
	waku   *wakuv2.Waku
	cancel context.CancelFunc
	wg     *sync.WaitGroup
}

func NewService(nodeConfig *params.NodeConfig, appDB *sql.DB, timeSource *timesource.NTPTimeSource) (*Service, error) {
	cfg := &wakuv2.Config{
		MaxMessageSize:                         wakuv2common.DefaultMaxMessageSize,
		Host:                                   nodeConfig.WakuV2Config.Host,
		Port:                                   nodeConfig.WakuV2Config.Port,
		LightClient:                            nodeConfig.WakuV2Config.LightClient,
		WakuNodes:                              nodeConfig.ClusterConfig.WakuNodes,
		EnableStore:                            nodeConfig.WakuV2Config.EnableStore,
		StoreCapacity:                          nodeConfig.WakuV2Config.StoreCapacity,
		StoreSeconds:                           nodeConfig.WakuV2Config.StoreSeconds,
		DiscoveryLimit:                         nodeConfig.WakuV2Config.DiscoveryLimit,
		DiscV5BootstrapNodes:                   nodeConfig.ClusterConfig.DiscV5BootstrapNodes,
		Nameserver:                             nodeConfig.WakuV2Config.Nameserver,
		UDPPort:                                nodeConfig.WakuV2Config.UDPPort,
		AutoUpdate:                             nodeConfig.WakuV2Config.AutoUpdate,
		DefaultShardPubsubTopic:                wakuv2.DefaultShardPubsubTopic(),
		TelemetryServerURL:                     nodeConfig.WakuV2Config.TelemetryServerURL,
		ClusterID:                              nodeConfig.ClusterConfig.ClusterID,
		EnableMissingMessageVerification:       nodeConfig.WakuV2Config.EnableMissingMessageVerification,
		EnableStoreConfirmationForMessagesSent: nodeConfig.WakuV2Config.EnableStoreConfirmationForMessagesSent,
		UseThrottledPublish:                    true,
	}

	// Configure peer exchange and discv5 settings based on node type
	if cfg.LightClient {
		cfg.EnablePeerExchangeServer = false
		cfg.EnablePeerExchangeClient = true
		cfg.EnableDiscV5 = false
	} else {
		cfg.EnablePeerExchangeServer = true
		cfg.EnablePeerExchangeClient = false
		cfg.EnableDiscV5 = true
	}

	if nodeConfig.WakuV2Config.MaxMessageSize > 0 {
		cfg.MaxMessageSize = nodeConfig.WakuV2Config.MaxMessageSize
	}

	var nodeKey *ecdsa.PrivateKey
	var err error
	if nodeConfig.NodeKey != "" {
		nodeKey, err = crypto.HexToECDSA(nodeConfig.NodeKey)
		if err != nil {
			return nil, fmt.Errorf("could not convert nodekey into a valid private key: %v", err)
		}
	} else {
		nodeKeyStr := os.Getenv("WAKUV2_NODE_KEY")
		if nodeKeyStr != "" {
			nodeKeyBytes, err := hexutil.Decode(nodeKeyStr)
			if err != nil {
				return nil, fmt.Errorf("failed to decode the go-waku private key: %v", err)
			}

			nodeKey, err = crypto.ToECDSA(nodeKeyBytes)
			if err != nil {
				return nil, fmt.Errorf("could not convert nodekey into a valid private key: %v", err)
			}
		}
	}

	w, err := wakuv2.New(
		context.Background(),
		nodeKey,
		cfg,
		logutils.ZapLogger(),
		appDB,
		timeSource,
		signal.SendHistoricMessagesRequestFailed,
		signal.SendPeerStats,
	)

	if err != nil {
		return nil, err
	}

	return &Service{
		waku: w,
	}, nil
}

func (s *Service) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg = &sync.WaitGroup{}

	s.wg.Add(1)
	go func() {
		defer common.LogOnPanic()
		defer s.wg.Done()
		s.waku.Start(ctx) // FIXME: check error
	}()

	return nil
}

func (s *Service) Stop() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

// APIs returns the RPC descriptors the Waku implementation offers
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: wakuv2.Name,
			Version:   wakuv2.VersionStr,
			Service:   wakuv2.NewPublicWakuAPI(s.waku),
			Public:    false,
		},
	}
}

func (s *Service) Waku() *wakuv2.Waku {
	return s.waku
}
