package notificationserver

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/p2p"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/params"
)

// TO DO(rgeraldes)
// Restore sessions after reviewing the storage model

const (
	datadirDefaultSessionStore = "sessionstore"
	topicServerDiscovery       = "/server/discover"
	topicServerAcceptance      = "/server/accept"
)

// NotificationServer -
type NotificationServer struct {
	*whisper.Server                   // whisper server
	Store                             // session store
	Notifier                          // firebase client
	serverID        string            // server ID
	protocolKey     *ecdsa.PrivateKey // shared protocol secret
}

// Init initializes the notification server
func (s *NotificationServer) Init(w *whisper.Whisper, config *params.WhisperConfig) error {
	notificationConfig := config.NotificationServerNode
	protocolKey, err := config.ReadIdentityFile()
	if err != nil {
		return err
	}
	notifier, err := NewFirebaseNotifier(notificationConfig.FirebaseConfig)
	if err != nil {
		return err
	}
	storage, err := NewStore(config.DataDir)
	if err != nil {
		return err
	}

	s.Server = whisper.NewWhisperServer(w)
	s.protocolKey = protocolKey
	s.Notifier = notifier
	s.Store = storage

	if err := s.HandleFunc(topicServerDiscovery, s.protocolKey, s.ServerDiscovery); err != nil {
		return err
	}
	if err := s.HandleFunc(topicServerAcceptance, s.protocolKey, s.ServerAcceptance); err != nil {
		return err
	}

	return nil
}

// Start launches the whisper server activities
func (s *NotificationServer) Start(stack *p2p.Server) error {
	nodeInfo := stack.NodeInfo()
	s.serverID = nodeInfo.ID
	s.ListenAndServe()
	return nil
}

// Stop terminates the whisper server activities
func (s *NotificationServer) Stop() error {
	s.Server.Stop()
	return nil
}
