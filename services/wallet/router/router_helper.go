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
	gaspriceoracle "github.com/status-im/status-go/contracts/gas-price-oracle"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/services/wallet/collectibles"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/market"
	"github.com/status-im/status-go/services/wallet/router/fees"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	routs "github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/token"
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

func (r *Router) estimateGasForApproval(params pathprocessor.ProcessorInputParams, approvalContractAddress *common.Address) (uint64, error) {
	data, err := walletCommon.PackApprovalInputData(params.AmountIn, approvalContractAddress)
	if err != nil {
		return 0, err
	}

	ethClient, err := r.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, err
	}

	return ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: walletCommon.ZeroBigIntValue(),
		Data:  data,
	})
}

func (r *Router) calculateApprovalL1Fee(amountIn *big.Int, chainID uint64, approvalContractAddress *common.Address) (uint64, error) {
	data, err := walletCommon.PackApprovalInputData(amountIn, approvalContractAddress)
	if err != nil {
		return 0, err
	}

	ethClient, err := r.rpcClient.EthClient(chainID)
	if err != nil {
		return 0, err
	}

	var l1Fee uint64
	oracleContractAddress, err := gaspriceoracle.ContractAddress(chainID)
	if err == nil {
		oracleContract, err := gaspriceoracle.NewGaspriceoracleCaller(oracleContractAddress, ethClient)
		if err != nil {
			return 0, err
		}

		callOpt := &bind.CallOpts{}

		l1FeeResult, _ := oracleContract.GetL1Fee(callOpt, data)
		l1Fee = l1FeeResult.Uint64()
	}

	return l1Fee, nil
}

func (r *Router) getERC1155Balance(ctx context.Context, network *params.Network, token *token.Token, account common.Address) (*big.Int, error) {
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

func (r *Router) getBalance(ctx context.Context, chainID uint64, token *token.Token, account common.Address) (*big.Int, error) {
	client, err := r.rpcClient.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return r.tokenManager.GetBalance(ctx, client, account, token.Address)
}

func (r *Router) cacluateFees(ctx context.Context, path *routs.Path, fetchedFees *fees.SuggestedFees, testsMode bool, testApprovalL1Fee uint64) (err error) {

	var (
		l1ApprovalFee uint64
	)
	if path.ApprovalRequired {
		if testsMode {
			l1ApprovalFee = testApprovalL1Fee
		} else {
			l1ApprovalFee, err = r.calculateApprovalL1Fee(path.AmountIn.ToInt(), path.FromChain.ChainID, path.ApprovalContractAddress)
			if err != nil {
				return err
			}
		}
	}

	// TODO: keep l1 fees at 0 until we have the correct algorithm, as we do base fee x 2 that should cover the l1 fees
	var l1FeeWei uint64 = 0
	// if input.SendType.needL1Fee() {
	// 	txInputData, err := pProcessor.PackTxInputData(processorInputParams)
	// 	if err != nil {
	// 		continue
	// 	}

	// 	l1FeeWei, _ = r.feesManager.GetL1Fee(ctx, network.ChainID, txInputData)
	// }

	r.lastInputParamsMutex.Lock()
	gasFeeMode := r.lastInputParams.GasFeeMode
	r.lastInputParamsMutex.Unlock()
	maxFeesPerGas := fetchedFees.FeeFor(gasFeeMode)

	// calculate ETH fees
	ethTotalFees := big.NewInt(0)
	txFeeInWei := new(big.Int).Mul(maxFeesPerGas, big.NewInt(int64(path.TxGasAmount)))
	ethTotalFees.Add(ethTotalFees, txFeeInWei)

	txL1FeeInWei := big.NewInt(0)
	if l1FeeWei > 0 {
		txL1FeeInWei = big.NewInt(int64(l1FeeWei))
		ethTotalFees.Add(ethTotalFees, txL1FeeInWei)
	}

	approvalFeeInWei := big.NewInt(0)
	approvalL1FeeInWei := big.NewInt(0)
	if path.ApprovalRequired {
		approvalFeeInWei.Mul(maxFeesPerGas, big.NewInt(int64(path.ApprovalGasAmount)))
		ethTotalFees.Add(ethTotalFees, approvalFeeInWei)

		if l1ApprovalFee > 0 {
			approvalL1FeeInWei = big.NewInt(int64(l1ApprovalFee))
			ethTotalFees.Add(ethTotalFees, approvalL1FeeInWei)
		}
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
	path.MaxFeesPerGas = (*hexutil.Big)(maxFeesPerGas)

	path.TxBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
	path.TxPriorityFee = (*hexutil.Big)(fetchedFees.MaxPriorityFeePerGas)

	path.TxFee = (*hexutil.Big)(txFeeInWei)
	path.TxL1Fee = (*hexutil.Big)(txL1FeeInWei)

	path.ApprovalBaseFee = (*hexutil.Big)(fetchedFees.BaseFee)
	path.ApprovalPriorityFee = (*hexutil.Big)(fetchedFees.MaxPriorityFeePerGas)

	path.ApprovalFee = (*hexutil.Big)(approvalFeeInWei)
	path.ApprovalL1Fee = (*hexutil.Big)(approvalL1FeeInWei)

	path.TxTotalFee = (*hexutil.Big)(ethTotalFees)

	path.RequiredTokenBalance = requiredTokenBalance
	path.RequiredNativeBalance = requiredNativeBalance

	return nil
}

func findToken(sendType sendtype.SendType, tokenManager *token.Manager, collectibles *collectibles.Service, account common.Address, network *params.Network, tokenID string) *token.Token {
	if !sendType.IsCollectiblesTransfer() {
		return tokenManager.FindToken(network, tokenID)
	}

	parts := strings.Split(tokenID, ":")
	contractAddress := common.HexToAddress(parts[0])
	collectibleTokenID, success := new(big.Int).SetString(parts[1], 10)
	if !success {
		return nil
	}
	uniqueID, err := collectibles.GetOwnedCollectible(walletCommon.ChainID(network.ChainID), account, contractAddress, collectibleTokenID)
	if err != nil || uniqueID == nil {
		return nil
	}

	return &token.Token{
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
