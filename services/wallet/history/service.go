package history

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"sort"
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

type chainIdentity uint64

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
	allData := make(map[chainIdentity][]*DataPoint)
	for _, chainID := range chainIDs {
		data, err := s.balance.get(ctx, chainID, currency, address, endTimestamp, timeInterval)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			allData[chainIdentity(chainID)] = data
		}
	}

	return mergeDataPoints(allData, strideDuration(timeInterval))
}

// mergeDataPoints merges close in time block numbers. Drops the ones that are not in a stride duration
// this should improve merging balance data from different chains which are incompatible due to different timelines
// and block length
func mergeDataPoints(data map[chainIdentity][]*DataPoint, stride time.Duration) ([]*DataPoint, error) {
	if len(data) == 0 {
		return make([]*DataPoint, 0), nil
	}

	pos := make(map[chainIdentity]int)
	for k := range data {
		pos[k] = 0
	}

	res := make([]*DataPoint, 0)
	strideStart := findFirstStrideWindow(data, stride)
	for {
		strideEnd := strideStart + int64(stride.Seconds())
		// - Gather all points in the stride window starting with current pos
		var strideIdentities map[chainIdentity][]timeIdentity
		strideIdentities, pos = dataInStrideWindowAndNextPos(data, pos, strideEnd)

		// Check if all chains have data
		strideComplete := true
		for k := range data {
			_, strideComplete = strideIdentities[k]
			if !strideComplete {
				break
			}
		}
		if strideComplete {
			chainMaxBalance := make(map[chainIdentity]*DataPoint)
			for chainID, identities := range strideIdentities {
				for _, identity := range identities {
					_, exists := chainMaxBalance[chainID]
					if exists && (*big.Int)(identity.dataPoint(data).Value).Cmp((*big.Int)(chainMaxBalance[chainID].Value)) <= 0 {
						continue
					}
					chainMaxBalance[chainID] = identity.dataPoint(data)
				}
			}
			balance := big.NewInt(0)
			for _, chainBalance := range chainMaxBalance {
				balance.Add(balance, (*big.Int)(chainBalance.Value))
			}
			res = append(res, &DataPoint{
				Timestamp:   uint64(strideEnd),
				Value:       (*hexutil.Big)(balance),
				BlockNumber: (*hexutil.Big)(getBlockID(chainMaxBalance)),
			})
		}

		if allPastEnd(data, pos) {
			return res, nil
		}

		strideStart = strideEnd
	}
}

func getBlockID(chainBalance map[chainIdentity]*DataPoint) *big.Int {
	var res *big.Int
	for _, balance := range chainBalance {
		if res == nil {
			res = new(big.Int).Set(balance.BlockNumber.ToInt())
		} else if res.Cmp(balance.BlockNumber.ToInt()) != 0 {
			return nil
		}
	}

	return res
}

type timeIdentity struct {
	chain chainIdentity
	index int
}

func (i timeIdentity) dataPoint(data map[chainIdentity][]*DataPoint) *DataPoint {
	return data[i.chain][i.index]
}

func (i timeIdentity) atEnd(data map[chainIdentity][]*DataPoint) bool {
	return (i.index + 1) == len(data[i.chain])
}

func (i timeIdentity) pastEnd(data map[chainIdentity][]*DataPoint) bool {
	return i.index >= len(data[i.chain])
}

func allPastEnd(data map[chainIdentity][]*DataPoint, pos map[chainIdentity]int) bool {
	for chainID := range pos {
		if !(timeIdentity{chainID, pos[chainID]}).pastEnd(data) {
			return false
		}
	}
	return true
}

// findFirstStrideWindow returns the start of the first stride window
// Tried to implement finding an optimal stride window but it was becoming too complicated and not worth it given that it will
// potentially save the first and last stride but it is not guaranteed. Current implementation should give good results
// as long as the the DataPoints are regular enough
func findFirstStrideWindow(data map[chainIdentity][]*DataPoint, stride time.Duration) int64 {
	pos := make(map[chainIdentity]int)
	for k := range data {
		pos[k] = 0
	}

	// Identify the current oldest and newest block
	cur := sortTimeAsc(data, pos)
	return int64(cur[0].dataPoint(data).Timestamp)
}

func copyMap[K comparable, V any](original map[K]V) map[K]V {
	copy := make(map[K]V, len(original))
	for key, value := range original {
		copy[key] = value
	}
	return copy
}

// startPos might have indexes past the end of the data for a chain
func dataInStrideWindowAndNextPos(data map[chainIdentity][]*DataPoint, startPos map[chainIdentity]int, endT int64) (identities map[chainIdentity][]timeIdentity, nextPos map[chainIdentity]int) {
	pos := copyMap(startPos)
	identities = make(map[chainIdentity][]timeIdentity)

	// Identify the current oldest and newest block
	lastLen := int(-1)
	for lastLen < len(identities) {
		lastLen = len(identities)
		sorted := sortTimeAsc(data, pos)
		for _, identity := range sorted {
			if identity.dataPoint(data).Timestamp < uint64(endT) {
				identities[identity.chain] = append(identities[identity.chain], identity)
				pos[identity.chain]++
			}
		}
	}
	return identities, pos
}

// sortTimeAsc expect indexes in pos past the end of the data for a chain
func sortTimeAsc(data map[chainIdentity][]*DataPoint, pos map[chainIdentity]int) []timeIdentity {
	res := make([]timeIdentity, 0, len(data))
	for k := range data {
		identity := timeIdentity{
			chain: k,
			index: pos[k],
		}
		if !identity.pastEnd(data) {
			res = append(res, identity)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].dataPoint(data).Timestamp < res[j].dataPoint(data).Timestamp
	})
	return res
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
