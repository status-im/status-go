package chatapi

import (
	"crypto/ecdsa"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/services/shhext"
	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/status-im/status-console-client/protocol/adapter"
	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/transport"
)

// Make sure that Service implements node.Service interface.
var _ gethnode.Service = (*Service)(nil)

// Service represents our own implementation of personal sign operations.
type Service struct {
	mailservers []string
	dataDir     string
	shh         *whisper.Whisper
	shhExt      *shhext.Service

	messenger *client.Messenger
}

// New returns a new Service.
func New(mailservers []string, dataDir string, shh *whisper.Whisper, shhExt *shhext.Service) *Service {
	return &Service{
		mailservers: mailservers,
		dataDir:     dataDir,
		shh:         shh,
		shhExt:      shhExt,
	}
}

// Init initialize the service. An account must be already selected
// before calling this method.
func (s *Service) Init(pk *ecdsa.PrivateKey, node transport.StatusNode) error {
	// TODO(adam): replace it with a proper key
	dbKey := crypto.PubkeyToAddress(pk.PublicKey).String()
	dbPath := filepath.Join(s.dataDir, "chat.sql")
	db, err := client.InitializeDB(dbPath, dbKey)
	if err != nil {
		return err
	}

	trnsp := transport.NewWhisperServiceTransport(node, s.mailservers, s.shh, s.shhExt, pk)
	protocolAdapter := adapter.NewProtocolWhisperAdapter(trnsp, nil)
	s.messenger = client.NewMessenger(pk, protocolAdapter, db)
	return nil
}

// Protocols returns a new protocols list. In this case, there are none.
func (s *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns a list of new APIs.
func (s *Service) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "status",
			Version:   "1.0",
			Service:   NewPrivateAPI(s),
			Public:    true,
		},
	}
}

// Start is run when a service is started.
func (s *Service) Start(server *p2p.Server) error {
	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	// TODO: should be able to stop Messenger
	return nil
}
