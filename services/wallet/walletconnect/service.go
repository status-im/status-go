package walletconnect

import (
	"github.com/ethereum/go-ethereum/event"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc/network"
)

type Service struct {
	networkManager *network.Manager
	accountsDB     *accounts.Database
	eventFeed      *event.Feed
}

func NewService(networkManager *network.Manager, accountsDB *accounts.Database, eventFeed *event.Feed) *Service {
	return &Service{
		networkManager: networkManager,
		accountsDB:     accountsDB,
		eventFeed:      eventFeed,
	}
}

func (s *Service) PairSessionProposal(proposal SessionProposal) (*PairSessionResponse, error) {
	namespace := Namespace{
		Methods: []string{"eth_sendTransaction", "personal_sign"},
		Events:  []string{"accountsChanged", "chainChanged"},
	}

	proposedChains := proposal.Params.RequiredNamespaces.Eip155.Chains
	chains, eipChains := sessionProposalToSupportedChain(proposedChains, func(chainID uint64) bool {
		return s.networkManager.Find(chainID) != nil
	})
	if len(chains) != len(proposedChains) {
		return nil, ErrorChainsNotSupported
	}
	namespace.Chains = eipChains

	activeAccounts, err := s.accountsDB.GetActiveAccounts()
	if err != nil {
		return nil, ErrorChainsNotSupported
	}

	addresses := activeToOwnedAccounts(activeAccounts)
	namespace.Accounts = caip10Accounts(addresses, chains)

	// TODO #12434: respond async
	return &PairSessionResponse{
		SessionProposal: proposal,
		SupportedNamespaces: Namespaces{
			Eip155: namespace,
		},
	}, nil
}
