package statusgo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/ethereum/go-ethereum/log"
	gethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/metrics"
	nodemetrics "github.com/status-im/status-go/metrics/node"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	protocol "github.com/status-im/status-go/protocol"
	"github.com/status-im/status-go/signal"
)

func StartDesktopNode(configJSON string) string {
	config, err := params.NewConfigFromJSON(configJSON)
	if err != nil {
		log.Error(err.Error())
		return makeJSONResponse(err)
	}

	config.RegisterTopics = append(config.RegisterTopics, params.WhisperDiscv5Topic)

	// We want statusd to be distinct from StatusIM client.
	config.Name = "DesktopNode"

	err = statusBackend.AccountManager().InitKeystore(config.KeyStoreDir)
	if err != nil {
		log.Error("Failed to init keystore", "error", err)
		return makeJSONResponse(err)
	}
	err = statusBackend.StartNode(config)
	if err != nil {
		log.Error("Node start failed", "error", err)
		return makeJSONResponse(err)
	}

	if config.NodeKey == "" {
		err := errors.New("node key is empty")
		log.Error(err.Error())
		return makeJSONResponse(err)
	}

	identity, err := crypto.HexToECDSA(config.NodeKey)
	if err != nil {
		log.Error("node key is invalid", "error", err)
		return makeJSONResponse(err)
	}

	// Generate installationID from public key, so it's always the same
	installationID, err := uuid.FromBytes(crypto.CompressPubkey(&identity.PublicKey)[:16])
	if err != nil {
		log.Error("cannot create installation id", "error", err)
		return makeJSONResponse(err)
	}

	db, err := appdatabase.InitializeDB(config.DataDir+"/"+installationID.String()+".db", "")
	if err != nil {
		log.Error("failed to initialize app db", "error", err)
		return makeJSONResponse(err)
	}

	messenger, err := protocol.NewMessenger(identity, statusBackend.StatusNode().NodeBridge(), installationID.String(), protocol.WithDatabase(db))
	if err != nil {
		log.Error("failed to create messenger", "error", err)
		return makeJSONResponse(err)
	}

	err = messenger.Init()
	if err != nil {
		log.Error("failed to init messenger", "error", err)
		return makeJSONResponse(err)
	}

	_, err = messenger.Start()
	if err != nil {
		log.Error("failed to start messenger", "error", err)
		return makeJSONResponse(err)
	}

	api.RunAsync(func() error {
		interruptCh := make(chan struct{})

		go startCollectingNodeMetrics(interruptCh, statusBackend.StatusNode())
		go gethmetrics.CollectProcessMetrics(3 * time.Second)
		go metrics.NewMetricsServer(9090, gethmetrics.DefaultRegistry).Listen()

		go retrieveStats(messenger, 5*time.Second, interruptCh)
		gethNode := statusBackend.StatusNode().GethNode()
		if gethNode != nil {
			// wait till node has been stopped
			gethNode.Wait()
			close(interruptCh)
		}
		return nil
	})

	return makeJSONResponse(nil)
}

// startCollectingStats collects various stats about the node and other protocols like Whisper.
func startCollectingNodeMetrics(interruptCh <-chan struct{}, statusNode *node.StatusNode) {
	log.Info("Starting collecting node metrics")

	gethNode := statusNode.GethNode()
	if gethNode == nil {
		log.Error("Failed to run metrics because it could not get the node")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		// Try to subscribe and collect metrics. In case of an error, retry.
		for {
			if err := nodemetrics.SubscribeServerEvents(ctx, gethNode); err != nil {
				log.Error("Failed to subscribe server events", "error", err)
			} else {
				// no error means that the subscription was terminated by purpose
				return
			}

			time.Sleep(time.Second)
		}
	}()

	<-interruptCh
}

func StopNode() string {
	return makeJSONResponse(statusBackend.StopNode())
}

func retrieveStats(messenger *protocol.Messenger, tick time.Duration, cancel <-chan struct{}) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			response := messenger.GetStats()
			signal.SendStats(response)
		case <-cancel:
			return
		}
	}
}
