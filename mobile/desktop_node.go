package statusgo

import (
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/eth-node/crypto"
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

	interruptCh := make(chan struct{})

	_, err = messenger.Start()
	if err != nil {
		log.Error("failed to start messenger", "error", err)
		return makeJSONResponse(err)
	}

	go retrieveStats(messenger, 5*time.Second, interruptCh)

	gethNode := statusBackend.StatusNode().GethNode()
	if gethNode != nil {
		// wait till node has been stopped
		gethNode.Wait()
		close(interruptCh)
	}

	return makeJSONResponse(nil)
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
