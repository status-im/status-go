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
	"github.com/status-im/status-go/services/wallet/requests"
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

func (r *Router) resolveSuggestedNonceForPath(ctx context.Context, path *routes.Path, address common.Address, usedNonces map[uint64]uint64) error {
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
		path.SuggestedTxNonce = (*hexutil.Uint64)(&nextNonce)
	} else {
		path.SuggestedApprovalTxNonce = (*hexutil.Uint64)(&nextNonce)
		txNonce := nextNonce + 1
		path.SuggestedTxNonce = (*hexutil.Uint64)(&txNonce)

		usedNonces[path.FromChain.ChainID] = txNonce
	}
	return nil
}

// applyCustomFields applies custom fields to the path based on fetched fees and used nonces
func (r *Router) applyCustomFields(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, usedNonces map[uint64]uint64) error {
	r.lastInputParamsMutex.Lock()
	defer r.lastInputParamsMutex.Unlock()

	eipP1559EnabledChain := path.FromChain.EIP1559Enabled

	if err := r.setSuggestedFields(ctx, path, fetchedFees, usedNonces, eipP1559EnabledChain); err != nil {
		return err
	}

	if err := r.setPathFields(path, fetchedFees); err != nil {
		return err
	}

	// Apply fee modes and custom parameters
	if len(r.lastInputParams.PathTxCustomParams) == 0 {
		return r.applyDefaultFeeModes(path, fetchedFees, eipP1559EnabledChain)
	}
	return r.applyCustomFeeModes(ctx, path, fetchedFees, eipP1559EnabledChain)
}

// setSuggestedFields sets suggested fee fields
func (r *Router) setSuggestedFields(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, usedNonces map[uint64]uint64, eipP1559EnabledChain bool) error {
	if !eipP1559EnabledChain {
		path.SuggestedNonEIP1559Fees = fetchedFees.NonEIP1559Fees
	} else {
		path.SuggestedLevelsForMaxFeesPerGas = fetchedFees.MaxFeesLevels
		if fetchedFees.MaxPriorityFeeSuggestedBounds != nil {
			if fetchedFees.MaxPriorityFeeSuggestedBounds.Lower != nil {
				path.SuggestedMinPriorityFee = (*hexutil.Big)(fetchedFees.MaxPriorityFeeSuggestedBounds.Lower)
			}
			if fetchedFees.MaxPriorityFeeSuggestedBounds.Upper != nil {
				path.SuggestedMaxPriorityFee = (*hexutil.Big)(fetchedFees.MaxPriorityFeeSuggestedBounds.Upper)
			}
		}
	}

	return r.resolveSuggestedNonceForPath(ctx, path, r.lastInputParams.AddrFrom, usedNonces)
}

// setPathFields sets path fields
func (r *Router) setPathFields(path *routes.Path, fetchedFees *fees.SuggestedFees) error {
	if fetchedFees.CurrentBaseFee != nil {
		path.CurrentBaseFee = (*hexutil.Big)(fetchedFees.CurrentBaseFee)
	}

	path.TxGasAmount = path.SuggestedTxGasAmount
	path.ApprovalGasAmount = path.SuggestedApprovalGasAmount
	path.TxNonce = path.SuggestedTxNonce
	path.ApprovalTxNonce = path.SuggestedApprovalTxNonce

	return nil
}

// applyDefaultFeeModes applies default fee modes to the path
func (r *Router) applyDefaultFeeModes(path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool) error {
	if !eipP1559EnabledChain {
		return r.applyDefaultNonEIP1559Fees(path, fetchedFees)
	}
	return r.applyDefaultEIP1559Fees(path, fetchedFees)
}

// applyDefaultNonEIP1559Fees applies default non-EIP1559 fees
func (r *Router) applyDefaultNonEIP1559Fees(path *routes.Path, fetchedFees *fees.SuggestedFees) error {
	if path.ApprovalRequired {
		path.ApprovalGasFeeMode = r.lastInputParams.GasFeeMode
		path.ApprovalGasPrice = fetchedFees.NonEIP1559Fees.GasPrice
		path.ApprovalEstimatedTime = fetchedFees.NonEIP1559Fees.EstimatedTime
	}

	path.TxGasFeeMode = r.lastInputParams.GasFeeMode
	path.TxGasPrice = fetchedFees.NonEIP1559Fees.GasPrice
	path.TxEstimatedTime = fetchedFees.NonEIP1559Fees.EstimatedTime
	return nil
}

// applyDefaultEIP1559Fees applies default EIP1559 fees
func (r *Router) applyDefaultEIP1559Fees(path *routes.Path, fetchedFees *fees.SuggestedFees) error {
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
	return nil
}

// applyCustomFeeModes applies custom fee modes to the path
func (r *Router) applyCustomFeeModes(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool) error {
	if path.ApprovalRequired {
		if err := r.applyCustomApprovalFees(ctx, path, fetchedFees, eipP1559EnabledChain); err != nil {
			return err
		}
	}

	return r.applyCustomTxFees(ctx, path, fetchedFees, eipP1559EnabledChain)
}

// applyCustomApprovalFees applies custom fees for approval transaction
func (r *Router) applyCustomApprovalFees(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool) error {
	approvalTxIdentityKey := path.TxIdentityKey(true)
	approvalTxCustomParams, ok := r.lastInputParams.PathTxCustomParams[approvalTxIdentityKey]
	if !ok {
		return nil
	}

	path.ApprovalGasFeeMode = approvalTxCustomParams.GasFeeMode
	if approvalTxCustomParams.GasFeeMode != fees.GasFeeCustom {
		return r.applyNonCustomApprovalFees(path, fetchedFees, eipP1559EnabledChain, approvalTxCustomParams)
	}
	return r.applyCustomApprovalFeeMode(ctx, path, fetchedFees, eipP1559EnabledChain, approvalTxCustomParams)
}

// applyNonCustomApprovalFees applies non-custom fees for approval transaction
func (r *Router) applyNonCustomApprovalFees(path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool, params *requests.PathTxCustomParams) error {
	if !eipP1559EnabledChain {
		path.ApprovalGasPrice = fetchedFees.NonEIP1559Fees.GasPrice
		path.ApprovalEstimatedTime = fetchedFees.NonEIP1559Fees.EstimatedTime
		return nil
	}

	maxFeesPerGas, priorityFee, estimatedTime, err := fetchedFees.FeeFor(params.GasFeeMode)
	if err != nil {
		return err
	}

	path.ApprovalMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
	path.ApprovalBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
	path.ApprovalPriorityFee = (*hexutil.Big)(priorityFee)
	path.ApprovalEstimatedTime = estimatedTime
	return nil
}

// applyCustomApprovalFeeMode applies custom fee mode for approval transaction
func (r *Router) applyCustomApprovalFeeMode(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool, params *requests.PathTxCustomParams) error {
	path.ApprovalTxNonce = (*hexutil.Uint64)(&params.Nonce)
	path.ApprovalGasAmount = params.GasAmount

	if !eipP1559EnabledChain {
		path.ApprovalGasPrice = params.GasPrice
		path.ApprovalEstimatedTime = r.feesManager.TransactionEstimatedTimeV2Legacy(ctx, path.FromChain.ChainID, path.ApprovalGasPrice.ToInt())
		return nil
	}

	path.ApprovalMaxFeesPerGas = params.MaxFeesPerGas
	path.ApprovalBaseFee = (*hexutil.Big)(new(big.Int).Sub(params.MaxFeesPerGas.ToInt(), params.PriorityFee.ToInt()))
	path.ApprovalPriorityFee = params.PriorityFee
	path.ApprovalEstimatedTime = r.feesManager.TransactionEstimatedTimeV2(ctx, path.FromChain.ChainID, path.ApprovalMaxFeesPerGas.ToInt(), path.ApprovalPriorityFee.ToInt())
	return nil
}

// applyCustomTxFees applies custom fees for main transaction
func (r *Router) applyCustomTxFees(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool) error {
	txIdentityKey := path.TxIdentityKey(false)
	txCustomParams, ok := r.lastInputParams.PathTxCustomParams[txIdentityKey]
	if !ok {
		return nil
	}

	path.TxGasFeeMode = txCustomParams.GasFeeMode
	if txCustomParams.GasFeeMode != fees.GasFeeCustom {
		return r.applyNonCustomTxFees(path, fetchedFees, eipP1559EnabledChain, txCustomParams)
	}
	return r.applyCustomTxFeeMode(ctx, path, fetchedFees, eipP1559EnabledChain, txCustomParams)
}

// applyNonCustomTxFees applies non-custom fees for main transaction
func (r *Router) applyNonCustomTxFees(path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool, params *requests.PathTxCustomParams) error {
	if !eipP1559EnabledChain {
		path.TxGasPrice = fetchedFees.NonEIP1559Fees.GasPrice
		path.TxEstimatedTime = fetchedFees.NonEIP1559Fees.EstimatedTime
		return nil
	}

	maxFeesPerGas, priorityFee, estimatedTime, err := fetchedFees.FeeFor(params.GasFeeMode)
	if err != nil {
		return err
	}

	path.TxMaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)
	path.TxBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
	path.TxPriorityFee = (*hexutil.Big)(priorityFee)
	path.TxEstimatedTime = estimatedTime
	return nil
}

// applyCustomTxFeeMode applies custom fee mode for main transaction
func (r *Router) applyCustomTxFeeMode(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees, eipP1559EnabledChain bool, params *requests.PathTxCustomParams) error {
	path.TxNonce = (*hexutil.Uint64)(&params.Nonce)
	path.TxGasAmount = params.GasAmount

	if !eipP1559EnabledChain {
		path.TxGasPrice = params.GasPrice
		path.TxEstimatedTime = r.feesManager.TransactionEstimatedTimeV2Legacy(ctx, path.FromChain.ChainID, path.TxGasPrice.ToInt())
		return nil
	}

	path.TxMaxFeesPerGas = params.MaxFeesPerGas
	path.TxBaseFee = (*hexutil.Big)(new(big.Int).Sub(params.MaxFeesPerGas.ToInt(), params.PriorityFee.ToInt()))
	path.TxPriorityFee = params.PriorityFee
	path.TxEstimatedTime = r.feesManager.TransactionEstimatedTimeV2(ctx, path.FromChain.ChainID, path.TxMaxFeesPerGas.ToInt(), path.TxPriorityFee.ToInt())
	return nil
}

func (r *Router) evaluateAndUpdatePathDetails(ctx context.Context, path *routes.Path, fetchedFees *fees.SuggestedFees,
	usedNonces map[uint64]uint64, testsMode bool, testApprovalL1Fee uint64) (err error) {

	path.FromChain.EIP1559Enabled = fetchedFees.EIP1559Enabled

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
	var txFeeInWei *big.Int
	if !path.FromChain.EIP1559Enabled {
		txFeeInWei = new(big.Int).Mul(path.TxGasPrice.ToInt(), big.NewInt(int64(path.TxGasAmount)))
	} else {
		txFeeInWei = new(big.Int).Mul(path.TxMaxFeesPerGas.ToInt(), big.NewInt(int64(path.TxGasAmount)))
	}
	ethTotalFees.Add(ethTotalFees, txFeeInWei)
	ethTotalFees.Add(ethTotalFees, l1TxFeeWei)

	approvalFeeInWei := big.NewInt(0)
	if path.ApprovalRequired {
		if !path.FromChain.EIP1559Enabled {
			approvalFeeInWei.Mul(path.ApprovalGasPrice.ToInt(), big.NewInt(int64(path.ApprovalGasAmount)))
		} else {
			approvalFeeInWei.Mul(path.ApprovalMaxFeesPerGas.ToInt(), big.NewInt(int64(path.ApprovalGasAmount)))
		}
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
	nonUniqueSymbols := append(tokenIDs, "ETH", "BNB")
	// remove duplicate enteries
	slices.Sort(nonUniqueSymbols)
	symbols := slices.Compact(nonUniqueSymbols)
	if sendType.IsCollectiblesTransfer() {
		symbols = []string{"ETH", "BNB"}
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
