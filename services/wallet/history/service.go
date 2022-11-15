package history

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	statustypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	statusrpc "github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/rpc/network"

	"github.com/status-im/status-go/services/wallet/chain"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

// EventBalanceHistoryUpdateStarted and EventBalanceHistoryUpdateDone are used to notify the UI that balance history is being updated
const (
	EventBalanceHistoryUpdateStarted           walletevent.EventType = "wallet-balance-history-update-started"
	EventBalanceHistoryUpdateFinished          walletevent.EventType = "wallet-balance-history-update-finished"
	EventBalanceHistoryUpdateFinishedWithError walletevent.EventType = "wallet-balance-history-update-finished-with-error"

	balanceHistoryUpdateInterval = 12 * time.Hour
)

type Service struct {
	balance        *Balance
	db             *sql.DB
	eventFeed      *event.Feed
	rpcClient      *statusrpc.Client
	networkManager *network.Manager
	tokenManager   *token.Manager
	serviceContext context.Context
	cancelFn       context.CancelFunc

	timer                    *time.Timer
	visibleTokenSymbols      []string
	visibleTokenSymbolsMutex sync.Mutex // Protects access to visibleSymbols
}

func NewService(db *sql.DB, eventFeed *event.Feed, rpcClient *statusrpc.Client, tokenManager *token.Manager) *Service {
	return &Service{
		balance:        NewBalance(NewBalanceDB(db)),
		db:             db,
		eventFeed:      eventFeed,
		rpcClient:      rpcClient,
		networkManager: rpcClient.NetworkManager,
		tokenManager:   tokenManager,
	}
}

func (s *Service) Stop() {
	if s.cancelFn != nil {
		s.cancelFn()
	}
}

func (s *Service) triggerEvent(eventType walletevent.EventType, account statustypes.Address, message string) {
	s.eventFeed.Send(walletevent.Event{
		Type: eventType,
		Accounts: []common.Address{
			common.Address(account),
		},
		Message: message,
	})
}

func (s *Service) StartBalanceHistory() {
	go func() {
		s.serviceContext, s.cancelFn = context.WithCancel(context.Background())
		s.timer = time.NewTimer(balanceHistoryUpdateInterval)

		update := func() (exit bool) {
			err := s.updateBalanceHistoryForAllEnabledNetworks(s.serviceContext)
			if s.serviceContext.Err() != nil {
				s.triggerEvent(EventBalanceHistoryUpdateFinished, statustypes.Address{}, "Service canceled")
				s.timer.Stop()
				return true
			}
			if err != nil {
				s.triggerEvent(EventBalanceHistoryUpdateFinishedWithError, statustypes.Address{}, err.Error())
			}
			return false
		}

		if update() {
			return
		}

		for range s.timer.C {
			s.resetTimer(balanceHistoryUpdateInterval)

			if update() {
				return
			}
		}
	}()
}

func (s *Service) resetTimer(interval time.Duration) {
	if s.timer != nil {
		s.timer.Stop()
		s.timer.Reset(interval)
	}
}

func (s *Service) UpdateVisibleTokens(symbols []string) {
	s.visibleTokenSymbolsMutex.Lock()
	defer s.visibleTokenSymbolsMutex.Unlock()

	startUpdate := len(s.visibleTokenSymbols) == 0 && len(symbols) > 0
	s.visibleTokenSymbols = symbols
	if startUpdate {
		s.resetTimer(0)
	}
}

func (s *Service) isTokenVisible(tokenSymbol string) bool {
	s.visibleTokenSymbolsMutex.Lock()
	defer s.visibleTokenSymbolsMutex.Unlock()

	for _, visibleSymbol := range s.visibleTokenSymbols {
		if visibleSymbol == tokenSymbol {
			return true
		}
	}
	return false
}

// Native token implementation of DataSource interface
type chainClientSource struct {
	chainClient *chain.Client
	currency    string
}

func (src *chainClientSource) HeaderByNumber(ctx context.Context, blockNo *big.Int) (*types.Header, error) {
	return src.chainClient.HeaderByNumber(ctx, blockNo)
}

func (src *chainClientSource) BalanceAt(ctx context.Context, account common.Address, blockNo *big.Int) (*big.Int, error) {
	return src.chainClient.BalanceAt(ctx, account, blockNo)
}

func (src *chainClientSource) ChainID() uint64 {
	return src.chainClient.ChainID
}

func (src *chainClientSource) Currency() string {
	return src.currency
}

func (src *chainClientSource) TimeNow() int64 {
	return time.Now().UTC().Unix()
}

type tokenChainClientSource struct {
	chainClientSource
	TokenManager   *token.Manager
	NetworkManager *network.Manager

	firstUnavailableBlockNo *big.Int
}

func (src *tokenChainClientSource) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	network := src.NetworkManager.Find(src.chainClient.ChainID)
	if network == nil {
		return nil, errors.New("network not found")
	}
	token := src.TokenManager.FindToken(network, src.currency)
	if token == nil {
		return nil, errors.New("token not found")
	}
	if src.firstUnavailableBlockNo != nil && blockNumber.Cmp(src.firstUnavailableBlockNo) < 0 {
		return big.NewInt(0), nil
	}
	balance, err := src.TokenManager.GetTokenBalanceAt(ctx, src.chainClient, account, token.Address, blockNumber)
	if err != nil {
		if err.Error() == "no contract code at given address" {
			// Ignore requests before contract deployment and mark this state for future requests
			src.firstUnavailableBlockNo = new(big.Int).Set(blockNumber)
			return big.NewInt(0), nil
		}
		return nil, err
	}
	return balance, err
}

// GetBalanceHistory returns token count balance
// TODO: fetch token to FIAT exchange rates and return FIAT balance
func (s *Service) GetBalanceHistory(ctx context.Context, chainIDs []uint64, address common.Address, currency string, endTimestamp int64, timeInterval TimeInterval) ([]*DataPoint, error) {
	allData := make(map[uint64][]*DataPoint)
	for _, chainID := range chainIDs {
		data, err := s.balance.get(ctx, chainID, currency, address, endTimestamp, timeInterval)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			allData[chainID] = data
		}
	}

	return mergeDataPoints(allData)
}

// mergeDataPoints merges same block numbers from different chains which are incompatible due to different timelines
// TODO: use time-based intervals instead of block numbers
func mergeDataPoints(data map[uint64][]*DataPoint) ([]*DataPoint, error) {
	if len(data) == 0 {
		return make([]*DataPoint, 0), nil
	}

	pos := make(map[uint64]int)
	for k := range data {
		pos[k] = 0
	}

	res := make([]*DataPoint, 0)
	done := false
	for !done {
		var minNo *big.Int
		var timestamp uint64
		// Take the smallest block number
		for k := range data {
			blockNo := new(big.Int).Set(data[k][pos[k]].BlockNumber.ToInt())
			if minNo == nil {
				minNo = new(big.Int).Set(blockNo)
				// We use it only if we have a full match
				timestamp = data[k][pos[k]].Timestamp
			} else if blockNo.Cmp(minNo) < 0 {
				minNo.Set(blockNo)
			}
		}
		// If all chains have the same block number sum it; also increment the processed position
		sumOfAll := big.NewInt(0)
		for k := range data {
			cur := data[k][pos[k]]
			if cur.BlockNumber.ToInt().Cmp(minNo) == 0 {
				pos[k]++
				if sumOfAll != nil {
					sumOfAll.Add(sumOfAll, cur.Value.ToInt())
				}
			} else {
				sumOfAll = nil
			}
		}
		// If sum of all make sense add it to the result otherwise ignore it
		if sumOfAll != nil {
			// TODO: convert to FIAT value
			res = append(res, &DataPoint{
				BlockNumber: (*hexutil.Big)(minNo),
				Timestamp:   timestamp,
				Value:       (*hexutil.Big)(sumOfAll),
			})
		}

		// Check if we reached the end of any chain
		for k := range data {
			if pos[k] == len(data[k]) {
				done = true
				break
			}
		}
	}
	return res, nil
}

// updateBalanceHistoryForAllEnabledNetworks iterates over all enabled and supported networks for the s.visibleTokenSymbol
// and updates the balance history for the given address
//
// expects ctx to have cancellation support and processing to be cancelled by the caller
func (s *Service) updateBalanceHistoryForAllEnabledNetworks(ctx context.Context) error {
	accountsDB, err := accounts.NewDB(s.db)
	if err != nil {
		return err
	}

	addresses, err := accountsDB.GetWalletAddresses()
	if err != nil {
		return err
	}

	networks, err := s.networkManager.Get(true)
	if err != nil {
		return err
	}

	for _, address := range addresses {
		s.triggerEvent(EventBalanceHistoryUpdateStarted, address, "")

		for _, network := range networks {
			tokensForChain, err := s.tokenManager.GetTokens(network.ChainID)
			if err != nil {
				tokensForChain = make([]*token.Token, 0)
			}
			tokensForChain = append(tokensForChain, s.tokenManager.ToToken(network))

			for _, token := range tokensForChain {
				if !s.isTokenVisible(token.Symbol) {
					continue
				}

				var dataSource DataSource
				chainClient, err := chain.NewClient(s.rpcClient, network.ChainID)
				if err != nil {
					return err
				}
				if token.IsNative() {
					dataSource = &chainClientSource{chainClient, token.Symbol}
				} else {
					dataSource = &tokenChainClientSource{
						chainClientSource: chainClientSource{
							chainClient: chainClient,
							currency:    token.Symbol,
						},
						TokenManager:   s.tokenManager,
						NetworkManager: s.networkManager,
					}
				}

				for currentInterval := int(BalanceHistoryAllTime); currentInterval >= int(BalanceHistory7Days); currentInterval-- {
					select {
					case <-ctx.Done():
						return errors.New("context cancelled")
					default:
					}
					err = s.balance.update(ctx, dataSource, common.Address(address), TimeInterval(currentInterval))
					if err != nil {
						log.Warn("Error updating balance history", "chainID", dataSource.ChainID(), "currency", dataSource.Currency(), "address", address.String(), "interval", currentInterval, "err", err)
					}
				}
			}
		}
		s.triggerEvent(EventBalanceHistoryUpdateFinished, address, "")
	}
	return nil
}
