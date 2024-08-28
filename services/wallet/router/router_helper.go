package router

import (
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/contracts"
	gaspriceoracle "github.com/status-im/status-go/contracts/gas-price-oracle"
	"github.com/status-im/status-go/contracts/ierc20"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/bigint"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/token"
)

func (r *Router) requireApproval(ctx context.Context, sendType sendtype.SendType, approvalContractAddress *common.Address, params pathprocessor.ProcessorInputParams) (
	bool, *big.Int, uint64, uint64, error) {
	if sendType.IsCollectiblesTransfer() || sendType.IsEnsTransfer() || sendType.IsStickersTransfer() {
		return false, nil, 0, 0, nil
	}

	if params.FromToken.IsNative() {
		return false, nil, 0, 0, nil
	}

	contractMaker, err := contracts.NewContractMaker(r.rpcClient)
	if err != nil {
		return false, nil, 0, 0, err
	}

	contract, err := contractMaker.NewERC20(params.FromChain.ChainID, params.FromToken.Address)
	if err != nil {
		return false, nil, 0, 0, err
	}

	if approvalContractAddress == nil || *approvalContractAddress == pathprocessor.ZeroAddress {
		return false, nil, 0, 0, nil
	}

	if params.TestsMode {
		return true, params.AmountIn, params.TestApprovalGasEstimation, params.TestApprovalL1Fee, nil
	}

	allowance, err := contract.Allowance(&bind.CallOpts{
		Context: ctx,
	}, params.FromAddr, *approvalContractAddress)

	if err != nil {
		return false, nil, 0, 0, err
	}

	if allowance.Cmp(params.AmountIn) >= 0 {
		return false, nil, 0, 0, nil
	}

	ethClient, err := r.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return false, nil, 0, 0, err
	}

	erc20ABI, err := abi.JSON(strings.NewReader(ierc20.IERC20ABI))
	if err != nil {
		return false, nil, 0, 0, err
	}

	data, err := erc20ABI.Pack("approve", approvalContractAddress, params.AmountIn)
	if err != nil {
		return false, nil, 0, 0, err
	}

	estimate, err := ethClient.EstimateGas(context.Background(), ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &params.FromToken.Address,
		Value: pathprocessor.ZeroBigIntValue,
		Data:  data,
	})
	if err != nil {
		return false, nil, 0, 0, err
	}

	// fetching l1 fee
	var l1Fee uint64
	oracleContractAddress, err := gaspriceoracle.ContractAddress(params.FromChain.ChainID)
	if err == nil {
		oracleContract, err := gaspriceoracle.NewGaspriceoracleCaller(oracleContractAddress, ethClient)
		if err != nil {
			return false, nil, 0, 0, err
		}

		callOpt := &bind.CallOpts{}

		l1FeeResult, _ := oracleContract.GetL1Fee(callOpt, data)
		l1Fee = l1FeeResult.Uint64()
	}

	return true, params.AmountIn, estimate, l1Fee, nil
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
