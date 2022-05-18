package wallet

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/services/wallet/transfer"
)

func NewReader(s *Service) *Reader {
	return &Reader{s}
}

type Reader struct {
	s *Service
}

type ReaderToken struct {
	Token         *Token       `json:"token"`
	OraclePrice   float64      `json:"oraclePrice"`
	CryptoBalance *hexutil.Big `json:"cryptoBalance"`
	FiatBalance   *big.Float   `json:"fiatBalance"`
}

type ReaderAccount struct {
	Account      *accounts.Account              `json:"account"`
	Collections  map[uint64][]OpenseaCollection `json:"collections"`
	Tokens       map[uint64][]ReaderToken       `json:"tokens"`
	Transactions map[uint64][]transfer.View     `json:"transactions"`

	FiatBalance *big.Float `json:"fiatBalance"`
}

type Wallet struct {
	Accounts            []ReaderAccount                  `json:"accounts"`
	Favorites           []Favourite                      `json:"favorites"`
	OnRamp              []CryptoOnRamp                   `json:"onRamp"`
	SavedAddresses      map[uint64][]SavedAddress        `json:"savedAddresses"`
	Tokens              map[uint64][]*Token              `json:"tokens"`
	CustomTokens        []*Token                         `json:"customTokens"`
	PendingTransactions map[uint64][]*PendingTransaction `json:"pendingTransactions"`

	FiatBalance *big.Float `json:"fiatBalance"`
	Currency    string     `json:"currency"`
}

func getAddresses(accounts []*accounts.Account) []common.Address {
	addresses := make([]common.Address, len(accounts))
	for _, account := range accounts {
		addresses = append(addresses, common.Address(account.Address))
	}
	return addresses
}

func (r *Reader) buildReaderAccount(
	ctx context.Context,
	chainIDs []uint64,
	account *accounts.Account,
	visibleTokens map[uint64][]*Token,
	prices map[string]float64,
	balances map[common.Address]*hexutil.Big,
) (ReaderAccount, error) {
	limit := (*hexutil.Big)(big.NewInt(20))
	toBlock := (*hexutil.Big)(big.NewInt(0))

	collections := make(map[uint64][]OpenseaCollection)
	tokens := make(map[uint64][]ReaderToken)
	transactions := make(map[uint64][]transfer.View)
	accountFiatBalance := big.NewFloat(0)
	for _, chainID := range chainIDs {
		client, err := newOpenseaClient(chainID, r.s.openseaAPIKey)
		if err == nil {
			c, _ := client.fetchAllCollectionsByOwner(common.Address(account.Address))
			collections[chainID] = c
		}

		for _, token := range visibleTokens[chainID] {
			oraclePrice := prices[token.Symbol]
			cryptoBalance := balances[token.Address].ToInt()
			fiatBalance := big.NewFloat(0).Mul(big.NewFloat(oraclePrice), new(big.Float).SetInt(cryptoBalance))
			tokens[chainID] = append(tokens[chainID], ReaderToken{
				Token:         token,
				OraclePrice:   oraclePrice,
				CryptoBalance: balances[token.Address],
				FiatBalance:   fiatBalance,
			})
			accountFiatBalance = accountFiatBalance.Add(accountFiatBalance, fiatBalance)
		}
		t, err := r.s.transferController.GetTransfersByAddress(ctx, chainID, common.Address(account.Address), toBlock, limit, false)
		if err == nil {
			transactions[chainID] = t
		}
	}
	return ReaderAccount{
		Account:      account,
		Collections:  collections,
		Tokens:       tokens,
		Transactions: transactions,
		FiatBalance:  accountFiatBalance,
	}, nil
}

func (r *Reader) Start(ctx context.Context, chainIDs []uint64) error {
	accounts, err := r.s.accountsDB.GetAccounts()
	if err != nil {
		return err
	}

	return r.s.transferController.CheckRecentHistory(chainIDs, getAddresses(accounts))
}

func (r *Reader) GetWallet(ctx context.Context, chainIDs []uint64) (*Wallet, error) {
	currency, err := r.s.accountsDB.GetCurrency()
	if err != nil {
		return nil, err
	}

	tokensMap := make(map[uint64][]*Token)
	for _, chainID := range chainIDs {
		tokens, err := r.s.tokenManager.getTokens(chainID)
		if err != nil {
			return nil, err
		}
		tokensMap[chainID] = tokens
	}

	customTokens, err := r.s.tokenManager.getCustoms()
	if err != nil {
		return nil, err
	}

	visibleTokens, err := r.s.tokenManager.getVisible(chainIDs)
	if err != nil {
		return nil, err
	}

	tokenAddresses := make([]common.Address, 0)
	tokenSymbols := make([]string, 0)
	for _, tokens := range visibleTokens {
		for _, token := range tokens {
			tokenAddresses = append(tokenAddresses, token.Address)
			tokenSymbols = append(tokenSymbols, token.Symbol)
		}
	}

	accounts, err := r.s.accountsDB.GetAccounts()
	if err != nil {
		return nil, err
	}

	prices, err := fetchCryptoComparePrices(tokenSymbols, currency)
	if err != nil {
		return nil, err
	}

	clients, err := chain.NewClients(r.s.rpcClient, chainIDs)
	if err != nil {
		return nil, err
	}

	balances, err := r.s.tokenManager.getBalances(ctx, clients, getAddresses(accounts), tokenAddresses)
	if err != nil {
		return nil, err
	}

	readerAccounts := make([]ReaderAccount, len(accounts))
	walletFiatBalance := big.NewFloat(0)
	for i, account := range accounts {
		readerAccount, err := r.buildReaderAccount(
			ctx,
			chainIDs,
			account,
			visibleTokens,
			prices,
			balances[common.Address(account.Address)],
		)
		if err != nil {
			return nil, err
		}
		walletFiatBalance = walletFiatBalance.Add(walletFiatBalance, readerAccount.FiatBalance)
		readerAccounts[i] = readerAccount
	}

	savedAddressesMap := make(map[uint64][]SavedAddress)
	for _, chainID := range chainIDs {
		savedAddresses, err := r.s.savedAddressesManager.GetSavedAddresses(chainID)
		if err != nil {
			return nil, err
		}
		savedAddressesMap[chainID] = savedAddresses
	}

	onRamp, err := r.s.cryptoOnRampManager.Get()
	if err != nil {
		return nil, err
	}

	favorites, err := r.s.favouriteManager.GetFavourites()
	if err != nil {
		return nil, err
	}

	pendingTransactions := make(map[uint64][]*PendingTransaction)
	for _, chainID := range chainIDs {
		pendingTx, err := r.s.transactionManager.getAllPendings(chainID)
		if err != nil {
			return nil, err
		}
		pendingTransactions[chainID] = pendingTx
	}
	return &Wallet{
		Accounts:            readerAccounts,
		Favorites:           favorites,
		OnRamp:              onRamp,
		SavedAddresses:      savedAddressesMap,
		Tokens:              tokensMap,
		CustomTokens:        customTokens,
		PendingTransactions: pendingTransactions,
		Currency:            currency,
		FiatBalance:         walletFiatBalance,
	}, nil
}
