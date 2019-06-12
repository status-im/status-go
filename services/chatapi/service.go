package chatapi

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	gethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/node"

	"github.com/status-im/status-console-client/protocol/adapters"
	"github.com/status-im/status-console-client/protocol/client"
)

// Make sure that Service implements node.Service interface.
var _ gethnode.Service = (*Service)(nil)

// Service represents our own implementation of personal sign operations.
type Service struct {
	node           *node.StatusNode
	accountManager *account.Manager
	messenger      *client.Messenger
}

// New returns a new Service.
func New(node *node.StatusNode, accountManager *account.Manager) (*Service, error) {
	return &Service{
		node:           node,
		accountManager: accountManager,
	}, nil
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
	// get the current account's private key
	chatAccount, err := s.accountManager.SelectedChatAccount()
	if err != nil {
		return err
	}
	privateKey := chatAccount.AccountKey.PrivateKey

	// setup database
	// TODO(adam): replace it with a proper key
	dbKey := crypto.PubkeyToAddress(privateKey.PublicKey).String()
	dbPath := filepath.Join(s.node.Config().DataDir, "chat.sql")
	db, err := client.InitializeDB(dbPath, dbKey)
	if err != nil {
		return err
	}

	shhService, err := s.node.WhisperService()
	if err != nil {
		return err
	}
	adapter := adapters.NewWhisperServiceAdapter(s.node, shhService, privateKey)
	s.messenger = client.NewMessenger(privateKey, adapter, db)
	return nil
}

// Stop is run when a service is stopped.
func (s *Service) Stop() error {
	// TODO: should be able to stop Messenger
	return nil
}
