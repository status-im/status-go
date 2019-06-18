package chatapi

import (
	"crypto/ecdsa"
	"fmt"
	"path/filepath"

	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/shhext/chat"
	whisper "github.com/status-im/whisper/whisperv6"

	"github.com/status-im/status-console-client/protocol/adapter"
	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/transport"
)

const (
	chatSQLFileName        = "chat.sql"
	pfsSQLFileNameV1Format = "pfs_v1.%s.sql"
)

// Make sure that Service implements node.Service interface.
var _ gethnode.Service = (*Service)(nil)

type Config struct {
	Mailservers    []string
	DataDir        string // ShhextConfig.BackupDisabledDataDir
	PFSEnabled     bool   // ShhextConfig.PFSEnabled
	InstallationID string // ShhextConfig.InstalationID
}

// Service represents our own implementation of personal sign operations.
type Service struct {
	config Config
	messenger *client.Messenger
}

// New returns a new Service.
func New(c Config) *Service {
	return &Service{
		config: c,
	}
}

// Init initialize the service. An account must be already selected
// before calling this method.
// This method can be called multiple times with different keys.
func (s *Service) Init(
	node transport.StatusNode,
	shh *whisper.Whisper,
	shhExt *shhext.Service,
	pk *ecdsa.PrivateKey,
	encryptionKey string,
) error {
	dbPath := filepath.Join(s.config.DataDir, chatSQLFileName)
	db, err := client.InitializeDB(dbPath, encryptionKey)
	if err != nil {
		return err
	}

	trnsp := transport.NewWhisperServiceTransport(node, s.config.Mailservers, shh, shhExt, pk)
	pfs, err := pfsFactory(s.config.PFSEnabled, s.config.DataDir, s.config.InstallationID, encryptionKey)
	if err != nil {
		return err
	}
	protocolAdapter := adapter.NewProtocolWhisperAdapter(trnsp, pfs)

	if s.messenger != nil {
		s.messenger.Stop()
	}
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

func pfsFactory(enabled bool, baseDir, installationID, encKey string) (*chat.ProtocolService, error) {
	if !enabled {
		return nil, nil
	}

	dbPath := filepath.Join(baseDir, fmt.Sprintf(pfsSQLFileNameV1Format, installationID))
	persistence, err := chat.NewSQLLitePersistence(dbPath, encKey)
	if err != nil {
		return nil, err
	}

	return chat.NewProtocolService(
		chat.NewEncryptionService(
			persistence,
			chat.DefaultEncryptionServiceConfig(installationID),
		),
		addedBundlesHandler,
	), nil
}

func addedBundlesHandler(addedBundles []chat.IdentityAndIDPair) {
	handler := shhext.EnvelopeSignalHandler{}
	for _, bundle := range addedBundles {
		handler.BundleAdded(bundle[0], bundle[1])
	}
}
