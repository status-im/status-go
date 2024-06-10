package bridge

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
	"github.com/status-im/status-go/services/wallet/bridge/cbridge"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/transactions"
)

const (
	baseURL     = "https://cbridge-prod2.celer.app"
	testBaseURL = "https://cbridge-v2-test.celer.network"

	maxSlippage = uint32(1000)
)

type CBridgeTxArgs struct {
	transactions.SendTxArgs
	ChainID   uint64         `json:"chainId"`
	Symbol    string         `json:"symbol"`
	Recipient common.Address `json:"recipient"`
	Amount    *hexutil.Big   `json:"amount"`
}

type CBridge struct {
	rpcClient          *rpc.Client
	httpClient         *thirdparty.HTTPClient
	transactor         transactions.TransactorIface
	tokenManager       *token.Manager
	prodTransferConfig *cbridge.GetTransferConfigsResponse
	testTransferConfig *cbridge.GetTransferConfigsResponse
}

func NewCbridge(rpcClient *rpc.Client, transactor transactions.TransactorIface, tokenManager *token.Manager) *CBridge {
	return &CBridge{
		rpcClient:    rpcClient,
		httpClient:   thirdparty.NewHTTPClient(),
		transactor:   transactor,
		tokenManager: tokenManager,
	}
}

func (s *CBridge) Name() string {
	return CBridgeName
}

func (s *CBridge) estimateAmt(from, to *params.Network, amountIn *big.Int, symbol string) (*cbridge.EstimateAmtResponse, error) {
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
	response, err := s.httpClient.DoGetRequest(context.Background(), url, params)
	if err != nil {
		return nil, err
	}

	var res cbridge.EstimateAmtResponse
	err = json.Unmarshal(response, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (s *CBridge) getTransferConfig(isTest bool) (*cbridge.GetTransferConfigsResponse, error) {
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
	response, err := s.httpClient.DoGetRequest(context.Background(), url, nil)
	if err != nil {
		return nil, err
	}

	var res cbridge.GetTransferConfigsResponse
	err = json.Unmarshal(response, &res)
	if err != nil {
		return nil, err
	}
	if isTest {
		s.testTransferConfig = &res
	} else {
		s.prodTransferConfig = &res
	}
	return &res, nil
}

func (s *CBridge) AvailableFor(params BridgeParams) (bool, error) {
	if params.FromChain.ChainID == params.ToChain.ChainID || params.ToToken != nil {
		return false, nil
	}

	transferConfig, err := s.getTransferConfig(params.FromChain.IsTest)
	if err != nil {
		return false, err
	}
	if transferConfig.Err != nil {
		return false, errors.New(transferConfig.Err.Msg)
	}

	var fromAvailable *cbridge.Chain
	var toAvailable *cbridge.Chain
	for _, chain := range transferConfig.Chains {
		if uint64(chain.GetId()) == params.FromChain.ChainID && chain.GasTokenSymbol == EthSymbol {
			fromAvailable = chain
		}

		if uint64(chain.GetId()) == params.ToChain.ChainID && chain.GasTokenSymbol == EthSymbol {
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

func (s *CBridge) CalculateFees(params BridgeParams) (*big.Int, *big.Int, error) {
	amt, err := s.estimateAmt(params.FromChain, params.ToChain, params.AmountIn, params.FromToken.Symbol)
	if err != nil {
		return nil, nil, err
	}
	baseFee, ok := new(big.Int).SetString(amt.BaseFee, 10)
	if !ok {
		return nil, nil, errors.New("failed to parse base fee")
	}
	percFee, ok := new(big.Int).SetString(amt.PercFee, 10)
	if !ok {
		return nil, nil, errors.New("failed to parse percentage fee")
	}

	return big.NewInt(0), new(big.Int).Add(baseFee, percFee), nil
}

func (c *CBridge) PackTxInputData(params BridgeParams) ([]byte, error) {
	abi, err := abi.JSON(strings.NewReader(celer.CelerABI))
	if err != nil {
		return []byte{}, err
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

func (s *CBridge) EstimateGas(params BridgeParams) (uint64, error) {
	value := new(big.Int)

	input, err := s.PackTxInputData(params)
	if err != nil {
		return 0, err
	}

	contractAddress, err := s.GetContractAddress(params)
	if err != nil {
		return 0, err
	}

	ethClient, err := s.rpcClient.EthClient(params.FromChain.ChainID)
	if err != nil {
		return 0, err
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
			return 0, err
		}
	}
	increasedEstimation := float64(estimation) * IncreaseEstimatedGasFactor
	return uint64(increasedEstimation), nil
}

func (s *CBridge) BuildTx(params BridgeParams) (*ethTypes.Transaction, error) {
	toAddr := types.Address(params.ToAddr)
	sendArgs := &TransactionBridge{
		CbridgeTx: &CBridgeTxArgs{
			SendTxArgs: transactions.SendTxArgs{
				From:  types.Address(params.FromAddr),
				To:    &toAddr,
				Value: (*hexutil.Big)(params.AmountIn),
				Data:  types.HexBytes("0x0"),
			},
			ChainID:   params.ToChain.ChainID,
			Symbol:    params.FromToken.Symbol,
			Recipient: params.ToAddr,
			Amount:    (*hexutil.Big)(params.AmountIn),
		},
		ChainID: params.FromChain.ChainID,
	}

	return s.BuildTransaction(sendArgs)
}

func (s *CBridge) GetContractAddress(params BridgeParams) (common.Address, error) {
	transferConfig, err := s.getTransferConfig(params.FromChain.IsTest)
	if err != nil {
		return common.Address{}, err
	}
	if transferConfig.Err != nil {
		return common.Address{}, errors.New(transferConfig.Err.Msg)
	}

	for _, chain := range transferConfig.Chains {
		if uint64(chain.Id) == params.FromChain.ChainID {
			return common.HexToAddress(chain.ContractAddr), nil
		}
	}

	return common.Address{}, errors.New("contract not found")
}

func (s *CBridge) sendOrBuild(sendArgs *TransactionBridge, signerFn bind.SignerFn) (*ethTypes.Transaction, error) {
	fromChain := s.rpcClient.NetworkManager.Find(sendArgs.ChainID)
	if fromChain == nil {
		return nil, errors.New("network not found")
	}
	token := s.tokenManager.FindToken(fromChain, sendArgs.CbridgeTx.Symbol)
	if token == nil {
		return nil, errors.New("token not found")
	}
	addrs, err := s.GetContractAddress(BridgeParams{
		FromChain: fromChain,
	})
	if err != nil {
		return nil, err
	}

	backend, err := s.rpcClient.EthClient(sendArgs.ChainID)
	if err != nil {
		return nil, err
	}
	contract, err := celer.NewCeler(addrs, backend)
	if err != nil {
		return nil, err
	}

	txOpts := sendArgs.CbridgeTx.ToTransactOpts(signerFn)
	if token.IsNative() {
		return contract.SendNative(
			txOpts,
			sendArgs.CbridgeTx.Recipient,
			(*big.Int)(sendArgs.CbridgeTx.Amount),
			sendArgs.CbridgeTx.ChainID,
			uint64(time.Now().UnixMilli()),
			maxSlippage,
		)
	}

	return contract.Send(
		txOpts,
		sendArgs.CbridgeTx.Recipient,
		token.Address,
		(*big.Int)(sendArgs.CbridgeTx.Amount),
		sendArgs.CbridgeTx.ChainID,
		uint64(time.Now().UnixMilli()),
		maxSlippage,
	)
}

func (s *CBridge) Send(sendArgs *TransactionBridge, verifiedAccount *account.SelectedExtKey) (types.Hash, error) {
	tx, err := s.sendOrBuild(sendArgs, getSigner(sendArgs.ChainID, sendArgs.CbridgeTx.From, verifiedAccount))
	if err != nil {
		return types.HexToHash(""), err
	}

	return types.Hash(tx.Hash()), nil
}

func (s *CBridge) BuildTransaction(sendArgs *TransactionBridge) (*ethTypes.Transaction, error) {
	return s.sendOrBuild(sendArgs, nil)
}

func (s *CBridge) CalculateAmountOut(params BridgeParams) (*big.Int, error) {
	amt, err := s.estimateAmt(params.FromChain, params.ToChain, params.AmountIn, params.FromToken.Symbol)
	if err != nil {
		return nil, err
	}
	if amt.Err != nil {
		return nil, err
	}
	amountOut, _ := new(big.Int).SetString(amt.EqValueTokenAmt, 10)
	return amountOut, nil
}
