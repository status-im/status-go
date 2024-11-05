package pathprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts/celer"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/rpc"

	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor/cbridge"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/transactions"
)

const (
	baseURL     = "https://cbridge-prod2.celer.app"
	testBaseURL = "https://cbridge-v2-test.celer.network"

	maxSlippage = uint32(1000)
)

type CelerBridgeTxArgs struct {
	wallettypes.SendTxArgs
	ChainID   uint64         `json:"chainId"`
	Symbol    string         `json:"symbol"`
	Recipient common.Address `json:"recipient"`
	Amount    *hexutil.Big   `json:"amount"`
}

type CelerBridgeProcessor struct {
	rpcClient          *rpc.Client
	httpClient         *thirdparty.HTTPClient
	transactor         transactions.TransactorIface
	tokenManager       *token.Manager
	prodTransferConfig *cbridge.GetTransferConfigsResponse
	testTransferConfig *cbridge.GetTransferConfigsResponse
}

func NewCelerBridgeProcessor(rpcClient *rpc.Client, transactor transactions.TransactorIface, tokenManager *token.Manager) *CelerBridgeProcessor {
	return &CelerBridgeProcessor{
		rpcClient:    rpcClient,
		httpClient:   thirdparty.NewHTTPClient(),
		transactor:   transactor,
		tokenManager: tokenManager,
	}
}

func createBridgeCellerErrorResponse(err error) error {
	return createErrorResponse(walletCommon.ProcessorBridgeCelerName, err)
}

func (s *CelerBridgeProcessor) Name() string {
	return walletCommon.ProcessorBridgeCelerName
}

func (s *CelerBridgeProcessor) estimateAmt(from, to *params.Network, amountIn *big.Int, symbol string) (*cbridge.EstimateAmtResponse, error) {
	base := baseURL
	if from.IsTest {
		base = testBaseURL
	}

	params := url.Values{}
	params.Add("src_chain_id", strconv.Itoa(int(from.ChainID)))
	params.Add("dst_chain_id", strconv.Itoa(int(to.ChainID)))
	params.Add("token_symbol", symbol)
	params.Add("amt", amountIn.String())
	params.Add("usr_addr", "0xaa47c83316edc05cf9ff7136296b026c5de7eccd")
	params.Add("slippage_tolerance", "500")

	url := fmt.Sprintf("%s/v2/estimateAmt", base)
	response, err := s.httpClient.DoGetRequest(context.Background(), url, params, nil)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}

	var res cbridge.EstimateAmtResponse
	err = json.Unmarshal(response, &res)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}
	return &res, nil
}

func (s *CelerBridgeProcessor) getTransferConfig(isTest bool) (*cbridge.GetTransferConfigsResponse, error) {
	if !isTest && s.prodTransferConfig != nil {
		return s.prodTransferConfig, nil
	}

	if isTest && s.testTransferConfig != nil {
		return s.testTransferConfig, nil
	}

	base := baseURL
	if isTest {
		base = testBaseURL
	}
	url := fmt.Sprintf("%s/v2/getTransferConfigs", base)
	response, err := s.httpClient.DoGetRequest(context.Background(), url, nil, nil)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}

	var res cbridge.GetTransferConfigsResponse
	err = json.Unmarshal(response, &res)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}
	if isTest {
		s.testTransferConfig = &res
	} else {
		s.prodTransferConfig = &res
	}
	return &res, nil
}

func (s *CelerBridgeProcessor) AvailableFor(params ProcessorInputParams) (bool, error) {
	if params.FromChain.ChainID == params.ToChain.ChainID || params.ToToken != nil {
		return false, nil
	}

	transferConfig, err := s.getTransferConfig(params.FromChain.IsTest)
	if err != nil {
		return false, createBridgeCellerErrorResponse(err)
	}
	if transferConfig.Err != nil {
		return false, createBridgeCellerErrorResponse(errors.New(transferConfig.Err.Msg))
	}

	var fromAvailable *cbridge.Chain
	var toAvailable *cbridge.Chain
	for _, chain := range transferConfig.Chains {
		if uint64(chain.GetId()) == params.FromChain.ChainID && chain.GasTokenSymbol == walletCommon.EthSymbol {
			fromAvailable = chain
		}

		if uint64(chain.GetId()) == params.ToChain.ChainID && chain.GasTokenSymbol == walletCommon.EthSymbol {
			toAvailable = chain
		}
	}

	if fromAvailable == nil || toAvailable == nil {
		return false, nil
	}

	found := false
	if _, ok := transferConfig.ChainToken[fromAvailable.GetId()]; !ok {
		return false, nil
	}

	for _, tokenInfo := range transferConfig.ChainToken[fromAvailable.GetId()].Token {
		if tokenInfo.Token.Symbol == params.FromToken.Symbol {
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}

	found = false
	for _, tokenInfo := range transferConfig.ChainToken[toAvailable.GetId()].Token {
		if tokenInfo.Token.Symbol == params.FromToken.Symbol {
			found = true
			break
		}
	}

	if !found {
		return false, nil
	}

	return true, nil
}

func (s *CelerBridgeProcessor) CalculateFees(params ProcessorInputParams) (*big.Int, *big.Int, error) {
	amt, err := s.estimateAmt(params.FromChain, params.ToChain, params.AmountIn, params.FromToken.Symbol)
	if err != nil {
		return nil, nil, createBridgeCellerErrorResponse(err)
	}
	baseFee, ok := new(big.Int).SetString(amt.BaseFee, 10)
	if !ok {
		return nil, nil, ErrFailedToParseBaseFee
	}
	percFee, ok := new(big.Int).SetString(amt.PercFee, 10)
	if !ok {
		return nil, nil, ErrFailedToParsePercentageFee
	}

	return walletCommon.ZeroBigIntValue(), new(big.Int).Add(baseFee, percFee), nil
}

func (c *CelerBridgeProcessor) PackTxInputData(params ProcessorInputParams) ([]byte, error) {
	abi, err := abi.JSON(strings.NewReader(celer.CelerABI))
	if err != nil {
		return []byte{}, createBridgeCellerErrorResponse(err)
	}

	if params.FromToken.IsNative() {
		return abi.Pack("sendNative",
			params.ToAddr,
			params.AmountIn,
			params.ToChain.ChainID,
			uint64(time.Now().UnixMilli()),
			maxSlippage,
		)
	} else {
		return abi.Pack("send",
			params.ToAddr,
			params.FromToken.Address,
			params.AmountIn,
			params.ToChain.ChainID,
			uint64(time.Now().UnixMilli()),
			maxSlippage,
		)
	}
}

func (s *CelerBridgeProcessor) EstimateGas(params ProcessorInputParams) (uint64, error) {
	if params.TestsMode {
		if params.TestEstimationMap != nil {
			if val, ok := params.TestEstimationMap[s.Name()]; ok {
				return val.Value, val.Err
			}
		}
		return 0, ErrNoEstimationFound
	}

	value := new(big.Int)

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, createBridgeCellerErrorResponse(err)
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, createBridgeCellerErrorResponse(err)
	}

	ethClient, err := s.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, createBridgeCellerErrorResponse(err)
	}

	ctx := context.Background()

	msg := ethereum.CallMsg{
		From:  params.FromAddr,
		To:    &contractAddress,
		Value: value,
		Data:  input,
	}

	estimation, err := ethClient.EstimateGas(ctx, msg)
	if err != nil {
		if !params.FromToken.IsNative() {
			// TODO: this is a temporary solution until we find a better way to estimate the gas
			// hardcoding the estimation for other than ETH, cause we cannot get a proper estimation without having an approval placed first
			// this is an error we're facing otherwise: `execution reverted: ERC20: transfer amount exceeds allowance`
			estimation = 350000
		} else {
			return 0, createBridgeCellerErrorResponse(err)
		}
	}
	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *CelerBridgeProcessor) GetContractAddress(params ProcessorInputParams) (common.Address, error) {
	transferConfig, err := s.getTransferConfig(params.FromChain.IsTest)
	if err != nil {
		return common.Address{}, createBridgeCellerErrorResponse(err)
	}
	if transferConfig.Err != nil {
		return common.Address{}, createBridgeCellerErrorResponse(errors.New(transferConfig.Err.Msg))
	}

	for _, chain := range transferConfig.Chains {
		if uint64(chain.Id) == params.FromChain.ChainID {
			return common.HexToAddress(chain.ContractAddr), nil
		}
	}

	return common.Address{}, ErrContractNotFound
}

// TODO: remove this struct once mobile switches to the new approach
func (s *CelerBridgeProcessor) sendOrBuild(sendArgs *MultipathProcessorTxArgs, signerFn bind.SignerFn, lastUsedNonce int64) (*ethTypes.Transaction, error) {
	fromChain := s.rpcClient.NetworkManager.Find(sendArgs.ChainID)
	if fromChain == nil {
		return nil, ErrNetworkNotFound
	}
	token := s.tokenManager.FindToken(fromChain, sendArgs.CbridgeTx.Symbol)
	if token == nil {
		return nil, ErrTokenNotFound
	}
	addrs, err := s.GetContractAddress(ProcessorInputParams{
		FromChain: fromChain,
	})
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}

	backend, err := s.rpcClient.EthClient(sendArgs.ChainID)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}
	contract, err := celer.NewCeler(addrs, backend)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}

	if lastUsedNonce >= 0 {
		lastUsedNonceHexUtil := hexutil.Uint64(uint64(lastUsedNonce) + 1)
		sendArgs.CbridgeTx.Nonce = &lastUsedNonceHexUtil
	}

	var tx *ethTypes.Transaction
	txOpts := sendArgs.CbridgeTx.ToTransactOpts(signerFn)
	if token.IsNative() {
		tx, err = contract.SendNative(
			txOpts,
			sendArgs.CbridgeTx.Recipient,
			(*big.Int)(sendArgs.CbridgeTx.Amount),
			sendArgs.CbridgeTx.ChainID,
			uint64(time.Now().UnixMilli()),
			maxSlippage,
		)
	} else {
		tx, err = contract.Send(
			txOpts,
			sendArgs.CbridgeTx.Recipient,
			token.Address,
			(*big.Int)(sendArgs.CbridgeTx.Amount),
			sendArgs.CbridgeTx.ChainID,
			uint64(time.Now().UnixMilli()),
			maxSlippage,
		)
	}
	if err != nil {
		return tx, createBridgeCellerErrorResponse(err)
	}
	err = s.transactor.StoreAndTrackPendingTx(txOpts.From, sendArgs.CbridgeTx.Symbol, sendArgs.ChainID, sendArgs.CbridgeTx.MultiTransactionID, tx)
	if err != nil {
		return tx, createBridgeCellerErrorResponse(err)
	}
	return tx, nil
}

func (s *CelerBridgeProcessor) sendOrBuildV2(sendArgs *wallettypes.SendTxArgs, signerFn bind.SignerFn, lastUsedNonce int64) (*ethTypes.Transaction, error) {
	fromChain := s.rpcClient.NetworkManager.Find(sendArgs.FromChainID)
	if fromChain == nil {
		return nil, ErrNetworkNotFound
	}
	token := s.tokenManager.FindToken(fromChain, sendArgs.FromTokenID)
	if token == nil {
		return nil, ErrTokenNotFound
	}
	addrs, err := s.GetContractAddress(ProcessorInputParams{
		FromChain: fromChain,
	})
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}

	backend, err := s.rpcClient.EthClient(sendArgs.FromChainID)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}
	contract, err := celer.NewCeler(addrs, backend)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}

	if lastUsedNonce >= 0 {
		lastUsedNonceHexUtil := hexutil.Uint64(uint64(lastUsedNonce) + 1)
		sendArgs.Nonce = &lastUsedNonceHexUtil
	}

	var tx *ethTypes.Transaction
	txOpts := sendArgs.ToTransactOpts(signerFn)
	if token.IsNative() {
		tx, err = contract.SendNative(
			txOpts,
			common.Address(*sendArgs.To),
			(*big.Int)(sendArgs.Value),
			sendArgs.FromChainID,
			uint64(time.Now().UnixMilli()),
			maxSlippage,
		)
	} else {
		tx, err = contract.Send(
			txOpts,
			common.Address(*sendArgs.To),
			token.Address,
			(*big.Int)(sendArgs.Value),
			sendArgs.FromChainID,
			uint64(time.Now().UnixMilli()),
			maxSlippage,
		)
	}
	if err != nil {
		return tx, createBridgeCellerErrorResponse(err)
	}
	err = s.transactor.StoreAndTrackPendingTx(txOpts.From, sendArgs.FromTokenID, sendArgs.FromChainID, sendArgs.MultiTransactionID, tx)
	if err != nil {
		return tx, createBridgeCellerErrorResponse(err)
	}
	return tx, nil
}

func (s *CelerBridgeProcessor) Send(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64, verifiedAccount *account.SelectedExtKey) (types.Hash, uint64, error) {
	tx, err := s.sendOrBuild(sendArgs, getSigner(sendArgs.ChainID, sendArgs.CbridgeTx.From, verifiedAccount), lastUsedNonce)
	if err != nil {
		return types.HexToHash(""), 0, createBridgeCellerErrorResponse(err)
	}

	return types.Hash(tx.Hash()), tx.Nonce(), nil
}

func (s *CelerBridgeProcessor) BuildTransaction(sendArgs *MultipathProcessorTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	tx, err := s.sendOrBuild(sendArgs, nil, lastUsedNonce)
	return tx, tx.Nonce(), err
}

func (s *CelerBridgeProcessor) BuildTransactionV2(sendArgs *wallettypes.SendTxArgs, lastUsedNonce int64) (*ethTypes.Transaction, uint64, error) {
	tx, err := s.sendOrBuildV2(sendArgs, nil, lastUsedNonce)
	if err != nil {
		return nil, 0, createBridgeCellerErrorResponse(err)
	}
	return tx, tx.Nonce(), err
}

func (s *CelerBridgeProcessor) CalculateAmountOut(params ProcessorInputParams) (*big.Int, error) {
	amt, err := s.estimateAmt(params.FromChain, params.ToChain, params.AmountIn, params.FromToken.Symbol)
	if err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}
	if amt.Err != nil {
		return nil, createBridgeCellerErrorResponse(err)
	}
	amountOut, _ := new(big.Int).SetString(amt.EqValueTokenAmt, 10)
	return amountOut, nil
}
