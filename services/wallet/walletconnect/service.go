package walletconnect

import (
	"fmt"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc/network"
	"github.com/status-im/status-go/transactions"
)

type Service struct {
	networkManager *network.Manager
	accountsDB     *accounts.Database
	eventFeed      *event.Feed

	transactor  *transactions.Transactor
	gethManager *account.GethManager
}

func NewService(networkManager *network.Manager, accountsDB *accounts.Database, transactor *transactions.Transactor, gethManager *account.GethManager, eventFeed *event.Feed) *Service {
	return &Service{
		networkManager: networkManager,
		accountsDB:     accountsDB,
		eventFeed:      eventFeed,
		transactor:     transactor,
		gethManager:    gethManager,
	}
}

func (s *Service) PairSessionProposal(proposal SessionProposal) (*PairSessionResponse, error) {
	namespace := Namespace{
		Methods: []string{params.SendTransactionMethodName, params.PersonalSignMethodName},
		Events:  []string{"accountsChanged", "chainChanged"},
	}

	proposedChains := proposal.Params.RequiredNamespaces.Eip155.Chains
	chains, eipChains := sessionProposalToSupportedChain(proposedChains, func(chainID uint64) bool {
		return s.networkManager.Find(chainID) != nil
	})
	if len(chains) != len(proposedChains) {
		log.Warn("Some chains are not supported; wanted: ", proposedChains, "; supported: ", chains)
		return nil, ErrorChainsNotSupported
	}
	namespace.Chains = eipChains

	activeAccounts, err := s.accountsDB.GetActiveAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to get active accounts: %w", err)
	}

	// Filter out non-own accounts
	usableAccounts := make([]*accounts.Account, 0, 1)
	for _, acc := range activeAccounts {
		if !acc.IsOwnAccount() || acc.Operable != accounts.AccountFullyOperable {
			continue
		}
		usableAccounts = append(usableAccounts, acc)
	}

	addresses := activeToOwnedAccounts(usableAccounts)
	namespace.Accounts = caip10Accounts(addresses, chains)

	// TODO #12434: respond async
	return &PairSessionResponse{
		SessionProposal: proposal,
		SupportedNamespaces: Namespaces{
			Eip155: namespace,
		},
	}, nil
}

func (s *Service) SessionRequest(request SessionRequest, hashedPassword string) (response *SessionRequestResponse, err error) {
	// TODO #12434: should we check topic for validity? It might make sense if we
	// want to cache the paired sessions

	if request.Params.Request.Method == params.SendTransactionMethodName {
		return s.sendTransaction(request, hashedPassword)
	} else if request.Params.Request.Method == params.PersonalSignMethodName {
		return s.personalSign(request, hashedPassword)
	}

	// TODO #12434: respond async
	return nil, ErrorMethodNotSupported
}
