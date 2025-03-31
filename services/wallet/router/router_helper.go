package router

import (
	"context"
	"errors"
	"math/big"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/contracts"
	gaspriceproxy "github.com/status-im/status-go/contracts/gas-price-proxy"
	"github.com/status-im/status-go/contracts/hop"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc/chain"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

func (r *Router) requireApproval(ctx context.Context, sendType sendtype.SendType, approvalContractAddress *common.Address, params pathprocessor.ProcessorInputParams) (
	bool, *big.Int, error) {
	if sendType.IsCollectiblesTransfer() || sendType.IsEnsTransfer() || sendType.IsStickersTransfer() {
		return false, nil, nil
	}

	if params.FromToken.IsNative() {
		return false, nil, nil
	}

	contractMaker, err := contracts.NewContractMaker(r.rpcClient)
	if err != nil {
		return false, nil, err
	}

	contract, err := contractMaker.NewERC20(params.FromChain.ChainID, params.FromToken.Address)
	if err != nil {
		return false, nil, err
	}

	if approvalContractAddress == nil || *approvalContractAddress == walletCommon.ZeroAddress() {
		return false, nil, nil
	}

	if params.TestsMode {
		return true, params.AmountIn, nil
	}

	allowance, err := contract.Allowance(&bind.CallOpts{
		Context: ctx,
	}, params.FromAddr, *approvalContractAddress)

	if err != nil {
		return false, nil, err
	}

	if allowance.Cmp(params.AmountIn) >= 0 {
		return false, nil, nil
	}

	return true, params.AmountIn, nil
}

func (r *Router) estimateGasForApproval(params pathprocessor.ProcessorInputParams, input []byte) (uint64, error) {
	ethClient, err := r.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, err
	}

	return ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  input,
	})
}

func (r *Router) calculateL1Fee(chainID uint64, data []byte) (*big.Int, error) {
	ethClient, err := r.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return CalculateL1Fee(chainID, data, ethClient)
}

func CalculateL1Fee(chainID uint64, data []byte, ethClient chain.ClientInterface) (*big.Int, error) {
	oracleContractAddress, err := gaspriceproxy.ContractAddress(chainID)
	if err != nil {
		return nil, err
	}

	proxyContract, err := gaspriceproxy.NewGaspriceproxy(oracleContractAddress, ethClient)
	if err != nil {
		return nil, err
	}

	callOpt := &bind.CallOpts{}

	return proxyContract.GetL1Fee(callOpt, data)
}

func (r *Router) getERC1155Balance(ctx context.Context, network *params.Network, token *tokenTypes.Token, account common.Address) (*big.Int, error) {
	tokenID, success := new(big.Int).SetString(token.Symbol, 10)
	if !success {
		return nil, errors.New("failed to convert token symbol to big.Int")
	}

	balances, err := r.collectiblesManager.FetchERC1155Balances(
		ctx,
		account,
		walletCommon.ChainID(network.ChainID),
		token.Address,
		[]*bigint.BigInt{&bigint.BigInt{Int: tokenID}},
	)
	if err != nil {
		return nil, err
	}

	if len(balances) != 1 || balances[0] == nil {
		return nil, errors.New("invalid ERC1155 balance fetch response")
	}

	return balances[0].Int, nil
}

func (r *Router) getBalance(ctx context.Context, chainID uint64, token *tokenTypes.Token, account common.Address) (*big.Int, error) {
	client, err := r.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return r.tokenManager.GetBalance(ctx, client, account, token.Address)
}

func (r *Router) resolveNonceForPath(ctx context.Context, path *routes.Path, address common.Address, usedNonces map[uint64]uint64) error {
	var nextNonce uint64
	if nonce, ok := usedNonces[path.FromChain.ChainID]; ok {
		nextNonce = nonce + 1
	} else {
		nonce, err := r.transactor.NextNonce(ctx, r.rpcClient, path.FromChain.ChainID, types.Address(address))
		if err != nil {
			return err
		}
		nextNonce = nonce
	}

	usedNonces[path.FromChain.ChainID] = nextNonce
	if !path.ApprovalRequired {
		path.TxNonce = (*hexutil.Uint64)(&nextNonce)
		path.SuggestedTxNonce = (*hexutil.Uint64)(&nextNonce)
	} else {
		path.ApprovalTxNonce = (*hexutil.Uint64)(&nextNonce)
		path.SuggestedApprovalTxNonce = (*hexutil.Uint64)(&nextNonce)
		txNonce := nextNonce + 1
		path.TxNonce = (*hexutil.Uint64)(&txNonce)
		path.SuggestedTxNonce = (*hexutil.Uint64)(&txNonce)

		usedNonces[path.FromChain.ChainID] = txNonce
	}
	return nil
}

func (r *Router) applyCustomFields(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, usedNonces map[uint64]uint64) error {
	r.lastInputParamsMutex.Lock()
	defer r.lastInputParamsMutex.Unlock()

	// set network fields
	if fetchedFees.CurrentBaseFee != nil {
		path.CurrentBaseFee = (*hexutil.Big)(fetchedFees.CurrentBaseFee)
	}
	if fetchedFees.MaxPriorityFeeSuggestedBounds != nil {
		if fetchedFees.MaxPriorityFeeSuggestedBounds.Lower != nil {
			path.SuggestedMinPriorityFee = (*hexutil.Big)(fetchedFees.MaxPriorityFeeSuggestedBounds.Lower)
		}
		if fetchedFees.MaxPriorityFeeSuggestedBounds.Upper != nil {
			path.SuggestedMaxPriorityFee = (*hexutil.Big)(fetchedFees.MaxPriorityFeeSuggestedBounds.Upper)
		}
	}

	// set appropriate gas amount and update later in this function if custom gas amount is provided
	path.TxGasAmount = path.SuggestedTxGasAmount
	path.ApprovalGasAmount = path.SuggestedApprovalGasAmount

	// set appropriate nonce/s, and update later in this function if custom nonce/s are provided
	err := r.resolveNonceForPath(ctx, path, r.lastInputParams.AddrFrom, usedNonces)
	if err != nil {
		return err
	}

	if len(r.lastInputParams.PathTxCustomParams) == 0 {
		// if no custom params are provided, use the initial fee mode
		maxFeesPerGas, priorityFee, estimatedTime, err := fetchedFees.FeeFor(r.lastInputParams.GasFeeMode)
		if err != nil {
			return err
		}
		if path.ApprovalRequired {
			path.ApprovalGasFeeMode = r.lastInputParams.GasFeeMode
			path.ApprovalMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
			path.ApprovalBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
			path.ApprovalPriorityFee = (*hexutil.Big)(priorityFee)
			path.ApprovalEstimatedTime = estimatedTime
		}

		path.TxGasFeeMode = r.lastInputParams.GasFeeMode
		path.TxMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
		path.TxBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
		path.TxPriorityFee = (*hexutil.Big)(priorityFee)
		path.TxEstimatedTime = estimatedTime
	} else {
		if path.ApprovalRequired {
			approvalTxIdentityKey := path.TxIdentityKey(true)
			if approvalTxCustomParams, ok := r.lastInputParams.PathTxCustomParams[approvalTxIdentityKey]; ok {
				path.ApprovalGasFeeMode = approvalTxCustomParams.GasFeeMode
				if approvalTxCustomParams.GasFeeMode != fees.GasFeeCustom {
					maxFeesPerGas, priorityFee, estimatedTime, err := fetchedFees.FeeFor(approvalTxCustomParams.GasFeeMode)
					if err != nil {
						return err
					}
					path.ApprovalMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
					path.ApprovalBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
					path.ApprovalPriorityFee = (*hexutil.Big)(priorityFee)
					path.ApprovalEstimatedTime = estimatedTime
				} else {
					path.ApprovalTxNonce = (*hexutil.Uint64)(&approvalTxCustomParams.Nonce)
					path.ApprovalGasAmount = approvalTxCustomParams.GasAmount
					path.ApprovalMaxFeesPerGas = approvalTxCustomParams.MaxFeesPerGas
					path.ApprovalBaseFee = (*hexutil.Big)(new(big.Int).Sub(approvalTxCustomParams.MaxFeesPerGas.ToInt(), approvalTxCustomParams.PriorityFee.ToInt()))
					path.ApprovalPriorityFee = approvalTxCustomParams.PriorityFee
					path.ApprovalEstimatedTime = r.feesManager.TransactionEstimatedTimeV2(ctx, path.FromChain.ChainID, path.ApprovalMaxFeesPerGas.ToInt(), path.ApprovalPriorityFee.ToInt())
				}
			}
		}

		txIdentityKey := path.TxIdentityKey(false)
		if txCustomParams, ok := r.lastInputParams.PathTxCustomParams[txIdentityKey]; ok {
			path.TxGasFeeMode = txCustomParams.GasFeeMode
			if txCustomParams.GasFeeMode != fees.GasFeeCustom {
				maxFeesPerGas, priorityFee, estimatedTime, err := fetchedFees.FeeFor(txCustomParams.GasFeeMode)
				if err != nil {
					return err
				}
				path.TxMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
				path.TxBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
				path.TxPriorityFee = (*hexutil.Big)(priorityFee)
				path.TxEstimatedTime = estimatedTime
			} else {
				path.TxNonce = (*hexutil.Uint64)(&txCustomParams.Nonce)
				path.TxGasAmount = txCustomParams.GasAmount
				path.TxMaxFeesPerGas = txCustomParams.MaxFeesPerGas
				path.TxBaseFee = (*hexutil.Big)(new(big.Int).Sub(txCustomParams.MaxFeesPerGas.ToInt(), txCustomParams.PriorityFee.ToInt()))
				path.TxPriorityFee = txCustomParams.PriorityFee
				path.TxEstimatedTime = r.feesManager.TransactionEstimatedTimeV2(ctx, path.FromChain.ChainID, path.TxMaxFeesPerGas.ToInt(), path.TxPriorityFee.ToInt())
			}
		}
	}

	return nil
}

func (r *Router) evaluateAndUpdatePathDetails(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees,
	usedNonces map[uint64]uint64, testsMode bool, testApprovalL1Fee uint64) (err error) {

	l1TxFeeWei := big.NewInt(0)
	l1ApprovalFeeWei := big.NewInt(0)

	needL1Fee := path.FromChain.ChainID == walletCommon.OptimismMainnet ||
		path.FromChain.ChainID == walletCommon.OptimismSepolia

	if testsMode {
		usedNonces[path.FromChain.ChainID] = usedNonces[path.FromChain.ChainID] + 1
	}

	if path.ApprovalRequired && needL1Fee {
		if testsMode {
			l1ApprovalFeeWei = big.NewInt(int64(testApprovalL1Fee))
		} else {
			l1ApprovalFeeWei, err = r.calculateL1Fee(path.FromChain.ChainID, path.ApprovalPackedData)
			if err != nil {
				return err
			}
		}
	}

	err = r.applyCustomFields(ctx, path, fetchedFees, usedNonces)
	if err != nil {
		return
	}

	if needL1Fee {
		if !testsMode {
			l1TxFeeWei, err = r.calculateL1Fee(path.FromChain.ChainID, path.TxPackedData)
			if err != nil {
				return err
			}
		}
	}

	// calculate ETH fees
	ethTotalFees := big.NewInt(0)
	txFeeInWei := new(big.Int).Mul(path.TxMaxFeesPerGas.ToInt(), big.NewInt(int64(path.TxGasAmount)))
	ethTotalFees.Add(ethTotalFees, txFeeInWei)
	ethTotalFees.Add(ethTotalFees, l1TxFeeWei)

	approvalFeeInWei := big.NewInt(0)
	if path.ApprovalRequired {
		approvalFeeInWei.Mul(path.ApprovalMaxFeesPerGas.ToInt(), big.NewInt(int64(path.ApprovalGasAmount)))
		ethTotalFees.Add(ethTotalFees, approvalFeeInWei)
		ethTotalFees.Add(ethTotalFees, l1ApprovalFeeWei)
	}

	// calculate required balances (bonder and token fees are already included in the amountIn by Hop bridge (once we include Celar we need to check how they handle the fees))
	requiredNativeBalance := big.NewInt(0)
	requiredTokenBalance := big.NewInt(0)

	if path.FromToken.IsNative() {
		requiredNativeBalance.Add(requiredNativeBalance, path.AmountIn.ToInt())
		if !path.SubtractFees {
			requiredNativeBalance.Add(requiredNativeBalance, ethTotalFees)
		}
	} else {
		requiredTokenBalance.Add(requiredTokenBalance, path.AmountIn.ToInt())
		requiredNativeBalance.Add(requiredNativeBalance, ethTotalFees)
	}

	// set the values
	path.SuggestedLevelsForMaxFeesPerGas = fetchedFees.MaxFeesLevels

	path.TxFee = (*hexutil.Big)(txFeeInWei)
	path.TxL1Fee = (*hexutil.Big)(l1TxFeeWei)

	path.ApprovalFee = (*hexutil.Big)(approvalFeeInWei)
	path.ApprovalL1Fee = (*hexutil.Big)(l1ApprovalFeeWei)

	path.TxTotalFee = (*hexutil.Big)(ethTotalFees)

	path.RequiredTokenBalance = requiredTokenBalance
	path.RequiredNativeBalance = requiredNativeBalance

	return
}

func ParseCollectibleID(ID string) (contractAddress common.Address, tokenID *big.Int, success bool) {
	success = false

	parts := strings.Split(ID, ":")
	if len(parts) != 2 {
		return
	}
	contractAddress = common.HexToAddress(parts[0])
	tokenID, success = new(big.Int).SetString(parts[1], 10)
	return
}

func findToken(sendType sendtype.SendType, tokenManager *token.Manager, collectibles *collectibles.Service, account common.Address, network *params.Network, tokenID string) *tokenTypes.Token {
	if !sendType.IsCollectiblesTransfer() {
		return tokenManager.FindToken(network, tokenID)
	}

	if sendType.IsCommunityRelatedTransfer() {
		// TODO: optimize tokens to handle community tokens
		return nil
	}

	contractAddress, collectibleTokenID, success := ParseCollectibleID(tokenID)
	if !success {
		return nil
	}
	uniqueID, err := collectibles.GetOwnedCollectible(walletCommon.ChainID(network.ChainID), account, contractAddress, collectibleTokenID)
	if err != nil || uniqueID == nil {
		return nil
	}

	return &tokenTypes.Token{
		Address:  contractAddress,
		Symbol:   collectibleTokenID.String(),
		Decimals: 0,
		ChainID:  network.ChainID,
	}
}

func fetchPrices(sendType sendtype.SendType, marketManager *market.Manager, tokenIDs []string) (map[string]float64, error) {
	nonUniqueSymbols := append(tokenIDs, "ETH")
	// remove duplicate enteries
	slices.Sort(nonUniqueSymbols)
	symbols := slices.Compact(nonUniqueSymbols)
	if sendType.IsCollectiblesTransfer() {
		symbols = []string{"ETH"}
	}

	pricesMap, err := marketManager.GetOrFetchPrices(symbols, []string{"USD"}, market.MaxAgeInSecondsForFresh)

	if err != nil {
		return nil, err
	}
	prices := make(map[string]float64, 0)
	for symbol, pricePerCurrency := range pricesMap {
		prices[symbol] = pricePerCurrency["USD"].Price
	}
	if sendType.IsCollectiblesTransfer() {
		for _, tokenID := range tokenIDs {
			prices[tokenID] = 0
		}
	}
	return prices, nil
}

func (r *Router) GetTokensAvailableForBridgeOnChain(chainID uint64) []*tokenTypes.Token {
	symbols := hop.GetSymbolsAvailableOnChain(chainID)

	tokens := make([]*tokenTypes.Token, 0)
	for _, symbol := range symbols {
		t, _ := r.tokenManager.LookupToken(&chainID, symbol)
		if t == nil {
			continue
		}
		tokens = append(tokens, t)
	}
	return tokens
}
