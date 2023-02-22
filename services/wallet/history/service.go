package history

import (
	"context"
	"database/sql"
	"errors"
	"math"
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
	"github.com/status-im/status-go/services/wallet/thirdparty"
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

	exchange *Exchange

	timer                    *time.Timer
	visibleTokenSymbols      []string
	visibleTokenSymbolsMutex sync.Mutex
}

type chainIdentity uint64

func NewService(db *sql.DB, eventFeed *event.Feed, rpcClient *statusrpc.Client, tokenManager *token.Manager, cryptoCompare *thirdparty.CryptoCompare) *Service {
	return &Service{
		balance:        NewBalance(NewBalanceDB(db)),
		db:             db,
		eventFeed:      eventFeed,
		rpcClient:      rpcClient,
		networkManager: rpcClient.NetworkManager,
		tokenManager:   tokenManager,
		exchange:       NewExchange(cryptoCompare),
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

func (s *Service) Start() {
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

type ValuePoint struct {
	Value       float64      `json:"value"`
	Timestamp   uint64       `json:"time"`
	BlockNumber *hexutil.Big `json:"blockNumber"`
}

// GetBalanceHistory returns token count balance
func (s *Service) GetBalanceHistory(ctx context.Context, chainIDs []uint64, address common.Address, tokenSymbol string, currencySymbol string, endTimestamp int64, timeInterval TimeInterval) ([]*ValuePoint, error) {
	// Retrieve cached data for all chains
	allData := make(map[chainIdentity][]*DataPoint)
	for _, chainID := range chainIDs {
		data, err := s.balance.get(ctx, chainID, tokenSymbol, address, endTimestamp, timeInterval)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			allData[chainIdentity(chainID)] = data
		} else {
			return make([]*ValuePoint, 0), nil
		}
	}

	data, err := mergeDataPoints(allData, timeIntervalToStrideDuration[timeInterval])
	if err != nil {
		return nil, err
	} else if len(data) == 0 {
		return make([]*ValuePoint, 0), nil
	}

	// Check if historical exchange rate for data point is present and fetch remaining if not
	lastDayTime := time.Unix(int64(data[len(data)-1].Timestamp), 0).UTC()
	currentTime := time.Now().UTC()
	currentDayStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)
	if lastDayTime.After(currentDayStart) {
		// No chance to have today, use the previous day value for the last data point
		lastDayTime = lastDayTime.AddDate(0, 0, -1)
	}

	_, err = s.exchange.GetExchangeRateForDay(tokenSymbol, currencySymbol, lastDayTime)
	if err != nil {
		err := s.exchange.FetchAndCacheMissingRates(tokenSymbol, currencySymbol)
		if err != nil {
			return nil, err
		}
	}

	decimals, err := s.decimalsForToken(tokenSymbol, chainIDs[0])
	if err != nil {
		return nil, err
	}
	weisInOneMain := big.NewFloat(math.Pow(10, float64(decimals)))

	var res []*ValuePoint
	for _, d := range data {
		dayTime := time.Unix(int64(d.Timestamp), 0).UTC()
		if dayTime.After(currentDayStart) {
			// No chance to have today, use the previous day value for the last data point
			dayTime = lastDayTime
		}
		dayValue, err := s.exchange.GetExchangeRateForDay(tokenSymbol, currencySymbol, dayTime)
		if err != nil {
			log.Warn("Echange rate missing for", dayTime, "- err", err)
			continue
		}

		// The big.Int values are discarded, hence copy the original values
		res = append(res, &ValuePoint{
			Timestamp:   d.Timestamp,
			Value:       tokenToValue((*big.Int)(d.Balance), dayValue, weisInOneMain),
			BlockNumber: d.BlockNumber,
		})
	}
	return res, nil
}

func (s *Service) decimalsForToken(tokenSymbol string, chainID uint64) (int, error) {
	network := s.networkManager.Find(chainID)
	if network == nil {
		return 0, errors.New("network not found")
	}
	token := s.tokenManager.FindToken(network, tokenSymbol)
	if token == nil {
		return 0, errors.New("token not found")
	}
	return int(token.Decimals), nil
}

func tokenToValue(tokenCount *big.Int, mainDenominationValue float32, weisInOneMain *big.Float) float64 {
	weis := new(big.Float).SetInt(tokenCount)
	mainTokens := new(big.Float).Quo(weis, weisInOneMain)
	mainTokenValue := new(big.Float).SetFloat64(float64(mainDenominationValue))
	res, accuracy := new(big.Float).Mul(mainTokens, mainTokenValue).Float64()
	if res == 0 && accuracy == big.Below {
		return math.SmallestNonzeroFloat64
	} else if res == math.Inf(1) && accuracy == big.Above {
		return math.Inf(1)
	}

	return res
}

// mergeDataPoints merges close in time block numbers. Drops the ones that are not in a stride duration
// this should improve merging balance data from different chains which are incompatible due to different timelines
// and block length
func mergeDataPoints(data map[chainIdentity][]*DataPoint, stride time.Duration) ([]*DataPoint, error) {
	// Special cases
	if len(data) == 0 {
		return make([]*DataPoint, 0), nil
	} else if len(data) == 1 {
		for k := range data {
			return data[k], nil
		}
	}

	res := make([]*DataPoint, 0)
	strideStart, pos := findFirstStrideWindow(data, stride)
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
					if exists && (*big.Int)(identity.dataPoint(data).Balance).Cmp((*big.Int)(chainMaxBalance[chainID].Balance)) <= 0 {
						continue
					}
					chainMaxBalance[chainID] = identity.dataPoint(data)
				}
			}
			balance := big.NewInt(0)
			for _, chainBalance := range chainMaxBalance {
				balance.Add(balance, (*big.Int)(chainBalance.Balance))
			}

			// if last stride, the timestamp might be in the future
			if strideEnd > time.Now().UTC().Unix() {
				strideEnd = time.Now().UTC().Unix()
			}

			res = append(res, &DataPoint{
				Timestamp:   uint64(strideEnd),
				Balance:     (*hexutil.Big)(balance),
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

// findFirstStrideWindow returns the start of the first stride window (timestamp and all positions)
//
// Note: tried to implement finding an optimal stride window but it was becoming too complicated and not worth it given that it will potentially save the first and last stride but it is not guaranteed. Current implementation should give good results as long as the the DataPoints are regular enough
func findFirstStrideWindow(data map[chainIdentity][]*DataPoint, stride time.Duration) (firstTimestamp int64, pos map[chainIdentity]int) {
	pos = make(map[chainIdentity]int)
	for k := range data {
		pos[k] = 0
	}

	cur := sortTimeAsc(data, pos)
	return int64(cur[0].dataPoint(data).Timestamp), pos
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
