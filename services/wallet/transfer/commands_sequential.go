package transfer

import (
	"context"
	"math/big"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/contracts"
	nodetypes "github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/rpc/chain/rpclimiter"
	"github.com/status-im/status-go/services/wallet/async"
	"github.com/status-im/status-go/services/wallet/balance"
	"github.com/status-im/status-go/services/wallet/blockchainstate"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/transactions"
)

var findBlocksRetryInterval = 5 * time.Second

const (
	transferHistoryTag    = "transfer_history"
	newTransferHistoryTag = "new_transfer_history"

	transferHistoryLimit           = 10000
	transferHistoryLimitPerAccount = 5000
	transferHistoryLimitPeriod     = 24 * time.Hour
)

type nonceInfo struct {
	nonce       *int64
	blockNumber *big.Int
}

type findNewBlocksCommand struct {
	*findBlocksCommand
	contractMaker                *contracts.ContractMaker
	iteration                    int
	blockChainState              *blockchainstate.BlockChainState
	lastNonces                   map[common.Address]nonceInfo
	nonceCheckIntervalIterations int
	logsCheckIntervalIterations  int
}

func (c *findNewBlocksCommand) Command() async.Command {
	return async.InfiniteCommand{
		Interval: 2 * time.Minute,
		Runable:  c.Run,
	}.Run
}

var requestTimeout = 20 * time.Second

func (c *findNewBlocksCommand) detectTransfers(parent context.Context, accounts []common.Address) (*big.Int, []common.Address, error) {
	bc, err := c.contractMaker.NewBalanceChecker(c.chainClient.NetworkID())
	if err != nil {
		logutils.ZapLogger().Error("findNewBlocksCommand error creating balance checker", zap.Uint64("chain", c.chainClient.NetworkID()), zap.Error(err))
		return nil, nil, err
	}

	tokens, err := c.tokenManager.GetTokens(c.chainClient.NetworkID())
	if err != nil {
		return nil, nil, err
	}
	tokenAddresses := []common.Address{}
	nilAddress := common.Address{}
	for _, token := range tokens {
		if token.Address != nilAddress {
			tokenAddresses = append(tokenAddresses, token.Address)
		}
	}
	logutils.ZapLogger().Debug("findNewBlocksCommand detectTransfers", zap.Int("cnt", len(tokenAddresses)))

	ctx, cancel := context.WithTimeout(parent, requestTimeout)
	defer cancel()
	blockNum, hashes, err := bc.BalancesHash(&bind.CallOpts{Context: ctx}, c.accounts, tokenAddresses)
	if err != nil {
		logutils.ZapLogger().Error("findNewBlocksCommand can't get balances hashes", zap.Error(err))
		return nil, nil, err
	}

	addressesToCheck := []common.Address{}
	for idx, account := range accounts {
		blockRange, _, err := c.blockRangeDAO.getBlockRange(c.chainClient.NetworkID(), account)
		if err != nil {
			logutils.ZapLogger().Error("findNewBlocksCommand can't get block range",
				zap.Stringer("account", account),
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Error(err),
			)
			return nil, nil, err
		}

		checkHash := common.BytesToHash(hashes[idx][:])
		logutils.ZapLogger().Debug("findNewBlocksCommand comparing hashes",
			zap.Stringer("account", account),
			zap.Uint64("network", c.chainClient.NetworkID()),
			zap.String("old hash", blockRange.balanceCheckHash),
			zap.Stringer("new hash", checkHash),
		)
		if checkHash.String() != blockRange.balanceCheckHash {
			addressesToCheck = append(addressesToCheck, account)
		}

		blockRange.balanceCheckHash = checkHash.String()

		err = c.blockRangeDAO.upsertRange(c.chainClient.NetworkID(), account, blockRange)
		if err != nil {
			logutils.ZapLogger().Error("findNewBlocksCommand can't update balance check",
				zap.Stringer("account", account),
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Error(err),
			)
			return nil, nil, err
		}
	}

	return blockNum, addressesToCheck, nil
}

func (c *findNewBlocksCommand) detectNonceChange(parent context.Context, to *big.Int, accounts []common.Address) (map[common.Address]*big.Int, error) {
	addressesWithChange := map[common.Address]*big.Int{}
	for _, account := range accounts {
		var oldNonce *int64

		blockRange, _, err := c.blockRangeDAO.getBlockRange(c.chainClient.NetworkID(), account)
		if err != nil {
			logutils.ZapLogger().Error("findNewBlocksCommand can't get block range",
				zap.Stringer("account", account),
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Error(err),
			)
			return nil, err
		}

		lastNonceInfo, ok := c.lastNonces[account]
		if !ok || lastNonceInfo.blockNumber.Cmp(blockRange.eth.LastKnown) != 0 {
			logutils.ZapLogger().Debug("Fetching old nonce",
				zap.Stringer("at", blockRange.eth.LastKnown),
				zap.Stringer("acc", account),
			)
			if blockRange.eth.LastKnown == nil {
				blockRange.eth.LastKnown = big.NewInt(0)
				oldNonce = new(int64) // At 0 block nonce is 0
			} else {
				oldNonce, err = c.balanceCacher.NonceAt(parent, c.chainClient, account, blockRange.eth.LastKnown)
				if err != nil {
					logutils.ZapLogger().Error("findNewBlocksCommand can't get nonce",
						zap.Stringer("account", account),
						zap.Uint64("chain", c.chainClient.NetworkID()),
						zap.Error(err),
					)
					return nil, err
				}
			}
		} else {
			oldNonce = lastNonceInfo.nonce
		}

		newNonce, err := c.balanceCacher.NonceAt(parent, c.chainClient, account, to)
		if err != nil {
			logutils.ZapLogger().Error("findNewBlocksCommand can't get nonce",
				zap.Stringer("account", account),
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Error(err),
			)
			return nil, err
		}

		logutils.ZapLogger().Debug("Comparing nonces",
			zap.Int64p("oldNonce", oldNonce),
			zap.Int64p("newNonce", newNonce),
			zap.Stringer("to", to),
			zap.Stringer("acc", account),
		)

		if *newNonce != *oldNonce {
			addressesWithChange[account] = blockRange.eth.LastKnown
		}

		if c.lastNonces == nil {
			c.lastNonces = map[common.Address]nonceInfo{}
		}

		c.lastNonces[account] = nonceInfo{
			nonce:       newNonce,
			blockNumber: to,
		}
	}

	return addressesWithChange, nil
}

var nonceCheckIntervalIterations = 30
var logsCheckIntervalIterations = 5

func (c *findNewBlocksCommand) Run(parent context.Context) error {
	mnemonicWasNotShown, err := c.accountsDB.GetMnemonicWasNotShown()
	if err != nil {
		return err
	}

	accountsToCheck := []common.Address{}
	// accounts which might have outgoing transfers initiated outside
	// the application, e.g. watch only or restored from mnemonic phrase
	accountsWithOutsideTransfers := []common.Address{}

	for _, account := range c.accounts {
		acc, err := c.accountsDB.GetAccountByAddress(nodetypes.Address(account))
		if err != nil {
			return err
		}
		if mnemonicWasNotShown {
			if acc.AddressWasNotShown {
				logutils.ZapLogger().Info("skip findNewBlocksCommand, mnemonic has not been shown and the address has not been shared yet", zap.Stringer("address", account))
				continue
			}
		}
		if !mnemonicWasNotShown || acc.Type != accounts.AccountTypeGenerated {
			accountsWithOutsideTransfers = append(accountsWithOutsideTransfers, account)
		}

		accountsToCheck = append(accountsToCheck, account)
	}

	if len(accountsToCheck) == 0 {
		return nil
	}

	headNum, accountsWithDetectedChanges, err := c.detectTransfers(parent, accountsToCheck)
	if err != nil {
		logutils.ZapLogger().Error("findNewBlocksCommand error on transfer detection",
			zap.Uint64("chain", c.chainClient.NetworkID()),
			zap.Error(err),
		)
		return err
	}

	c.blockChainState.SetLastBlockNumber(c.chainClient.NetworkID(), headNum.Uint64())

	if len(accountsWithDetectedChanges) != 0 {
		logutils.ZapLogger().Debug("findNewBlocksCommand detected accounts with changes",
			zap.Stringers("accounts", accountsWithDetectedChanges),
			zap.Stringer("from", c.fromBlockNumber),
		)
		err = c.findAndSaveEthBlocks(parent, c.fromBlockNumber, headNum, accountsToCheck)
		if err != nil {
			return err
		}
	} else if c.iteration%c.nonceCheckIntervalIterations == 0 && len(accountsWithOutsideTransfers) > 0 {
		logutils.ZapLogger().Debug("findNewBlocksCommand nonce check", zap.Stringers("accounts", accountsWithOutsideTransfers))
		accountsWithNonceChanges, err := c.detectNonceChange(parent, headNum, accountsWithOutsideTransfers)
		if err != nil {
			return err
		}

		if len(accountsWithNonceChanges) > 0 {
			logutils.ZapLogger().Debug("findNewBlocksCommand detected nonce diff", zap.Any("accounts", accountsWithNonceChanges))
			for account, from := range accountsWithNonceChanges {
				err = c.findAndSaveEthBlocks(parent, from, headNum, []common.Address{account})
				if err != nil {
					return err
				}
			}
		}

		for _, account := range accountsToCheck {
			if _, ok := accountsWithNonceChanges[account]; ok {
				continue
			}
			err := c.markEthBlockRangeChecked(account, &BlockRange{nil, c.fromBlockNumber, headNum})
			if err != nil {
				return err
			}
		}
	}

	if len(accountsWithDetectedChanges) != 0 || c.iteration%c.logsCheckIntervalIterations == 0 {
		from := c.fromBlockNumber
		if c.logsCheckLastKnownBlock != nil {
			from = c.logsCheckLastKnownBlock
		}
		err = c.findAndSaveTokenBlocks(parent, from, headNum)
		if err != nil {
			return err
		}
		c.logsCheckLastKnownBlock = headNum
	}
	c.fromBlockNumber = headNum
	c.iteration++

	return nil
}

func (c *findNewBlocksCommand) findAndSaveEthBlocks(parent context.Context, fromNum, headNum *big.Int, accounts []common.Address) error {
	// Check ETH transfers for each account independently
	mnemonicWasNotShown, err := c.accountsDB.GetMnemonicWasNotShown()
	if err != nil {
		return err
	}

	for _, account := range accounts {
		if mnemonicWasNotShown {
			acc, err := c.accountsDB.GetAccountByAddress(nodetypes.Address(account))
			if err != nil {
				return err
			}
			if acc.AddressWasNotShown {
				logutils.ZapLogger().Info("skip findNewBlocksCommand, mnemonic has not been shown and the address has not been shared yet", zap.Stringer("address", account))
				continue
			}
		}

		logutils.ZapLogger().Debug("start findNewBlocksCommand",
			zap.Stringer("account", account),
			zap.Uint64("chain", c.chainClient.NetworkID()),
			zap.Bool("noLimit", c.noLimit),
			zap.Stringer("from", fromNum),
			zap.Stringer("to", headNum),
		)

		headers, startBlockNum, err := c.findBlocksWithEthTransfers(parent, account, fromNum, headNum)
		if err != nil {
			return err
		}

		if len(headers) > 0 {
			logutils.ZapLogger().Debug("findNewBlocksCommand saving headers",
				zap.Int("len", len(headers)),
				zap.Stringer("lastBlockNumber", headNum),
				zap.Stringer("balance", c.balanceCacher.Cache().GetBalance(account, c.chainClient.NetworkID(), headNum)),
				zap.Int64p("nonce", c.balanceCacher.Cache().GetNonce(account, c.chainClient.NetworkID(), headNum)),
			)

			err := c.db.SaveBlocks(c.chainClient.NetworkID(), headers)
			if err != nil {
				return err
			}

			c.blocksFound(headers)
		}

		err = c.markEthBlockRangeChecked(account, &BlockRange{startBlockNum, fromNum, headNum})
		if err != nil {
			return err
		}

		logutils.ZapLogger().Debug("end findNewBlocksCommand",
			zap.Stringer("account", account),
			zap.Uint64("chain", c.chainClient.NetworkID()),
			zap.Bool("noLimit", c.noLimit),
			zap.Stringer("from", fromNum),
			zap.Stringer("to", headNum),
		)
	}

	return nil
}

func (c *findNewBlocksCommand) findAndSaveTokenBlocks(parent context.Context, fromNum, headNum *big.Int) error {
	// Check token transfers for all accounts.
	// Each account's last checked block can be different, so we can get duplicated headers,
	// so we need to deduplicate them
	const incomingOnly = false
	erc20Headers, err := c.fastIndexErc20(parent, fromNum, headNum, incomingOnly)
	if err != nil {
		logutils.ZapLogger().Error("findNewBlocksCommand fastIndexErc20",
			zap.Stringers("account", c.accounts),
			zap.Uint64("chain", c.chainClient.NetworkID()),
			zap.Error(err),
		)
		return err
	}

	if len(erc20Headers) > 0 {
		logutils.ZapLogger().Debug("findNewBlocksCommand saving headers",
			zap.Int("len", len(erc20Headers)),
			zap.Stringer("from", fromNum),
			zap.Stringer("to", headNum),
		)

		// get not loaded headers from DB for all accs and blocks
		preLoadedTransactions, err := c.db.GetTransactionsToLoad(c.chainClient.NetworkID(), common.Address{}, nil)
		if err != nil {
			return err
		}

		tokenBlocksFiltered := filterNewPreloadedTransactions(erc20Headers, preLoadedTransactions)

		err = c.db.SaveBlocks(c.chainClient.NetworkID(), tokenBlocksFiltered)
		if err != nil {
			return err
		}

		c.blocksFound(tokenBlocksFiltered)
	}

	return c.markTokenBlockRangeChecked(c.accounts, fromNum, headNum)
}

func (c *findBlocksCommand) markTokenBlockRangeChecked(accounts []common.Address, from, to *big.Int) error {
	logutils.ZapLogger().Debug("markTokenBlockRangeChecked",
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Uint64("from", from.Uint64()),
		zap.Uint64("to", to.Uint64()),
	)

	for _, account := range accounts {
		err := c.blockRangeDAO.updateTokenRange(c.chainClient.NetworkID(), account, &BlockRange{FirstKnown: from, LastKnown: to})
		if err != nil {
			logutils.ZapLogger().Error("findNewBlocksCommand upsertTokenRange", zap.Error(err))
			return err
		}
	}

	return nil
}

func filterNewPreloadedTransactions(erc20Headers []*DBHeader, preLoadedTransfers []*PreloadedTransaction) []*DBHeader {
	var uniqueErc20Headers []*DBHeader
	for _, header := range erc20Headers {
		loaded := false
		for _, transfer := range preLoadedTransfers {
			if header.PreloadedTransactions[0].ID == transfer.ID {
				loaded = true
				break
			}
		}

		if !loaded {
			uniqueErc20Headers = append(uniqueErc20Headers, header)
		}
	}

	return uniqueErc20Headers
}

func (c *findNewBlocksCommand) findBlocksWithEthTransfers(parent context.Context, account common.Address, fromOrig, toOrig *big.Int) (headers []*DBHeader, startBlockNum *big.Int, err error) {
	logutils.ZapLogger().Debug("start findNewBlocksCommand::findBlocksWithEthTransfers",
		zap.Stringer("account", account),
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Bool("noLimit", c.noLimit),
		zap.Stringer("from", c.fromBlockNumber),
		zap.Stringer("to", c.toBlockNumber),
	)

	rangeSize := big.NewInt(int64(c.defaultNodeBlockChunkSize))

	from, to := new(big.Int).Set(fromOrig), new(big.Int).Set(toOrig)

	// Limit the range size to DefaultNodeBlockChunkSize
	if new(big.Int).Sub(to, from).Cmp(rangeSize) > 0 {
		from.Sub(to, rangeSize)
	}

	for {
		if from.Cmp(to) == 0 {
			logutils.ZapLogger().Debug("findNewBlocksCommand empty range",
				zap.Stringer("from", from),
				zap.Stringer("to", to),
			)
			break
		}

		fromBlock := &Block{Number: from}

		var newFromBlock *Block
		var ethHeaders []*DBHeader
		newFromBlock, ethHeaders, startBlockNum, err = c.fastIndex(parent, account, c.balanceCacher, fromBlock, to)
		if err != nil {
			logutils.ZapLogger().Error("findNewBlocksCommand checkRange fastIndex",
				zap.Stringer("account", account),
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Error(err),
			)
			return nil, nil, err
		}
		logutils.ZapLogger().Debug("findNewBlocksCommand checkRange",
			zap.Uint64("chainID", c.chainClient.NetworkID()),
			zap.Stringer("account", account),
			zap.Stringer("startBlock", startBlockNum),
			zap.Stringer("newFromBlock", newFromBlock.Number),
			zap.Stringer("toBlockNumber", to),
			zap.Bool("noLimit", c.noLimit),
		)

		headers = append(headers, ethHeaders...)

		if startBlockNum != nil && startBlockNum.Cmp(from) >= 0 {
			logutils.ZapLogger().Debug("Checked all ranges, stop execution",
				zap.Stringer("startBlock", startBlockNum),
				zap.Stringer("from", from),
				zap.Stringer("to", to),
			)
			break
		}

		nextFrom, nextTo := nextRange(c.defaultNodeBlockChunkSize, newFromBlock.Number, fromOrig)

		if nextFrom.Cmp(from) == 0 && nextTo.Cmp(to) == 0 {
			logutils.ZapLogger().Debug("findNewBlocksCommand empty next range",
				zap.Stringer("from", from),
				zap.Stringer("to", to),
			)
			break
		}

		from = nextFrom
		to = nextTo
	}

	logutils.ZapLogger().Debug("end findNewBlocksCommand::findBlocksWithEthTransfers",
		zap.Stringer("account", account),
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Bool("noLimit", c.noLimit),
	)

	return headers, startBlockNum, nil
}

// TODO NewFindBlocksCommand
type findBlocksCommand struct {
	accounts                  []common.Address
	db                        *Database
	accountsDB                *accounts.Database
	blockRangeDAO             BlockRangeDAOer
	chainClient               chain.ClientInterface
	balanceCacher             balance.Cacher
	feed                      *event.Feed
	noLimit                   bool
	tokenManager              *token.Manager
	fromBlockNumber           *big.Int
	logsCheckLastKnownBlock   *big.Int
	toBlockNumber             *big.Int
	blocksLoadedCh            chan<- []*DBHeader
	defaultNodeBlockChunkSize int

	// Not to be set by the caller
	resFromBlock           *Block
	startBlockNumber       *big.Int
	reachedETHHistoryStart bool
}

func (c *findBlocksCommand) Runner(interval ...time.Duration) async.Runner {
	intvl := findBlocksRetryInterval
	if len(interval) > 0 {
		intvl = interval[0]
	}
	return async.FiniteCommandWithErrorCounter{
		FiniteCommand: async.FiniteCommand{
			Interval: intvl,
			Runable:  c.Run,
		},
		ErrorCounter: async.NewErrorCounter(3, "findBlocksCommand"),
	}
}

func (c *findBlocksCommand) Command(interval ...time.Duration) async.Command {
	return c.Runner(interval...).Run
}

type ERC20BlockRange struct {
	from *big.Int
	to   *big.Int
}

func (c *findBlocksCommand) ERC20ScanByBalance(parent context.Context, account common.Address, fromBlock, toBlock *big.Int, token common.Address) ([]ERC20BlockRange, error) {
	var err error
	batchSize := getErc20BatchSize(c.chainClient.NetworkID())
	ranges := [][]*big.Int{{fromBlock, toBlock}}
	foundRanges := []ERC20BlockRange{}
	cache := map[int64]*big.Int{}
	for {
		nextRanges := [][]*big.Int{}
		for _, blockRange := range ranges {
			from, to := blockRange[0], blockRange[1]
			fromBalance, ok := cache[from.Int64()]
			if !ok {
				fromBalance, err = c.tokenManager.GetTokenBalanceAt(parent, c.chainClient, account, token, from)
				if err != nil {
					return nil, err
				}

				if fromBalance == nil {
					fromBalance = big.NewInt(0)
				}
				cache[from.Int64()] = fromBalance
			}

			toBalance, ok := cache[to.Int64()]
			if !ok {
				toBalance, err = c.tokenManager.GetTokenBalanceAt(parent, c.chainClient, account, token, to)
				if err != nil {
					return nil, err
				}
				if toBalance == nil {
					toBalance = big.NewInt(0)
				}
				cache[to.Int64()] = toBalance
			}

			if fromBalance.Cmp(toBalance) != 0 {
				diff := new(big.Int).Sub(to, from)
				if diff.Cmp(batchSize) <= 0 {
					foundRanges = append(foundRanges, ERC20BlockRange{from, to})
					continue
				}

				halfOfDiff := new(big.Int).Div(diff, big.NewInt(2))
				mid := new(big.Int).Add(from, halfOfDiff)

				nextRanges = append(nextRanges, []*big.Int{from, mid})
				nextRanges = append(nextRanges, []*big.Int{mid, to})
			}
		}

		if len(nextRanges) == 0 {
			break
		}

		ranges = nextRanges
	}

	return foundRanges, nil
}

func (c *findBlocksCommand) checkERC20Tail(parent context.Context, account common.Address) ([]*DBHeader, error) {
	logutils.ZapLogger().Debug(
		"checkERC20Tail",
		zap.Stringer("account", account),
		zap.Stringer("to block", c.startBlockNumber),
		zap.Stringer("from", c.resFromBlock.Number),
	)
	tokens, err := c.tokenManager.GetTokens(c.chainClient.NetworkID())
	if err != nil {
		return nil, err
	}
	addresses := make([]common.Address, len(tokens))
	for i, token := range tokens {
		addresses[i] = token.Address
	}

	from := new(big.Int).Sub(c.resFromBlock.Number, big.NewInt(1))

	clients := make(map[uint64]chain.ClientInterface, 1)
	clients[c.chainClient.NetworkID()] = c.chainClient
	atBlocks := make(map[uint64]*big.Int, 1)
	atBlocks[c.chainClient.NetworkID()] = from
	balances, err := c.tokenManager.GetBalancesAtByChain(parent, clients, []common.Address{account}, addresses, atBlocks)
	if err != nil {
		return nil, err
	}

	foundRanges := []ERC20BlockRange{}
	for token, balance := range balances[c.chainClient.NetworkID()][account] {
		bigintBalance := big.NewInt(balance.ToInt().Int64())
		if bigintBalance.Cmp(big.NewInt(0)) <= 0 {
			continue
		}
		result, err := c.ERC20ScanByBalance(parent, account, big.NewInt(0), from, token)
		if err != nil {
			return nil, err
		}

		foundRanges = append(foundRanges, result...)
	}

	uniqRanges := []ERC20BlockRange{}
	rangesMap := map[string]bool{}
	for _, rangeItem := range foundRanges {
		key := rangeItem.from.String() + "-" + rangeItem.to.String()
		if _, ok := rangesMap[key]; !ok {
			rangesMap[key] = true
			uniqRanges = append(uniqRanges, rangeItem)
		}
	}

	foundHeaders := []*DBHeader{}
	for _, rangeItem := range uniqRanges {
		headers, err := c.fastIndexErc20(parent, rangeItem.from, rangeItem.to, true)
		if err != nil {
			return nil, err
		}
		foundHeaders = append(foundHeaders, headers...)
	}

	return foundHeaders, nil
}

func (c *findBlocksCommand) Run(parent context.Context) (err error) {
	logutils.ZapLogger().Debug("start findBlocksCommand",
		zap.Any("accounts", c.accounts),
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Bool("noLimit", c.noLimit),
		zap.Stringer("from", c.fromBlockNumber),
		zap.Stringer("to", c.toBlockNumber),
	)

	account := c.accounts[0] // For now this command supports only 1 account
	mnemonicWasNotShown, err := c.accountsDB.GetMnemonicWasNotShown()
	if err != nil {
		return err
	}

	if mnemonicWasNotShown {
		account, err := c.accountsDB.GetAccountByAddress(nodetypes.BytesToAddress(account.Bytes()))
		if err != nil {
			return err
		}
		if account.AddressWasNotShown {
			logutils.ZapLogger().Info("skip findBlocksCommand, mnemonic has not been shown and the address has not been shared yet", zap.Stringer("address", account.Address))
			return nil
		}
	}

	rangeSize := big.NewInt(int64(c.defaultNodeBlockChunkSize))
	from, to := new(big.Int).Set(c.fromBlockNumber), new(big.Int).Set(c.toBlockNumber)

	// Limit the range size to DefaultNodeBlockChunkSize
	if new(big.Int).Sub(to, from).Cmp(rangeSize) > 0 {
		from.Sub(to, rangeSize)
	}

	for {
		if from.Cmp(to) == 0 {
			logutils.ZapLogger().Debug("findBlocksCommand empty range",
				zap.Stringer("from", from),
				zap.Stringer("to", to))
			break
		}

		var headers []*DBHeader
		if c.reachedETHHistoryStart {
			if c.fromBlockNumber.Cmp(zero) == 0 && c.startBlockNumber != nil && c.startBlockNumber.Cmp(zero) == 1 {
				headers, err = c.checkERC20Tail(parent, account)
				if err != nil {
					logutils.ZapLogger().Error("findBlocksCommand checkERC20Tail",
						zap.Stringer("account", account),
						zap.Uint64("chain", c.chainClient.NetworkID()),
						zap.Error(err),
					)
					break
				}
			}
		} else {
			headers, err = c.checkRange(parent, from, to)
			if err != nil {
				break
			}
		}

		if len(headers) > 0 {
			logutils.ZapLogger().Debug("findBlocksCommand saving headers",
				zap.Int("len", len(headers)),
				zap.Stringer("lastBlockNumber", to),
				zap.Stringer("balance", c.balanceCacher.Cache().GetBalance(account, c.chainClient.NetworkID(), to)),
				zap.Int64p("nonce", c.balanceCacher.Cache().GetNonce(account, c.chainClient.NetworkID(), to)),
			)

			err = c.db.SaveBlocks(c.chainClient.NetworkID(), headers)
			if err != nil {
				break
			}

			c.blocksFound(headers)
		}

		if c.reachedETHHistoryStart {
			err = c.markTokenBlockRangeChecked([]common.Address{account}, big.NewInt(0), to)
			if err != nil {
				break
			}
			logutils.ZapLogger().Debug("findBlocksCommand reached first ETH transfer and checked erc20 tail",
				zap.Uint64("chain", c.chainClient.NetworkID()),
				zap.Stringer("account", account),
			)
			break
		}

		err = c.markEthBlockRangeChecked(account, &BlockRange{c.startBlockNumber, c.resFromBlock.Number, to})
		if err != nil {
			break
		}

		err = c.markTokenBlockRangeChecked([]common.Address{account}, c.resFromBlock.Number, to)
		if err != nil {
			break
		}

		// if we have found first ETH block and we have not reached the start of ETH history yet
		if c.startBlockNumber != nil && c.fromBlockNumber.Cmp(from) == -1 {
			logutils.ZapLogger().Debug("ERC20 tail should be checked",
				zap.Stringer("initial from", c.fromBlockNumber),
				zap.Stringer("actual from", from),
				zap.Stringer("first ETH block", c.startBlockNumber),
			)
			c.reachedETHHistoryStart = true
			continue
		}

		if c.startBlockNumber != nil && c.startBlockNumber.Cmp(from) >= 0 {
			logutils.ZapLogger().Debug("Checked all ranges, stop execution",
				zap.Stringer("startBlock", c.startBlockNumber),
				zap.Stringer("from", from),
				zap.Stringer("to", to),
			)
			break
		}

		nextFrom, nextTo := nextRange(c.defaultNodeBlockChunkSize, c.resFromBlock.Number, c.fromBlockNumber)

		if nextFrom.Cmp(from) == 0 && nextTo.Cmp(to) == 0 {
			logutils.ZapLogger().Debug("findBlocksCommand empty next range",
				zap.Stringer("from", from),
				zap.Stringer("to", to),
			)
			break
		}

		from = nextFrom
		to = nextTo
	}

	logutils.ZapLogger().Debug("end findBlocksCommand",
		zap.Stringer("account", account),
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Bool("noLimit", c.noLimit),
		zap.Error(err),
	)

	return err
}

func (c *findBlocksCommand) blocksFound(headers []*DBHeader) {
	c.blocksLoadedCh <- headers
}

func (c *findBlocksCommand) markEthBlockRangeChecked(account common.Address, blockRange *BlockRange) error {
	logutils.ZapLogger().Debug("upsert block range",
		zap.Stringer("Start", blockRange.Start),
		zap.Stringer("FirstKnown", blockRange.FirstKnown),
		zap.Stringer("LastKnown", blockRange.LastKnown),
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Stringer("account", account),
	)

	err := c.blockRangeDAO.upsertEthRange(c.chainClient.NetworkID(), account, blockRange)
	if err != nil {
		logutils.ZapLogger().Error("findBlocksCommand upsertRange", zap.Error(err))
		return err
	}

	return nil
}

func (c *findBlocksCommand) checkRange(parent context.Context, from *big.Int, to *big.Int) (
	foundHeaders []*DBHeader, err error) {

	account := c.accounts[0]
	fromBlock := &Block{Number: from}

	newFromBlock, ethHeaders, startBlock, err := c.fastIndex(parent, account, c.balanceCacher, fromBlock, to)
	if err != nil {
		logutils.ZapLogger().Error("findBlocksCommand checkRange fastIndex",
			zap.Stringer("account", account),
			zap.Uint64("chain", c.chainClient.NetworkID()),
			zap.Error(err),
		)
		return nil, err
	}
	logutils.ZapLogger().Debug("findBlocksCommand checkRange",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringer("account", account),
		zap.Stringer("startBlock", startBlock),
		zap.Stringer("newFromBlock", newFromBlock.Number),
		zap.Stringer("toBlockNumber", to),
		zap.Bool("noLimit", c.noLimit),
	)

	// There could be incoming ERC20 transfers which don't change the balance
	// and nonce of ETH account, so we keep looking for them
	erc20Headers, err := c.fastIndexErc20(parent, newFromBlock.Number, to, false)
	if err != nil {
		logutils.ZapLogger().Error("findBlocksCommand checkRange fastIndexErc20",
			zap.Stringer("account", account),
			zap.Uint64("chain", c.chainClient.NetworkID()),
			zap.Error(err),
		)
		return nil, err
	}

	allHeaders := append(ethHeaders, erc20Headers...)

	if len(allHeaders) > 0 {
		foundHeaders = uniqueHeaderPerBlockHash(allHeaders)
	}

	c.resFromBlock = newFromBlock
	c.startBlockNumber = startBlock

	logutils.ZapLogger().Debug("end findBlocksCommand checkRange",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringer("account", account),
		zap.Stringer("c.startBlock", c.startBlockNumber),
		zap.Stringer("newFromBlock", newFromBlock.Number),
		zap.Stringer("toBlockNumber", to),
		zap.Stringer("c.resFromBlock", c.resFromBlock.Number),
	)

	return
}

func loadBlockRangeInfo(chainID uint64, account common.Address, blockDAO BlockRangeDAOer) (
	*ethTokensBlockRanges, error) {

	blockRange, _, err := blockDAO.getBlockRange(chainID, account)
	if err != nil {
		logutils.ZapLogger().Error(
			"failed to load block ranges from database",
			zap.Uint64("chain", chainID),
			zap.Stringer("account", account),
			zap.Error(err),
		)
		return nil, err
	}

	return blockRange, nil
}

// Returns if all blocks are loaded, which means that start block (beginning of account history)
// has been found and all block headers saved to the DB
func areAllHistoryBlocksLoaded(blockInfo *BlockRange) bool {
	if blockInfo != nil && blockInfo.FirstKnown != nil &&
		((blockInfo.Start != nil && blockInfo.Start.Cmp(blockInfo.FirstKnown) >= 0) ||
			blockInfo.FirstKnown.Cmp(zero) == 0) {
		return true
	}

	return false
}

func areAllHistoryBlocksLoadedForAddress(blockRangeDAO BlockRangeDAOer, chainID uint64,
	address common.Address) (bool, error) {

	blockRange, _, err := blockRangeDAO.getBlockRange(chainID, address)
	if err != nil {
		logutils.ZapLogger().Error("findBlocksCommand getBlockRange", zap.Error(err))
		return false, err
	}

	return areAllHistoryBlocksLoaded(blockRange.eth) && areAllHistoryBlocksLoaded(blockRange.tokens), nil
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findBlocksCommand) fastIndex(ctx context.Context, account common.Address, bCacher balance.Cacher,
	fromBlock *Block, toBlockNumber *big.Int) (resultingFrom *Block, headers []*DBHeader,
	startBlock *big.Int, err error) {

	logutils.ZapLogger().Debug("fast index started",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringer("account", account),
		zap.Stringer("from", fromBlock.Number),
		zap.Stringer("to", toBlockNumber),
	)

	start := time.Now()
	group := async.NewGroup(ctx)

	command := &ethHistoricalCommand{
		chainClient:   c.chainClient,
		balanceCacher: bCacher,
		address:       account,
		feed:          c.feed,
		from:          fromBlock,
		to:            toBlockNumber,
		noLimit:       c.noLimit,
		threadLimit:   SequentialThreadLimit,
	}
	group.Add(command.Command())

	select {
	case <-ctx.Done():
		err = ctx.Err()
		logutils.ZapLogger().Debug("fast indexer ctx Done", zap.Error(err))
		return
	case <-group.WaitAsync():
		if command.error != nil {
			err = command.error
			return
		}
		resultingFrom = &Block{Number: command.resultingFrom}
		headers = command.foundHeaders
		startBlock = command.startBlock
		logutils.ZapLogger().Debug("fast indexer finished",
			zap.Uint64("chainID", c.chainClient.NetworkID()),
			zap.Stringer("account", account),
			zap.Duration("in", time.Since(start)),
			zap.Stringer("startBlock", command.startBlock),
			zap.Stringer("resultingFrom", resultingFrom.Number),
			zap.Int("headers", len(headers)),
		)
		return
	}
}

// run fast indexing for every accont up to canonical chain head minus safety depth.
// every account will run it from last synced header.
func (c *findBlocksCommand) fastIndexErc20(ctx context.Context, fromBlockNumber *big.Int,
	toBlockNumber *big.Int, incomingOnly bool) ([]*DBHeader, error) {

	start := time.Now()
	group := async.NewGroup(ctx)

	erc20 := &erc20HistoricalCommand{
		erc20:        NewERC20TransfersDownloader(c.chainClient, c.accounts, types.LatestSignerForChainID(c.chainClient.ToBigInt()), incomingOnly),
		chainClient:  c.chainClient,
		feed:         c.feed,
		from:         fromBlockNumber,
		to:           toBlockNumber,
		foundHeaders: []*DBHeader{},
	}
	group.Add(erc20.Command())

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-group.WaitAsync():
		headers := erc20.foundHeaders
		logutils.ZapLogger().Debug("fast indexer Erc20 finished",
			zap.Uint64("chainID", c.chainClient.NetworkID()),
			zap.Duration("in", time.Since(start)),
			zap.Int("headers", len(headers)),
		)
		return headers, nil
	}
}

// Start transfers loop to load transfers for new blocks
func (c *loadBlocksAndTransfersCommand) startTransfersLoop(ctx context.Context) {
	c.incLoops()
	go func() {
		defer gocommon.LogOnPanic()
		defer func() {
			c.decLoops()
		}()

		logutils.ZapLogger().Debug("loadTransfersLoop start", zap.Uint64("chain", c.chainClient.NetworkID()))

		for {
			select {
			case <-ctx.Done():
				logutils.ZapLogger().Debug("startTransfersLoop done",
					zap.Uint64("chain", c.chainClient.NetworkID()),
					zap.Error(ctx.Err()),
				)
				return
			case dbHeaders := <-c.blocksLoadedCh:
				logutils.ZapLogger().Debug("loadTransfersOnDemand transfers received",
					zap.Uint64("chain", c.chainClient.NetworkID()),
					zap.Int("headers", len(dbHeaders)),
				)

				blocksByAddress := map[common.Address][]*big.Int{}
				// iterate over headers and group them by address
				for _, dbHeader := range dbHeaders {
					blocksByAddress[dbHeader.Address] = append(blocksByAddress[dbHeader.Address], dbHeader.Number)
				}

				go func() {
					defer gocommon.LogOnPanic()
					_ = loadTransfers(ctx, c.blockDAO, c.db, c.chainClient, noBlockLimit,
						blocksByAddress, c.pendingTxManager, c.tokenManager, c.feed)
				}()
			}
		}
	}()
}

func newLoadBlocksAndTransfersCommand(accounts []common.Address, db *Database, accountsDB *accounts.Database,
	blockDAO *BlockDAO, blockRangesSeqDAO BlockRangeDAOer, chainClient chain.ClientInterface, feed *event.Feed,
	pendingTxManager *transactions.PendingTxTracker,
	tokenManager *token.Manager, balanceCacher balance.Cacher, omitHistory bool,
	blockChainState *blockchainstate.BlockChainState) *loadBlocksAndTransfersCommand {

	return &loadBlocksAndTransfersCommand{
		accounts:         accounts,
		db:               db,
		blockRangeDAO:    blockRangesSeqDAO,
		accountsDB:       accountsDB,
		blockDAO:         blockDAO,
		chainClient:      chainClient,
		feed:             feed,
		balanceCacher:    balanceCacher,
		pendingTxManager: pendingTxManager,
		tokenManager:     tokenManager,
		blocksLoadedCh:   make(chan []*DBHeader, 100),
		omitHistory:      omitHistory,
		contractMaker:    tokenManager.ContractMaker,
		blockChainState:  blockChainState,
	}
}

type loadBlocksAndTransfersCommand struct {
	accounts      []common.Address
	db            *Database
	accountsDB    *accounts.Database
	blockRangeDAO BlockRangeDAOer
	blockDAO      *BlockDAO
	chainClient   chain.ClientInterface
	feed          *event.Feed
	balanceCacher balance.Cacher
	// nonArchivalRPCNode bool // TODO Make use of it
	pendingTxManager *transactions.PendingTxTracker
	tokenManager     *token.Manager
	blocksLoadedCh   chan []*DBHeader
	omitHistory      bool
	contractMaker    *contracts.ContractMaker
	blockChainState  *blockchainstate.BlockChainState

	// Not to be set by the caller
	transfersLoaded map[common.Address]bool // For event RecentHistoryReady to be sent only once per account during app lifetime
	loops           atomic.Int32
}

func (c *loadBlocksAndTransfersCommand) incLoops() {
	c.loops.Add(1)
}

func (c *loadBlocksAndTransfersCommand) decLoops() {
	c.loops.Add(-1)
}

func (c *loadBlocksAndTransfersCommand) isStarted() bool {
	return c.loops.Load() > 0
}

func (c *loadBlocksAndTransfersCommand) Run(parent context.Context) (err error) {
	logutils.ZapLogger().Debug("start load all transfers command",
		zap.Uint64("chain", c.chainClient.NetworkID()),
		zap.Any("accounts", c.accounts),
	)

	// Finite processes (to be restarted on error, but stopped on success or context cancel):
	// fetching transfers for loaded blocks
	// fetching history blocks

	// Infinite processes (to be restarted on error), but stopped on context cancel:
	// fetching new blocks
	// fetching transfers for new blocks

	ctx := parent
	finiteGroup := async.NewAtomicGroup(ctx)
	finiteGroup.SetName("finiteGroup")
	defer func() {
		finiteGroup.Stop()
		finiteGroup.Wait()
	}()

	blockRanges, err := c.blockRangeDAO.getBlockRanges(c.chainClient.NetworkID(), c.accounts)
	if err != nil {
		return err
	}

	firstScan := false
	var headNum *big.Int
	for _, address := range c.accounts {
		blockRange, ok := blockRanges[address]
		if !ok || blockRange.tokens.LastKnown == nil {
			firstScan = true
			break
		}

		if headNum == nil || blockRange.tokens.LastKnown.Cmp(headNum) < 0 {
			headNum = blockRange.tokens.LastKnown
		}
	}

	fromNum := big.NewInt(0)
	if firstScan {
		headNum, err = getHeadBlockNumber(ctx, c.chainClient)
		if err != nil {
			return err
		}
	}

	// It will start loadTransfersCommand which will run until all transfers from DB are loaded or any one failed to load
	err = c.startFetchingTransfersForLoadedBlocks(finiteGroup)
	if err != nil {
		logutils.ZapLogger().Error("loadBlocksAndTransfersCommand fetchTransfersForLoadedBlocks", zap.Error(err))
		return err
	}

	if !c.isStarted() {
		c.startTransfersLoop(ctx)
		c.startFetchingNewBlocks(ctx, c.accounts, headNum, c.blocksLoadedCh)
	}

	// It will start findBlocksCommands which will run until success when all blocks are loaded
	err = c.fetchHistoryBlocks(finiteGroup, c.accounts, fromNum, headNum, c.blocksLoadedCh)
	if err != nil {
		logutils.ZapLogger().Error("loadBlocksAndTransfersCommand fetchHistoryBlocks", zap.Error(err))
		return err
	}

	select {
	case <-ctx.Done():
		logutils.ZapLogger().Debug("loadBlocksAndTransfers command cancelled",
			zap.Uint64("chain", c.chainClient.NetworkID()),
			zap.Stringers("accounts", c.accounts),
			zap.Error(ctx.Err()),
		)
	case <-finiteGroup.WaitAsync():
		err = finiteGroup.Error() // if there was an error, rerun the command
		logutils.ZapLogger().Debug(
			"end loadBlocksAndTransfers command",
			zap.Uint64("chain", c.chainClient.NetworkID()),
			zap.Stringers("accounts", c.accounts),
			zap.String("group", finiteGroup.Name()),
			zap.Error(err),
		)
	}

	return err
}

func (c *loadBlocksAndTransfersCommand) Runner(interval ...time.Duration) async.Runner {
	// 30s - default interval for Infura's delay returned in error. That should increase chances
	// for request to succeed with the next attempt for now until we have a proper retry mechanism
	intvl := 30 * time.Second
	if len(interval) > 0 {
		intvl = interval[0]
	}

	return async.FiniteCommand{
		Interval: intvl,
		Runable:  c.Run,
	}
}

func (c *loadBlocksAndTransfersCommand) Command(interval ...time.Duration) async.Command {
	return c.Runner(interval...).Run
}

func (c *loadBlocksAndTransfersCommand) fetchHistoryBlocks(group *async.AtomicGroup, accounts []common.Address, fromNum, toNum *big.Int, blocksLoadedCh chan []*DBHeader) (err error) {
	for _, account := range accounts {
		err = c.fetchHistoryBlocksForAccount(group, account, fromNum, toNum, c.blocksLoadedCh)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *loadBlocksAndTransfersCommand) fetchHistoryBlocksForAccount(group *async.AtomicGroup, account common.Address, fromNum, toNum *big.Int, blocksLoadedCh chan []*DBHeader) error {
	logutils.ZapLogger().Debug("fetchHistoryBlocks start",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringer("account", account),
		zap.Bool("omit", c.omitHistory),
	)

	if c.omitHistory {
		blockRange := &ethTokensBlockRanges{eth: &BlockRange{nil, big.NewInt(0), toNum}, tokens: &BlockRange{nil, big.NewInt(0), toNum}}
		err := c.blockRangeDAO.upsertRange(c.chainClient.NetworkID(), account, blockRange)
		logutils.ZapLogger().Error("fetchHistoryBlocks upsertRange", zap.Error(err))
		return err
	}

	blockRange, err := loadBlockRangeInfo(c.chainClient.NetworkID(), account, c.blockRangeDAO)
	if err != nil {
		logutils.ZapLogger().Error("fetchHistoryBlocks loadBlockRangeInfo", zap.Error(err))
		return err
	}

	ranges := [][]*big.Int{}
	// There are 2 history intervals:
	// 1) from 0 to FirstKnown
	// 2) from LastKnown to `toNum`` (head)
	// If we blockRange is nil, we need to load all blocks from `fromNum` to `toNum`
	// As current implementation checks ETH first then tokens, tokens ranges maybe behind ETH ranges in
	// cases when block searching was interrupted, so we use tokens ranges
	if blockRange.tokens.LastKnown != nil || blockRange.tokens.FirstKnown != nil {
		if blockRange.tokens.LastKnown != nil && toNum.Cmp(blockRange.tokens.LastKnown) > 0 {
			ranges = append(ranges, []*big.Int{blockRange.tokens.LastKnown, toNum})
		}

		if blockRange.tokens.FirstKnown != nil {
			if fromNum.Cmp(blockRange.tokens.FirstKnown) < 0 {
				ranges = append(ranges, []*big.Int{fromNum, blockRange.tokens.FirstKnown})
			} else {
				if !c.transfersLoaded[account] {
					transfersLoaded, err := c.areAllTransfersLoaded(account)
					if err != nil {
						return err
					}

					if transfersLoaded {
						if c.transfersLoaded == nil {
							c.transfersLoaded = make(map[common.Address]bool)
						}
						c.transfersLoaded[account] = true
						c.notifyHistoryReady(account)
					}
				}
			}
		}
	} else {
		ranges = append(ranges, []*big.Int{fromNum, toNum})
	}

	if len(ranges) > 0 {
		storage := rpclimiter.NewLimitsDBStorage(c.db.client)
		limiter := rpclimiter.NewRequestLimiter(storage)
		chainClient, _ := createChainClientWithLimiter(c.chainClient, account, limiter)
		if chainClient == nil {
			chainClient = c.chainClient
		}

		for _, rangeItem := range ranges {
			logutils.ZapLogger().Debug("range item",
				zap.Stringers("r", rangeItem),
				zap.Uint64("n", c.chainClient.NetworkID()),
				zap.Stringer("a", account),
			)

			fbc := &findBlocksCommand{
				accounts:                  []common.Address{account},
				db:                        c.db,
				accountsDB:                c.accountsDB,
				blockRangeDAO:             c.blockRangeDAO,
				chainClient:               chainClient,
				balanceCacher:             c.balanceCacher,
				feed:                      c.feed,
				noLimit:                   false,
				fromBlockNumber:           rangeItem[0],
				toBlockNumber:             rangeItem[1],
				tokenManager:              c.tokenManager,
				blocksLoadedCh:            blocksLoadedCh,
				defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
			}
			group.Add(fbc.Command())
		}
	}

	return nil
}

func (c *loadBlocksAndTransfersCommand) startFetchingNewBlocks(ctx context.Context, addresses []common.Address, fromNum *big.Int, blocksLoadedCh chan<- []*DBHeader) {
	logutils.ZapLogger().Debug("startFetchingNewBlocks start",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringers("accounts", addresses),
	)

	c.incLoops()
	go func() {
		defer gocommon.LogOnPanic()
		defer func() {
			c.decLoops()
		}()

		newBlocksCmd := &findNewBlocksCommand{
			findBlocksCommand: &findBlocksCommand{
				accounts:                  addresses,
				db:                        c.db,
				accountsDB:                c.accountsDB,
				blockRangeDAO:             c.blockRangeDAO,
				chainClient:               c.chainClient,
				balanceCacher:             c.balanceCacher,
				feed:                      c.feed,
				noLimit:                   false,
				fromBlockNumber:           fromNum,
				tokenManager:              c.tokenManager,
				blocksLoadedCh:            blocksLoadedCh,
				defaultNodeBlockChunkSize: DefaultNodeBlockChunkSize,
			},
			contractMaker:                c.contractMaker,
			blockChainState:              c.blockChainState,
			nonceCheckIntervalIterations: nonceCheckIntervalIterations,
			logsCheckIntervalIterations:  logsCheckIntervalIterations,
		}
		group := async.NewGroup(ctx)
		group.Add(newBlocksCmd.Command())

		// No need to wait for the group since it is infinite
		<-ctx.Done()

		logutils.ZapLogger().Debug("startFetchingNewBlocks end",
			zap.Uint64("chainID", c.chainClient.NetworkID()),
			zap.Stringers("accounts", addresses),
			zap.Error(ctx.Err()),
		)
	}()
}

func (c *loadBlocksAndTransfersCommand) getBlocksToLoad() (map[common.Address][]*big.Int, error) {
	blocksMap := make(map[common.Address][]*big.Int)
	for _, account := range c.accounts {
		blocks, err := c.blockDAO.GetBlocksToLoadByAddress(c.chainClient.NetworkID(), account, numberOfBlocksCheckedPerIteration)
		if err != nil {
			logutils.ZapLogger().Error("loadBlocksAndTransfersCommand GetBlocksToLoadByAddress", zap.Error(err))
			return nil, err
		}

		if len(blocks) == 0 {
			logutils.ZapLogger().Debug("fetchTransfers no blocks to load",
				zap.Uint64("chainID", c.chainClient.NetworkID()),
				zap.Stringer("account", account),
			)
			continue
		}

		blocksMap[account] = blocks
	}

	if len(blocksMap) == 0 {
		logutils.ZapLogger().Debug("fetchTransfers no blocks to load", zap.Uint64("chainID", c.chainClient.NetworkID()))
	}

	return blocksMap, nil
}

func (c *loadBlocksAndTransfersCommand) startFetchingTransfersForLoadedBlocks(group *async.AtomicGroup) error {
	logutils.ZapLogger().Debug("fetchTransfers start",
		zap.Uint64("chainID", c.chainClient.NetworkID()),
		zap.Stringers("accounts", c.accounts),
	)

	blocksMap, err := c.getBlocksToLoad()
	if err != nil {
		return err
	}

	go func() {
		defer gocommon.LogOnPanic()
		txCommand := &loadTransfersCommand{
			accounts:         c.accounts,
			db:               c.db,
			blockDAO:         c.blockDAO,
			chainClient:      c.chainClient,
			pendingTxManager: c.pendingTxManager,
			tokenManager:     c.tokenManager,
			blocksByAddress:  blocksMap,
			feed:             c.feed,
		}

		group.Add(txCommand.Command())
		logutils.ZapLogger().Debug("fetchTransfers end",
			zap.Uint64("chainID", c.chainClient.NetworkID()),
			zap.Stringers("accounts", c.accounts),
		)
	}()

	return nil
}

func (c *loadBlocksAndTransfersCommand) notifyHistoryReady(account common.Address) {
	if c.feed != nil {
		c.feed.Send(walletevent.Event{
			Type:     EventRecentHistoryReady,
			Accounts: []common.Address{account},
			ChainID:  c.chainClient.NetworkID(),
		})
	}
}

func (c *loadBlocksAndTransfersCommand) areAllTransfersLoaded(account common.Address) (bool, error) {
	allBlocksLoaded, err := areAllHistoryBlocksLoadedForAddress(c.blockRangeDAO, c.chainClient.NetworkID(), account)
	if err != nil {
		logutils.ZapLogger().Error("loadBlockAndTransfersCommand allHistoryBlocksLoaded", zap.Error(err))
		return false, err
	}

	if allBlocksLoaded {
		headers, err := c.blockDAO.GetBlocksToLoadByAddress(c.chainClient.NetworkID(), account, 1)
		if err != nil {
			logutils.ZapLogger().Error("loadBlocksAndTransfersCommand GetFirstSavedBlock", zap.Error(err))
			return false, err
		}

		if len(headers) == 0 {
			return true, nil
		}
	}

	return false, nil
}

// TODO - make it a common method for every service that wants head block number, that will cache the latest block
// and updates it on timeout
func getHeadBlockNumber(parent context.Context, chainClient chain.ClientInterface) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	head, err := chainClient.HeaderByNumber(ctx, nil)
	cancel()
	if err != nil {
		logutils.ZapLogger().Error("getHeadBlockNumber", zap.Error(err))
		return nil, err
	}

	return head.Number, err
}

func nextRange(maxRangeSize int, prevFrom, zeroBlockNumber *big.Int) (*big.Int, *big.Int) {
	logutils.ZapLogger().Debug("next range start",
		zap.Stringer("from", prevFrom),
		zap.Stringer("zeroBlockNumber", zeroBlockNumber),
	)

	rangeSize := big.NewInt(int64(maxRangeSize))

	to := big.NewInt(0).Set(prevFrom)
	from := big.NewInt(0).Sub(to, rangeSize)
	if from.Cmp(zeroBlockNumber) < 0 {
		from = new(big.Int).Set(zeroBlockNumber)
	}

	logutils.ZapLogger().Debug("next range end",
		zap.Stringer("from", from),
		zap.Stringer("to", to),
		zap.Stringer("zeroBlockNumber", zeroBlockNumber),
	)

	return from, to
}

func accountLimiterTag(account common.Address) string {
	return transferHistoryTag + "_" + account.String()
}

func createChainClientWithLimiter(client chain.ClientInterface, account common.Address, limiter rpclimiter.RequestLimiter) (chain.ClientInterface, error) {
	// Each account has its own limit and a global limit for all accounts
	accountTag := accountLimiterTag(account)
	chainClient := chain.ClientWithTag(client, accountTag, transferHistoryTag)

	// Check if limit is already reached, then skip the comamnd
	if allow, err := limiter.Allow(accountTag); !allow {
		logutils.ZapLogger().Info("fetchHistoryBlocksForAccount limit reached",
			zap.Stringer("account", account),
			zap.Uint64("chain", chainClient.NetworkID()),
			zap.Error(err),
		)
		return nil, err
	}

	if allow, err := limiter.Allow(transferHistoryTag); !allow {
		logutils.ZapLogger().Info("fetchHistoryBlocksForAccount common limit reached",
			zap.Uint64("chain", chainClient.NetworkID()),
			zap.Error(err),
		)
		return nil, err
	}

	limit, _ := limiter.GetLimit(accountTag)
	if limit == nil {
		err := limiter.SetLimit(accountTag, transferHistoryLimitPerAccount, rpclimiter.LimitInfinitely)
		if err != nil {
			logutils.ZapLogger().Error("fetchHistoryBlocksForAccount SetLimit",
				zap.String("accountTag", accountTag),
				zap.Error(err),
			)
		}
	}

	// Here total limit per day is overwriten on each app start, that still saves us RPC calls, but allows to proceed
	// after app restart if the limit was reached. Currently there is no way to reset the limit from UI
	err := limiter.SetLimit(transferHistoryTag, transferHistoryLimit, transferHistoryLimitPeriod)
	if err != nil {
		logutils.ZapLogger().Error("fetchHistoryBlocksForAccount SetLimit",
			zap.String("groupTag", transferHistoryTag),
			zap.Error(err),
		)
	}
	chainClient.SetLimiter(limiter)

	return chainClient, nil
}
