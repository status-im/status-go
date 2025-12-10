// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package hopL2ArbitrumBridge

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// BridgeTransferRoot is an auto generated low-level Go binding around an user-defined struct.
type BridgeTransferRoot struct {
	Total           *big.Int
	AmountWithdrawn *big.Int
	CreatedAt       *big.Int
}

// HopL2ArbitrumBridgeMetaData contains all meta data concerning the HopL2ArbitrumBridge contract.
var HopL2ArbitrumBridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractIArbSys\",\"name\":\"_messenger\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1Governance\",\"type\":\"address\"},{\"internalType\":\"contractHopBridgeToken\",\"name\":\"hToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1BridgeAddress\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"activeChainIds\",\"type\":\"uint256[]\"},{\"internalType\":\"address[]\",\"name\":\"bonders\",\"type\":\"address[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newBonder\",\"type\":\"address\"}],\"name\":\"BonderAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousBonder\",\"type\":\"address\"}],\"name\":\"BonderRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalBondsSettled\",\"type\":\"uint256\"}],\"name\":\"MultipleWithdrawalsSettled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Stake\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"relayer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"relayerFee\",\"type\":\"uint256\"}],\"name\":\"TransferFromL1Completed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"TransferRootSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"TransferSent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"rootCommittedAt\",\"type\":\"uint256\"}],\"name\":\"TransfersCommitted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Unstake\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"}],\"name\":\"WithdrawalBondSettled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"WithdrawalBonded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"}],\"name\":\"Withdrew\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"activeChainIds\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"chainIds\",\"type\":\"uint256[]\"}],\"name\":\"addActiveChainIds\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"addBonder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ammWrapper\",\"outputs\":[{\"internalType\":\"contractI_L2_AmmWrapper\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"}],\"name\":\"bondWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"bondWithdrawalAndDistribute\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"}],\"name\":\"commitTransfers\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"relayer\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"relayerFee\",\"type\":\"uint256\"}],\"name\":\"distribute\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"}],\"name\":\"getBondedWithdrawalAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getCredit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getDebitAndAdditionalDebit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"maybeBonder\",\"type\":\"address\"}],\"name\":\"getIsBonder\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNextTransferNonce\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getRawDebit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"getTransferId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"getTransferRoot\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountWithdrawn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdAt\",\"type\":\"uint256\"}],\"internalType\":\"structBridge.TransferRoot\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"getTransferRootId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"hToken\",\"outputs\":[{\"internalType\":\"contractHopBridgeToken\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"}],\"name\":\"isTransferIdSpent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BridgeAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BridgeCaller\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1Governance\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"lastCommitTimeForChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxPendingTransfers\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractIArbSys\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minBonderBps\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minBonderFeeAbsolute\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minimumForceCommitDelay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"pendingAmountForChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"pendingTransferIdsForChainId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"chainIds\",\"type\":\"uint256[]\"}],\"name\":\"removeActiveChainIds\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"removeBonder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"originalAmount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"rescueTransferRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"send\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractI_L2_AmmWrapper\",\"name\":\"_ammWrapper\",\"type\":\"address\"}],\"name\":\"setAmmWrapper\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"setHopBridgeTokenOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1BridgeAddress\",\"type\":\"address\"}],\"name\":\"setL1BridgeAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1BridgeCaller\",\"type\":\"address\"}],\"name\":\"setL1BridgeCaller\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Governance\",\"type\":\"address\"}],\"name\":\"setL1Governance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxPendingTransfers\",\"type\":\"uint256\"}],\"name\":\"setMaxPendingTransfers\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractIArbSys\",\"name\":\"_messenger\",\"type\":\"address\"}],\"name\":\"setMessenger\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minBonderBps\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_minBonderFeeAbsolute\",\"type\":\"uint256\"}],\"name\":\"setMinimumBonderFeeRequirements\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minimumForceCommitDelay\",\"type\":\"uint256\"}],\"name\":\"setMinimumForceCommitDelay\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"setTransferRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"transferRootTotalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"transferIdTreeIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalLeaves\",\"type\":\"uint256\"}],\"name\":\"settleBondedWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32[]\",\"name\":\"transferIds\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"settleBondedWithdrawals\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"stake\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"transferNonceIncrementer\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"unstake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"transferRootTotalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"transferIdTreeIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalLeaves\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// HopL2ArbitrumBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use HopL2ArbitrumBridgeMetaData.ABI instead.
var HopL2ArbitrumBridgeABI = HopL2ArbitrumBridgeMetaData.ABI

// HopL2ArbitrumBridge is an auto generated Go binding around an Ethereum contract.
type HopL2ArbitrumBridge struct {
	HopL2ArbitrumBridgeCaller     // Read-only binding to the contract
	HopL2ArbitrumBridgeTransactor // Write-only binding to the contract
	HopL2ArbitrumBridgeFilterer   // Log filterer for contract events
}

// HopL2ArbitrumBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type HopL2ArbitrumBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2ArbitrumBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HopL2ArbitrumBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2ArbitrumBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HopL2ArbitrumBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2ArbitrumBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HopL2ArbitrumBridgeSession struct {
	Contract     *HopL2ArbitrumBridge // Generic contract binding to set the session for
	CallOpts     bind.CallOpts        // Call options to use throughout this session
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// HopL2ArbitrumBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HopL2ArbitrumBridgeCallerSession struct {
	Contract *HopL2ArbitrumBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts              // Call options to use throughout this session
}

// HopL2ArbitrumBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HopL2ArbitrumBridgeTransactorSession struct {
	Contract     *HopL2ArbitrumBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts              // Transaction auth options to use throughout this session
}

// HopL2ArbitrumBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type HopL2ArbitrumBridgeRaw struct {
	Contract *HopL2ArbitrumBridge // Generic contract binding to access the raw methods on
}

// HopL2ArbitrumBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HopL2ArbitrumBridgeCallerRaw struct {
	Contract *HopL2ArbitrumBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// HopL2ArbitrumBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HopL2ArbitrumBridgeTransactorRaw struct {
	Contract *HopL2ArbitrumBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHopL2ArbitrumBridge creates a new instance of HopL2ArbitrumBridge, bound to a specific deployed contract.
func NewHopL2ArbitrumBridge(address common.Address, backend bind.ContractBackend) (*HopL2ArbitrumBridge, error) {
	contract, err := bindHopL2ArbitrumBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridge{HopL2ArbitrumBridgeCaller: HopL2ArbitrumBridgeCaller{contract: contract}, HopL2ArbitrumBridgeTransactor: HopL2ArbitrumBridgeTransactor{contract: contract}, HopL2ArbitrumBridgeFilterer: HopL2ArbitrumBridgeFilterer{contract: contract}}, nil
}

// NewHopL2ArbitrumBridgeCaller creates a new read-only instance of HopL2ArbitrumBridge, bound to a specific deployed contract.
func NewHopL2ArbitrumBridgeCaller(address common.Address, caller bind.ContractCaller) (*HopL2ArbitrumBridgeCaller, error) {
	contract, err := bindHopL2ArbitrumBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeCaller{contract: contract}, nil
}

// NewHopL2ArbitrumBridgeTransactor creates a new write-only instance of HopL2ArbitrumBridge, bound to a specific deployed contract.
func NewHopL2ArbitrumBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*HopL2ArbitrumBridgeTransactor, error) {
	contract, err := bindHopL2ArbitrumBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeTransactor{contract: contract}, nil
}

// NewHopL2ArbitrumBridgeFilterer creates a new log filterer instance of HopL2ArbitrumBridge, bound to a specific deployed contract.
func NewHopL2ArbitrumBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*HopL2ArbitrumBridgeFilterer, error) {
	contract, err := bindHopL2ArbitrumBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeFilterer{contract: contract}, nil
}

// bindHopL2ArbitrumBridge binds a generic wrapper to an already deployed contract.
func bindHopL2ArbitrumBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HopL2ArbitrumBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL2ArbitrumBridge.Contract.HopL2ArbitrumBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.HopL2ArbitrumBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.HopL2ArbitrumBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL2ArbitrumBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.contract.Transact(opts, method, params...)
}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) ActiveChainIds(opts *bind.CallOpts, arg0 *big.Int) (bool, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "activeChainIds", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) ActiveChainIds(arg0 *big.Int) (bool, error) {
	return _HopL2ArbitrumBridge.Contract.ActiveChainIds(&_HopL2ArbitrumBridge.CallOpts, arg0)
}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) ActiveChainIds(arg0 *big.Int) (bool, error) {
	return _HopL2ArbitrumBridge.Contract.ActiveChainIds(&_HopL2ArbitrumBridge.CallOpts, arg0)
}

// AmmWrapper is a free data retrieval call binding the contract method 0xe9cdfe51.
//
// Solidity: function ammWrapper() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) AmmWrapper(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "ammWrapper")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AmmWrapper is a free data retrieval call binding the contract method 0xe9cdfe51.
//
// Solidity: function ammWrapper() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) AmmWrapper() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.AmmWrapper(&_HopL2ArbitrumBridge.CallOpts)
}

// AmmWrapper is a free data retrieval call binding the contract method 0xe9cdfe51.
//
// Solidity: function ammWrapper() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) AmmWrapper() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.AmmWrapper(&_HopL2ArbitrumBridge.CallOpts)
}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetBondedWithdrawalAmount(opts *bind.CallOpts, bonder common.Address, transferId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getBondedWithdrawalAmount", bonder, transferId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetBondedWithdrawalAmount(bonder common.Address, transferId [32]byte) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetBondedWithdrawalAmount(&_HopL2ArbitrumBridge.CallOpts, bonder, transferId)
}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetBondedWithdrawalAmount(bonder common.Address, transferId [32]byte) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetBondedWithdrawalAmount(&_HopL2ArbitrumBridge.CallOpts, bonder, transferId)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getChainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetChainId() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetChainId(&_HopL2ArbitrumBridge.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetChainId() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetChainId(&_HopL2ArbitrumBridge.CallOpts)
}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetCredit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getCredit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetCredit(bonder common.Address) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetCredit(&_HopL2ArbitrumBridge.CallOpts, bonder)
}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetCredit(bonder common.Address) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetCredit(&_HopL2ArbitrumBridge.CallOpts, bonder)
}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetDebitAndAdditionalDebit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getDebitAndAdditionalDebit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetDebitAndAdditionalDebit(bonder common.Address) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetDebitAndAdditionalDebit(&_HopL2ArbitrumBridge.CallOpts, bonder)
}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetDebitAndAdditionalDebit(bonder common.Address) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetDebitAndAdditionalDebit(&_HopL2ArbitrumBridge.CallOpts, bonder)
}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetIsBonder(opts *bind.CallOpts, maybeBonder common.Address) (bool, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getIsBonder", maybeBonder)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetIsBonder(maybeBonder common.Address) (bool, error) {
	return _HopL2ArbitrumBridge.Contract.GetIsBonder(&_HopL2ArbitrumBridge.CallOpts, maybeBonder)
}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetIsBonder(maybeBonder common.Address) (bool, error) {
	return _HopL2ArbitrumBridge.Contract.GetIsBonder(&_HopL2ArbitrumBridge.CallOpts, maybeBonder)
}

// GetNextTransferNonce is a free data retrieval call binding the contract method 0x051e7216.
//
// Solidity: function getNextTransferNonce() view returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetNextTransferNonce(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getNextTransferNonce")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetNextTransferNonce is a free data retrieval call binding the contract method 0x051e7216.
//
// Solidity: function getNextTransferNonce() view returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetNextTransferNonce() ([32]byte, error) {
	return _HopL2ArbitrumBridge.Contract.GetNextTransferNonce(&_HopL2ArbitrumBridge.CallOpts)
}

// GetNextTransferNonce is a free data retrieval call binding the contract method 0x051e7216.
//
// Solidity: function getNextTransferNonce() view returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetNextTransferNonce() ([32]byte, error) {
	return _HopL2ArbitrumBridge.Contract.GetNextTransferNonce(&_HopL2ArbitrumBridge.CallOpts)
}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetRawDebit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getRawDebit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetRawDebit(bonder common.Address) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetRawDebit(&_HopL2ArbitrumBridge.CallOpts, bonder)
}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetRawDebit(bonder common.Address) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.GetRawDebit(&_HopL2ArbitrumBridge.CallOpts, bonder)
}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetTransferId(opts *bind.CallOpts, chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getTransferId", chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetTransferId(chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	return _HopL2ArbitrumBridge.Contract.GetTransferId(&_HopL2ArbitrumBridge.CallOpts, chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetTransferId(chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	return _HopL2ArbitrumBridge.Contract.GetTransferId(&_HopL2ArbitrumBridge.CallOpts, chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetTransferRoot(opts *bind.CallOpts, rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getTransferRoot", rootHash, totalAmount)

	if err != nil {
		return *new(BridgeTransferRoot), err
	}

	out0 := *abi.ConvertType(out[0], new(BridgeTransferRoot)).(*BridgeTransferRoot)

	return out0, err

}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	return _HopL2ArbitrumBridge.Contract.GetTransferRoot(&_HopL2ArbitrumBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	return _HopL2ArbitrumBridge.Contract.GetTransferRoot(&_HopL2ArbitrumBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) GetTransferRootId(opts *bind.CallOpts, rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "getTransferRootId", rootHash, totalAmount)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) GetTransferRootId(rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	return _HopL2ArbitrumBridge.Contract.GetTransferRootId(&_HopL2ArbitrumBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) GetTransferRootId(rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	return _HopL2ArbitrumBridge.Contract.GetTransferRootId(&_HopL2ArbitrumBridge.CallOpts, rootHash, totalAmount)
}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) HToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "hToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) HToken() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.HToken(&_HopL2ArbitrumBridge.CallOpts)
}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) HToken() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.HToken(&_HopL2ArbitrumBridge.CallOpts)
}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) IsTransferIdSpent(opts *bind.CallOpts, transferId [32]byte) (bool, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "isTransferIdSpent", transferId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) IsTransferIdSpent(transferId [32]byte) (bool, error) {
	return _HopL2ArbitrumBridge.Contract.IsTransferIdSpent(&_HopL2ArbitrumBridge.CallOpts, transferId)
}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) IsTransferIdSpent(transferId [32]byte) (bool, error) {
	return _HopL2ArbitrumBridge.Contract.IsTransferIdSpent(&_HopL2ArbitrumBridge.CallOpts, transferId)
}

// L1BridgeAddress is a free data retrieval call binding the contract method 0x5ab2a558.
//
// Solidity: function l1BridgeAddress() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) L1BridgeAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "l1BridgeAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1BridgeAddress is a free data retrieval call binding the contract method 0x5ab2a558.
//
// Solidity: function l1BridgeAddress() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) L1BridgeAddress() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.L1BridgeAddress(&_HopL2ArbitrumBridge.CallOpts)
}

// L1BridgeAddress is a free data retrieval call binding the contract method 0x5ab2a558.
//
// Solidity: function l1BridgeAddress() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) L1BridgeAddress() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.L1BridgeAddress(&_HopL2ArbitrumBridge.CallOpts)
}

// L1BridgeCaller is a free data retrieval call binding the contract method 0xd2442783.
//
// Solidity: function l1BridgeCaller() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) L1BridgeCaller(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "l1BridgeCaller")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1BridgeCaller is a free data retrieval call binding the contract method 0xd2442783.
//
// Solidity: function l1BridgeCaller() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) L1BridgeCaller() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.L1BridgeCaller(&_HopL2ArbitrumBridge.CallOpts)
}

// L1BridgeCaller is a free data retrieval call binding the contract method 0xd2442783.
//
// Solidity: function l1BridgeCaller() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) L1BridgeCaller() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.L1BridgeCaller(&_HopL2ArbitrumBridge.CallOpts)
}

// L1Governance is a free data retrieval call binding the contract method 0x3ef23f7f.
//
// Solidity: function l1Governance() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) L1Governance(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "l1Governance")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1Governance is a free data retrieval call binding the contract method 0x3ef23f7f.
//
// Solidity: function l1Governance() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) L1Governance() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.L1Governance(&_HopL2ArbitrumBridge.CallOpts)
}

// L1Governance is a free data retrieval call binding the contract method 0x3ef23f7f.
//
// Solidity: function l1Governance() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) L1Governance() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.L1Governance(&_HopL2ArbitrumBridge.CallOpts)
}

// LastCommitTimeForChainId is a free data retrieval call binding the contract method 0xd4e54c47.
//
// Solidity: function lastCommitTimeForChainId(uint256 ) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) LastCommitTimeForChainId(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "lastCommitTimeForChainId", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastCommitTimeForChainId is a free data retrieval call binding the contract method 0xd4e54c47.
//
// Solidity: function lastCommitTimeForChainId(uint256 ) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) LastCommitTimeForChainId(arg0 *big.Int) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.LastCommitTimeForChainId(&_HopL2ArbitrumBridge.CallOpts, arg0)
}

// LastCommitTimeForChainId is a free data retrieval call binding the contract method 0xd4e54c47.
//
// Solidity: function lastCommitTimeForChainId(uint256 ) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) LastCommitTimeForChainId(arg0 *big.Int) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.LastCommitTimeForChainId(&_HopL2ArbitrumBridge.CallOpts, arg0)
}

// MaxPendingTransfers is a free data retrieval call binding the contract method 0xbed93c84.
//
// Solidity: function maxPendingTransfers() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) MaxPendingTransfers(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "maxPendingTransfers")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxPendingTransfers is a free data retrieval call binding the contract method 0xbed93c84.
//
// Solidity: function maxPendingTransfers() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) MaxPendingTransfers() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.MaxPendingTransfers(&_HopL2ArbitrumBridge.CallOpts)
}

// MaxPendingTransfers is a free data retrieval call binding the contract method 0xbed93c84.
//
// Solidity: function maxPendingTransfers() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) MaxPendingTransfers() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.MaxPendingTransfers(&_HopL2ArbitrumBridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) Messenger() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.Messenger(&_HopL2ArbitrumBridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) Messenger() (common.Address, error) {
	return _HopL2ArbitrumBridge.Contract.Messenger(&_HopL2ArbitrumBridge.CallOpts)
}

// MinBonderBps is a free data retrieval call binding the contract method 0x35e2c4af.
//
// Solidity: function minBonderBps() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) MinBonderBps(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "minBonderBps")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinBonderBps is a free data retrieval call binding the contract method 0x35e2c4af.
//
// Solidity: function minBonderBps() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) MinBonderBps() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.MinBonderBps(&_HopL2ArbitrumBridge.CallOpts)
}

// MinBonderBps is a free data retrieval call binding the contract method 0x35e2c4af.
//
// Solidity: function minBonderBps() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) MinBonderBps() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.MinBonderBps(&_HopL2ArbitrumBridge.CallOpts)
}

// MinBonderFeeAbsolute is a free data retrieval call binding the contract method 0xc3035261.
//
// Solidity: function minBonderFeeAbsolute() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) MinBonderFeeAbsolute(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "minBonderFeeAbsolute")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinBonderFeeAbsolute is a free data retrieval call binding the contract method 0xc3035261.
//
// Solidity: function minBonderFeeAbsolute() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) MinBonderFeeAbsolute() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.MinBonderFeeAbsolute(&_HopL2ArbitrumBridge.CallOpts)
}

// MinBonderFeeAbsolute is a free data retrieval call binding the contract method 0xc3035261.
//
// Solidity: function minBonderFeeAbsolute() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) MinBonderFeeAbsolute() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.MinBonderFeeAbsolute(&_HopL2ArbitrumBridge.CallOpts)
}

// MinimumForceCommitDelay is a free data retrieval call binding the contract method 0x8f658198.
//
// Solidity: function minimumForceCommitDelay() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) MinimumForceCommitDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "minimumForceCommitDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinimumForceCommitDelay is a free data retrieval call binding the contract method 0x8f658198.
//
// Solidity: function minimumForceCommitDelay() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) MinimumForceCommitDelay() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.MinimumForceCommitDelay(&_HopL2ArbitrumBridge.CallOpts)
}

// MinimumForceCommitDelay is a free data retrieval call binding the contract method 0x8f658198.
//
// Solidity: function minimumForceCommitDelay() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) MinimumForceCommitDelay() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.MinimumForceCommitDelay(&_HopL2ArbitrumBridge.CallOpts)
}

// PendingAmountForChainId is a free data retrieval call binding the contract method 0x0f5e09e7.
//
// Solidity: function pendingAmountForChainId(uint256 ) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) PendingAmountForChainId(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "pendingAmountForChainId", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PendingAmountForChainId is a free data retrieval call binding the contract method 0x0f5e09e7.
//
// Solidity: function pendingAmountForChainId(uint256 ) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) PendingAmountForChainId(arg0 *big.Int) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.PendingAmountForChainId(&_HopL2ArbitrumBridge.CallOpts, arg0)
}

// PendingAmountForChainId is a free data retrieval call binding the contract method 0x0f5e09e7.
//
// Solidity: function pendingAmountForChainId(uint256 ) view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) PendingAmountForChainId(arg0 *big.Int) (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.PendingAmountForChainId(&_HopL2ArbitrumBridge.CallOpts, arg0)
}

// PendingTransferIdsForChainId is a free data retrieval call binding the contract method 0x98445caf.
//
// Solidity: function pendingTransferIdsForChainId(uint256 , uint256 ) view returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) PendingTransferIdsForChainId(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "pendingTransferIdsForChainId", arg0, arg1)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PendingTransferIdsForChainId is a free data retrieval call binding the contract method 0x98445caf.
//
// Solidity: function pendingTransferIdsForChainId(uint256 , uint256 ) view returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) PendingTransferIdsForChainId(arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	return _HopL2ArbitrumBridge.Contract.PendingTransferIdsForChainId(&_HopL2ArbitrumBridge.CallOpts, arg0, arg1)
}

// PendingTransferIdsForChainId is a free data retrieval call binding the contract method 0x98445caf.
//
// Solidity: function pendingTransferIdsForChainId(uint256 , uint256 ) view returns(bytes32)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) PendingTransferIdsForChainId(arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	return _HopL2ArbitrumBridge.Contract.PendingTransferIdsForChainId(&_HopL2ArbitrumBridge.CallOpts, arg0, arg1)
}

// TransferNonceIncrementer is a free data retrieval call binding the contract method 0x82c69f9d.
//
// Solidity: function transferNonceIncrementer() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCaller) TransferNonceIncrementer(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2ArbitrumBridge.contract.Call(opts, &out, "transferNonceIncrementer")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TransferNonceIncrementer is a free data retrieval call binding the contract method 0x82c69f9d.
//
// Solidity: function transferNonceIncrementer() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) TransferNonceIncrementer() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.TransferNonceIncrementer(&_HopL2ArbitrumBridge.CallOpts)
}

// TransferNonceIncrementer is a free data retrieval call binding the contract method 0x82c69f9d.
//
// Solidity: function transferNonceIncrementer() view returns(uint256)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeCallerSession) TransferNonceIncrementer() (*big.Int, error) {
	return _HopL2ArbitrumBridge.Contract.TransferNonceIncrementer(&_HopL2ArbitrumBridge.CallOpts)
}

// AddActiveChainIds is a paid mutator transaction binding the contract method 0xf8398fa4.
//
// Solidity: function addActiveChainIds(uint256[] chainIds) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) AddActiveChainIds(opts *bind.TransactOpts, chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "addActiveChainIds", chainIds)
}

// AddActiveChainIds is a paid mutator transaction binding the contract method 0xf8398fa4.
//
// Solidity: function addActiveChainIds(uint256[] chainIds) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) AddActiveChainIds(chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.AddActiveChainIds(&_HopL2ArbitrumBridge.TransactOpts, chainIds)
}

// AddActiveChainIds is a paid mutator transaction binding the contract method 0xf8398fa4.
//
// Solidity: function addActiveChainIds(uint256[] chainIds) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) AddActiveChainIds(chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.AddActiveChainIds(&_HopL2ArbitrumBridge.TransactOpts, chainIds)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) AddBonder(opts *bind.TransactOpts, bonder common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "addBonder", bonder)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) AddBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.AddBonder(&_HopL2ArbitrumBridge.TransactOpts, bonder)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) AddBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.AddBonder(&_HopL2ArbitrumBridge.TransactOpts, bonder)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) BondWithdrawal(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "bondWithdrawal", recipient, amount, transferNonce, bonderFee)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) BondWithdrawal(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.BondWithdrawal(&_HopL2ArbitrumBridge.TransactOpts, recipient, amount, transferNonce, bonderFee)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) BondWithdrawal(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.BondWithdrawal(&_HopL2ArbitrumBridge.TransactOpts, recipient, amount, transferNonce, bonderFee)
}

// BondWithdrawalAndDistribute is a paid mutator transaction binding the contract method 0x3d12a85a.
//
// Solidity: function bondWithdrawalAndDistribute(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) BondWithdrawalAndDistribute(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "bondWithdrawalAndDistribute", recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// BondWithdrawalAndDistribute is a paid mutator transaction binding the contract method 0x3d12a85a.
//
// Solidity: function bondWithdrawalAndDistribute(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) BondWithdrawalAndDistribute(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.BondWithdrawalAndDistribute(&_HopL2ArbitrumBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// BondWithdrawalAndDistribute is a paid mutator transaction binding the contract method 0x3d12a85a.
//
// Solidity: function bondWithdrawalAndDistribute(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) BondWithdrawalAndDistribute(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.BondWithdrawalAndDistribute(&_HopL2ArbitrumBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// CommitTransfers is a paid mutator transaction binding the contract method 0x32b949a2.
//
// Solidity: function commitTransfers(uint256 destinationChainId) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) CommitTransfers(opts *bind.TransactOpts, destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "commitTransfers", destinationChainId)
}

// CommitTransfers is a paid mutator transaction binding the contract method 0x32b949a2.
//
// Solidity: function commitTransfers(uint256 destinationChainId) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) CommitTransfers(destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.CommitTransfers(&_HopL2ArbitrumBridge.TransactOpts, destinationChainId)
}

// CommitTransfers is a paid mutator transaction binding the contract method 0x32b949a2.
//
// Solidity: function commitTransfers(uint256 destinationChainId) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) CommitTransfers(destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.CommitTransfers(&_HopL2ArbitrumBridge.TransactOpts, destinationChainId)
}

// Distribute is a paid mutator transaction binding the contract method 0xcc29a306.
//
// Solidity: function distribute(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) Distribute(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "distribute", recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// Distribute is a paid mutator transaction binding the contract method 0xcc29a306.
//
// Solidity: function distribute(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) Distribute(recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Distribute(&_HopL2ArbitrumBridge.TransactOpts, recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// Distribute is a paid mutator transaction binding the contract method 0xcc29a306.
//
// Solidity: function distribute(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) Distribute(recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Distribute(&_HopL2ArbitrumBridge.TransactOpts, recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// RemoveActiveChainIds is a paid mutator transaction binding the contract method 0x9f600a0b.
//
// Solidity: function removeActiveChainIds(uint256[] chainIds) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) RemoveActiveChainIds(opts *bind.TransactOpts, chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "removeActiveChainIds", chainIds)
}

// RemoveActiveChainIds is a paid mutator transaction binding the contract method 0x9f600a0b.
//
// Solidity: function removeActiveChainIds(uint256[] chainIds) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) RemoveActiveChainIds(chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.RemoveActiveChainIds(&_HopL2ArbitrumBridge.TransactOpts, chainIds)
}

// RemoveActiveChainIds is a paid mutator transaction binding the contract method 0x9f600a0b.
//
// Solidity: function removeActiveChainIds(uint256[] chainIds) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) RemoveActiveChainIds(chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.RemoveActiveChainIds(&_HopL2ArbitrumBridge.TransactOpts, chainIds)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) RemoveBonder(opts *bind.TransactOpts, bonder common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "removeBonder", bonder)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) RemoveBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.RemoveBonder(&_HopL2ArbitrumBridge.TransactOpts, bonder)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) RemoveBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.RemoveBonder(&_HopL2ArbitrumBridge.TransactOpts, bonder)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) RescueTransferRoot(opts *bind.TransactOpts, rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "rescueTransferRoot", rootHash, originalAmount, recipient)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) RescueTransferRoot(rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.RescueTransferRoot(&_HopL2ArbitrumBridge.TransactOpts, rootHash, originalAmount, recipient)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) RescueTransferRoot(rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.RescueTransferRoot(&_HopL2ArbitrumBridge.TransactOpts, rootHash, originalAmount, recipient)
}

// Send is a paid mutator transaction binding the contract method 0xa6bd1b33.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) Send(opts *bind.TransactOpts, chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "send", chainId, recipient, amount, bonderFee, amountOutMin, deadline)
}

// Send is a paid mutator transaction binding the contract method 0xa6bd1b33.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) Send(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Send(&_HopL2ArbitrumBridge.TransactOpts, chainId, recipient, amount, bonderFee, amountOutMin, deadline)
}

// Send is a paid mutator transaction binding the contract method 0xa6bd1b33.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) Send(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Send(&_HopL2ArbitrumBridge.TransactOpts, chainId, recipient, amount, bonderFee, amountOutMin, deadline)
}

// SetAmmWrapper is a paid mutator transaction binding the contract method 0x64c6fdb4.
//
// Solidity: function setAmmWrapper(address _ammWrapper) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetAmmWrapper(opts *bind.TransactOpts, _ammWrapper common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setAmmWrapper", _ammWrapper)
}

// SetAmmWrapper is a paid mutator transaction binding the contract method 0x64c6fdb4.
//
// Solidity: function setAmmWrapper(address _ammWrapper) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetAmmWrapper(_ammWrapper common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetAmmWrapper(&_HopL2ArbitrumBridge.TransactOpts, _ammWrapper)
}

// SetAmmWrapper is a paid mutator transaction binding the contract method 0x64c6fdb4.
//
// Solidity: function setAmmWrapper(address _ammWrapper) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetAmmWrapper(_ammWrapper common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetAmmWrapper(&_HopL2ArbitrumBridge.TransactOpts, _ammWrapper)
}

// SetHopBridgeTokenOwner is a paid mutator transaction binding the contract method 0x8295f258.
//
// Solidity: function setHopBridgeTokenOwner(address newOwner) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetHopBridgeTokenOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setHopBridgeTokenOwner", newOwner)
}

// SetHopBridgeTokenOwner is a paid mutator transaction binding the contract method 0x8295f258.
//
// Solidity: function setHopBridgeTokenOwner(address newOwner) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetHopBridgeTokenOwner(newOwner common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetHopBridgeTokenOwner(&_HopL2ArbitrumBridge.TransactOpts, newOwner)
}

// SetHopBridgeTokenOwner is a paid mutator transaction binding the contract method 0x8295f258.
//
// Solidity: function setHopBridgeTokenOwner(address newOwner) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetHopBridgeTokenOwner(newOwner common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetHopBridgeTokenOwner(&_HopL2ArbitrumBridge.TransactOpts, newOwner)
}

// SetL1BridgeAddress is a paid mutator transaction binding the contract method 0xe1825d06.
//
// Solidity: function setL1BridgeAddress(address _l1BridgeAddress) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetL1BridgeAddress(opts *bind.TransactOpts, _l1BridgeAddress common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setL1BridgeAddress", _l1BridgeAddress)
}

// SetL1BridgeAddress is a paid mutator transaction binding the contract method 0xe1825d06.
//
// Solidity: function setL1BridgeAddress(address _l1BridgeAddress) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetL1BridgeAddress(_l1BridgeAddress common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetL1BridgeAddress(&_HopL2ArbitrumBridge.TransactOpts, _l1BridgeAddress)
}

// SetL1BridgeAddress is a paid mutator transaction binding the contract method 0xe1825d06.
//
// Solidity: function setL1BridgeAddress(address _l1BridgeAddress) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetL1BridgeAddress(_l1BridgeAddress common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetL1BridgeAddress(&_HopL2ArbitrumBridge.TransactOpts, _l1BridgeAddress)
}

// SetL1BridgeCaller is a paid mutator transaction binding the contract method 0xaf33ae69.
//
// Solidity: function setL1BridgeCaller(address _l1BridgeCaller) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetL1BridgeCaller(opts *bind.TransactOpts, _l1BridgeCaller common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setL1BridgeCaller", _l1BridgeCaller)
}

// SetL1BridgeCaller is a paid mutator transaction binding the contract method 0xaf33ae69.
//
// Solidity: function setL1BridgeCaller(address _l1BridgeCaller) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetL1BridgeCaller(_l1BridgeCaller common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetL1BridgeCaller(&_HopL2ArbitrumBridge.TransactOpts, _l1BridgeCaller)
}

// SetL1BridgeCaller is a paid mutator transaction binding the contract method 0xaf33ae69.
//
// Solidity: function setL1BridgeCaller(address _l1BridgeCaller) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetL1BridgeCaller(_l1BridgeCaller common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetL1BridgeCaller(&_HopL2ArbitrumBridge.TransactOpts, _l1BridgeCaller)
}

// SetL1Governance is a paid mutator transaction binding the contract method 0xe40272d7.
//
// Solidity: function setL1Governance(address _l1Governance) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetL1Governance(opts *bind.TransactOpts, _l1Governance common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setL1Governance", _l1Governance)
}

// SetL1Governance is a paid mutator transaction binding the contract method 0xe40272d7.
//
// Solidity: function setL1Governance(address _l1Governance) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetL1Governance(_l1Governance common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetL1Governance(&_HopL2ArbitrumBridge.TransactOpts, _l1Governance)
}

// SetL1Governance is a paid mutator transaction binding the contract method 0xe40272d7.
//
// Solidity: function setL1Governance(address _l1Governance) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetL1Governance(_l1Governance common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetL1Governance(&_HopL2ArbitrumBridge.TransactOpts, _l1Governance)
}

// SetMaxPendingTransfers is a paid mutator transaction binding the contract method 0x4742bbfb.
//
// Solidity: function setMaxPendingTransfers(uint256 _maxPendingTransfers) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetMaxPendingTransfers(opts *bind.TransactOpts, _maxPendingTransfers *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setMaxPendingTransfers", _maxPendingTransfers)
}

// SetMaxPendingTransfers is a paid mutator transaction binding the contract method 0x4742bbfb.
//
// Solidity: function setMaxPendingTransfers(uint256 _maxPendingTransfers) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetMaxPendingTransfers(_maxPendingTransfers *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetMaxPendingTransfers(&_HopL2ArbitrumBridge.TransactOpts, _maxPendingTransfers)
}

// SetMaxPendingTransfers is a paid mutator transaction binding the contract method 0x4742bbfb.
//
// Solidity: function setMaxPendingTransfers(uint256 _maxPendingTransfers) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetMaxPendingTransfers(_maxPendingTransfers *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetMaxPendingTransfers(&_HopL2ArbitrumBridge.TransactOpts, _maxPendingTransfers)
}

// SetMessenger is a paid mutator transaction binding the contract method 0x66285967.
//
// Solidity: function setMessenger(address _messenger) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetMessenger(opts *bind.TransactOpts, _messenger common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setMessenger", _messenger)
}

// SetMessenger is a paid mutator transaction binding the contract method 0x66285967.
//
// Solidity: function setMessenger(address _messenger) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetMessenger(_messenger common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetMessenger(&_HopL2ArbitrumBridge.TransactOpts, _messenger)
}

// SetMessenger is a paid mutator transaction binding the contract method 0x66285967.
//
// Solidity: function setMessenger(address _messenger) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetMessenger(_messenger common.Address) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetMessenger(&_HopL2ArbitrumBridge.TransactOpts, _messenger)
}

// SetMinimumBonderFeeRequirements is a paid mutator transaction binding the contract method 0xa9fa4ed5.
//
// Solidity: function setMinimumBonderFeeRequirements(uint256 _minBonderBps, uint256 _minBonderFeeAbsolute) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetMinimumBonderFeeRequirements(opts *bind.TransactOpts, _minBonderBps *big.Int, _minBonderFeeAbsolute *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setMinimumBonderFeeRequirements", _minBonderBps, _minBonderFeeAbsolute)
}

// SetMinimumBonderFeeRequirements is a paid mutator transaction binding the contract method 0xa9fa4ed5.
//
// Solidity: function setMinimumBonderFeeRequirements(uint256 _minBonderBps, uint256 _minBonderFeeAbsolute) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetMinimumBonderFeeRequirements(_minBonderBps *big.Int, _minBonderFeeAbsolute *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetMinimumBonderFeeRequirements(&_HopL2ArbitrumBridge.TransactOpts, _minBonderBps, _minBonderFeeAbsolute)
}

// SetMinimumBonderFeeRequirements is a paid mutator transaction binding the contract method 0xa9fa4ed5.
//
// Solidity: function setMinimumBonderFeeRequirements(uint256 _minBonderBps, uint256 _minBonderFeeAbsolute) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetMinimumBonderFeeRequirements(_minBonderBps *big.Int, _minBonderFeeAbsolute *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetMinimumBonderFeeRequirements(&_HopL2ArbitrumBridge.TransactOpts, _minBonderBps, _minBonderFeeAbsolute)
}

// SetMinimumForceCommitDelay is a paid mutator transaction binding the contract method 0x9bf43028.
//
// Solidity: function setMinimumForceCommitDelay(uint256 _minimumForceCommitDelay) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetMinimumForceCommitDelay(opts *bind.TransactOpts, _minimumForceCommitDelay *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setMinimumForceCommitDelay", _minimumForceCommitDelay)
}

// SetMinimumForceCommitDelay is a paid mutator transaction binding the contract method 0x9bf43028.
//
// Solidity: function setMinimumForceCommitDelay(uint256 _minimumForceCommitDelay) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetMinimumForceCommitDelay(_minimumForceCommitDelay *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetMinimumForceCommitDelay(&_HopL2ArbitrumBridge.TransactOpts, _minimumForceCommitDelay)
}

// SetMinimumForceCommitDelay is a paid mutator transaction binding the contract method 0x9bf43028.
//
// Solidity: function setMinimumForceCommitDelay(uint256 _minimumForceCommitDelay) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetMinimumForceCommitDelay(_minimumForceCommitDelay *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetMinimumForceCommitDelay(&_HopL2ArbitrumBridge.TransactOpts, _minimumForceCommitDelay)
}

// SetTransferRoot is a paid mutator transaction binding the contract method 0xfd31c5ba.
//
// Solidity: function setTransferRoot(bytes32 rootHash, uint256 totalAmount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SetTransferRoot(opts *bind.TransactOpts, rootHash [32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "setTransferRoot", rootHash, totalAmount)
}

// SetTransferRoot is a paid mutator transaction binding the contract method 0xfd31c5ba.
//
// Solidity: function setTransferRoot(bytes32 rootHash, uint256 totalAmount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetTransferRoot(&_HopL2ArbitrumBridge.TransactOpts, rootHash, totalAmount)
}

// SetTransferRoot is a paid mutator transaction binding the contract method 0xfd31c5ba.
//
// Solidity: function setTransferRoot(bytes32 rootHash, uint256 totalAmount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SetTransferRoot(&_HopL2ArbitrumBridge.TransactOpts, rootHash, totalAmount)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SettleBondedWithdrawal(opts *bind.TransactOpts, bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "settleBondedWithdrawal", bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SettleBondedWithdrawal(bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SettleBondedWithdrawal(&_HopL2ArbitrumBridge.TransactOpts, bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SettleBondedWithdrawal(bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SettleBondedWithdrawal(&_HopL2ArbitrumBridge.TransactOpts, bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) SettleBondedWithdrawals(opts *bind.TransactOpts, bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "settleBondedWithdrawals", bonder, transferIds, totalAmount)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) SettleBondedWithdrawals(bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SettleBondedWithdrawals(&_HopL2ArbitrumBridge.TransactOpts, bonder, transferIds, totalAmount)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) SettleBondedWithdrawals(bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.SettleBondedWithdrawals(&_HopL2ArbitrumBridge.TransactOpts, bonder, transferIds, totalAmount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) Stake(opts *bind.TransactOpts, bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "stake", bonder, amount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) Stake(bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Stake(&_HopL2ArbitrumBridge.TransactOpts, bonder, amount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) Stake(bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Stake(&_HopL2ArbitrumBridge.TransactOpts, bonder, amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) Unstake(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "unstake", amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) Unstake(amount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Unstake(&_HopL2ArbitrumBridge.TransactOpts, amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) Unstake(amount *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Unstake(&_HopL2ArbitrumBridge.TransactOpts, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactor) Withdraw(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.contract.Transact(opts, "withdraw", recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeSession) Withdraw(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Withdraw(&_HopL2ArbitrumBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeTransactorSession) Withdraw(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2ArbitrumBridge.Contract.Withdraw(&_HopL2ArbitrumBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// HopL2ArbitrumBridgeBonderAddedIterator is returned from FilterBonderAdded and is used to iterate over the raw logs and unpacked data for BonderAdded events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeBonderAddedIterator struct {
	Event *HopL2ArbitrumBridgeBonderAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeBonderAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeBonderAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeBonderAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeBonderAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeBonderAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeBonderAdded represents a BonderAdded event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeBonderAdded struct {
	NewBonder common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterBonderAdded is a free log retrieval operation binding the contract event 0x2cec73b7434d3b91198ad1a618f63e6a0761ce281af5ec9ec76606d948d03e23.
//
// Solidity: event BonderAdded(address indexed newBonder)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterBonderAdded(opts *bind.FilterOpts, newBonder []common.Address) (*HopL2ArbitrumBridgeBonderAddedIterator, error) {

	var newBonderRule []interface{}
	for _, newBonderItem := range newBonder {
		newBonderRule = append(newBonderRule, newBonderItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "BonderAdded", newBonderRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeBonderAddedIterator{contract: _HopL2ArbitrumBridge.contract, event: "BonderAdded", logs: logs, sub: sub}, nil
}

// WatchBonderAdded is a free log subscription operation binding the contract event 0x2cec73b7434d3b91198ad1a618f63e6a0761ce281af5ec9ec76606d948d03e23.
//
// Solidity: event BonderAdded(address indexed newBonder)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchBonderAdded(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeBonderAdded, newBonder []common.Address) (event.Subscription, error) {

	var newBonderRule []interface{}
	for _, newBonderItem := range newBonder {
		newBonderRule = append(newBonderRule, newBonderItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "BonderAdded", newBonderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeBonderAdded)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "BonderAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBonderAdded is a log parse operation binding the contract event 0x2cec73b7434d3b91198ad1a618f63e6a0761ce281af5ec9ec76606d948d03e23.
//
// Solidity: event BonderAdded(address indexed newBonder)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseBonderAdded(log types.Log) (*HopL2ArbitrumBridgeBonderAdded, error) {
	event := new(HopL2ArbitrumBridgeBonderAdded)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "BonderAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeBonderRemovedIterator is returned from FilterBonderRemoved and is used to iterate over the raw logs and unpacked data for BonderRemoved events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeBonderRemovedIterator struct {
	Event *HopL2ArbitrumBridgeBonderRemoved // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeBonderRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeBonderRemoved)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeBonderRemoved)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeBonderRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeBonderRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeBonderRemoved represents a BonderRemoved event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeBonderRemoved struct {
	PreviousBonder common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterBonderRemoved is a free log retrieval operation binding the contract event 0x4234ba611d325b3ba434c4e1b037967b955b1274d4185ee9847b7491111a48ff.
//
// Solidity: event BonderRemoved(address indexed previousBonder)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterBonderRemoved(opts *bind.FilterOpts, previousBonder []common.Address) (*HopL2ArbitrumBridgeBonderRemovedIterator, error) {

	var previousBonderRule []interface{}
	for _, previousBonderItem := range previousBonder {
		previousBonderRule = append(previousBonderRule, previousBonderItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "BonderRemoved", previousBonderRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeBonderRemovedIterator{contract: _HopL2ArbitrumBridge.contract, event: "BonderRemoved", logs: logs, sub: sub}, nil
}

// WatchBonderRemoved is a free log subscription operation binding the contract event 0x4234ba611d325b3ba434c4e1b037967b955b1274d4185ee9847b7491111a48ff.
//
// Solidity: event BonderRemoved(address indexed previousBonder)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchBonderRemoved(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeBonderRemoved, previousBonder []common.Address) (event.Subscription, error) {

	var previousBonderRule []interface{}
	for _, previousBonderItem := range previousBonder {
		previousBonderRule = append(previousBonderRule, previousBonderItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "BonderRemoved", previousBonderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeBonderRemoved)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "BonderRemoved", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBonderRemoved is a log parse operation binding the contract event 0x4234ba611d325b3ba434c4e1b037967b955b1274d4185ee9847b7491111a48ff.
//
// Solidity: event BonderRemoved(address indexed previousBonder)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseBonderRemoved(log types.Log) (*HopL2ArbitrumBridgeBonderRemoved, error) {
	event := new(HopL2ArbitrumBridgeBonderRemoved)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "BonderRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeMultipleWithdrawalsSettledIterator is returned from FilterMultipleWithdrawalsSettled and is used to iterate over the raw logs and unpacked data for MultipleWithdrawalsSettled events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeMultipleWithdrawalsSettledIterator struct {
	Event *HopL2ArbitrumBridgeMultipleWithdrawalsSettled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeMultipleWithdrawalsSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeMultipleWithdrawalsSettled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeMultipleWithdrawalsSettled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeMultipleWithdrawalsSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeMultipleWithdrawalsSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeMultipleWithdrawalsSettled represents a MultipleWithdrawalsSettled event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeMultipleWithdrawalsSettled struct {
	Bonder            common.Address
	RootHash          [32]byte
	TotalBondsSettled *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterMultipleWithdrawalsSettled is a free log retrieval operation binding the contract event 0x78e830d08be9d5f957414c84d685c061ecbd8467be98b42ebb64f0118b57d2ff.
//
// Solidity: event MultipleWithdrawalsSettled(address indexed bonder, bytes32 indexed rootHash, uint256 totalBondsSettled)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterMultipleWithdrawalsSettled(opts *bind.FilterOpts, bonder []common.Address, rootHash [][32]byte) (*HopL2ArbitrumBridgeMultipleWithdrawalsSettledIterator, error) {

	var bonderRule []interface{}
	for _, bonderItem := range bonder {
		bonderRule = append(bonderRule, bonderItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "MultipleWithdrawalsSettled", bonderRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeMultipleWithdrawalsSettledIterator{contract: _HopL2ArbitrumBridge.contract, event: "MultipleWithdrawalsSettled", logs: logs, sub: sub}, nil
}

// WatchMultipleWithdrawalsSettled is a free log subscription operation binding the contract event 0x78e830d08be9d5f957414c84d685c061ecbd8467be98b42ebb64f0118b57d2ff.
//
// Solidity: event MultipleWithdrawalsSettled(address indexed bonder, bytes32 indexed rootHash, uint256 totalBondsSettled)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchMultipleWithdrawalsSettled(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeMultipleWithdrawalsSettled, bonder []common.Address, rootHash [][32]byte) (event.Subscription, error) {

	var bonderRule []interface{}
	for _, bonderItem := range bonder {
		bonderRule = append(bonderRule, bonderItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "MultipleWithdrawalsSettled", bonderRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeMultipleWithdrawalsSettled)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "MultipleWithdrawalsSettled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseMultipleWithdrawalsSettled is a log parse operation binding the contract event 0x78e830d08be9d5f957414c84d685c061ecbd8467be98b42ebb64f0118b57d2ff.
//
// Solidity: event MultipleWithdrawalsSettled(address indexed bonder, bytes32 indexed rootHash, uint256 totalBondsSettled)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseMultipleWithdrawalsSettled(log types.Log) (*HopL2ArbitrumBridgeMultipleWithdrawalsSettled, error) {
	event := new(HopL2ArbitrumBridgeMultipleWithdrawalsSettled)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "MultipleWithdrawalsSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeStakeIterator is returned from FilterStake and is used to iterate over the raw logs and unpacked data for Stake events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeStakeIterator struct {
	Event *HopL2ArbitrumBridgeStake // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeStakeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeStake)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeStake)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeStakeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeStakeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeStake represents a Stake event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeStake struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterStake is a free log retrieval operation binding the contract event 0xebedb8b3c678666e7f36970bc8f57abf6d8fa2e828c0da91ea5b75bf68ed101a.
//
// Solidity: event Stake(address indexed account, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterStake(opts *bind.FilterOpts, account []common.Address) (*HopL2ArbitrumBridgeStakeIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "Stake", accountRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeStakeIterator{contract: _HopL2ArbitrumBridge.contract, event: "Stake", logs: logs, sub: sub}, nil
}

// WatchStake is a free log subscription operation binding the contract event 0xebedb8b3c678666e7f36970bc8f57abf6d8fa2e828c0da91ea5b75bf68ed101a.
//
// Solidity: event Stake(address indexed account, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchStake(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeStake, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "Stake", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeStake)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "Stake", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseStake is a log parse operation binding the contract event 0xebedb8b3c678666e7f36970bc8f57abf6d8fa2e828c0da91ea5b75bf68ed101a.
//
// Solidity: event Stake(address indexed account, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseStake(log types.Log) (*HopL2ArbitrumBridgeStake, error) {
	event := new(HopL2ArbitrumBridgeStake)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "Stake", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeTransferFromL1CompletedIterator is returned from FilterTransferFromL1Completed and is used to iterate over the raw logs and unpacked data for TransferFromL1Completed events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeTransferFromL1CompletedIterator struct {
	Event *HopL2ArbitrumBridgeTransferFromL1Completed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeTransferFromL1CompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeTransferFromL1Completed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeTransferFromL1Completed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeTransferFromL1CompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeTransferFromL1CompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeTransferFromL1Completed represents a TransferFromL1Completed event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeTransferFromL1Completed struct {
	Recipient    common.Address
	Amount       *big.Int
	AmountOutMin *big.Int
	Deadline     *big.Int
	Relayer      common.Address
	RelayerFee   *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterTransferFromL1Completed is a free log retrieval operation binding the contract event 0x320958176930804eb66c2343c7343fc0367dc16249590c0f195783bee199d094.
//
// Solidity: event TransferFromL1Completed(address indexed recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address indexed relayer, uint256 relayerFee)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterTransferFromL1Completed(opts *bind.FilterOpts, recipient []common.Address, relayer []common.Address) (*HopL2ArbitrumBridgeTransferFromL1CompletedIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	var relayerRule []interface{}
	for _, relayerItem := range relayer {
		relayerRule = append(relayerRule, relayerItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "TransferFromL1Completed", recipientRule, relayerRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeTransferFromL1CompletedIterator{contract: _HopL2ArbitrumBridge.contract, event: "TransferFromL1Completed", logs: logs, sub: sub}, nil
}

// WatchTransferFromL1Completed is a free log subscription operation binding the contract event 0x320958176930804eb66c2343c7343fc0367dc16249590c0f195783bee199d094.
//
// Solidity: event TransferFromL1Completed(address indexed recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address indexed relayer, uint256 relayerFee)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchTransferFromL1Completed(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeTransferFromL1Completed, recipient []common.Address, relayer []common.Address) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	var relayerRule []interface{}
	for _, relayerItem := range relayer {
		relayerRule = append(relayerRule, relayerItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "TransferFromL1Completed", recipientRule, relayerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeTransferFromL1Completed)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "TransferFromL1Completed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransferFromL1Completed is a log parse operation binding the contract event 0x320958176930804eb66c2343c7343fc0367dc16249590c0f195783bee199d094.
//
// Solidity: event TransferFromL1Completed(address indexed recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address indexed relayer, uint256 relayerFee)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseTransferFromL1Completed(log types.Log) (*HopL2ArbitrumBridgeTransferFromL1Completed, error) {
	event := new(HopL2ArbitrumBridgeTransferFromL1Completed)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "TransferFromL1Completed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeTransferRootSetIterator is returned from FilterTransferRootSet and is used to iterate over the raw logs and unpacked data for TransferRootSet events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeTransferRootSetIterator struct {
	Event *HopL2ArbitrumBridgeTransferRootSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeTransferRootSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeTransferRootSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeTransferRootSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeTransferRootSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeTransferRootSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeTransferRootSet represents a TransferRootSet event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeTransferRootSet struct {
	RootHash    [32]byte
	TotalAmount *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTransferRootSet is a free log retrieval operation binding the contract event 0xb33d2162aead99dab59e77a7a67ea025b776bf8ca8079e132afdf9b23e03bd42.
//
// Solidity: event TransferRootSet(bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterTransferRootSet(opts *bind.FilterOpts, rootHash [][32]byte) (*HopL2ArbitrumBridgeTransferRootSetIterator, error) {

	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "TransferRootSet", rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeTransferRootSetIterator{contract: _HopL2ArbitrumBridge.contract, event: "TransferRootSet", logs: logs, sub: sub}, nil
}

// WatchTransferRootSet is a free log subscription operation binding the contract event 0xb33d2162aead99dab59e77a7a67ea025b776bf8ca8079e132afdf9b23e03bd42.
//
// Solidity: event TransferRootSet(bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchTransferRootSet(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeTransferRootSet, rootHash [][32]byte) (event.Subscription, error) {

	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "TransferRootSet", rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeTransferRootSet)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "TransferRootSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransferRootSet is a log parse operation binding the contract event 0xb33d2162aead99dab59e77a7a67ea025b776bf8ca8079e132afdf9b23e03bd42.
//
// Solidity: event TransferRootSet(bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseTransferRootSet(log types.Log) (*HopL2ArbitrumBridgeTransferRootSet, error) {
	event := new(HopL2ArbitrumBridgeTransferRootSet)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "TransferRootSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeTransferSentIterator is returned from FilterTransferSent and is used to iterate over the raw logs and unpacked data for TransferSent events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeTransferSentIterator struct {
	Event *HopL2ArbitrumBridgeTransferSent // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeTransferSentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeTransferSent)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeTransferSent)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeTransferSentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeTransferSentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeTransferSent represents a TransferSent event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeTransferSent struct {
	TransferId    [32]byte
	ChainId       *big.Int
	Recipient     common.Address
	Amount        *big.Int
	TransferNonce [32]byte
	BonderFee     *big.Int
	Index         *big.Int
	AmountOutMin  *big.Int
	Deadline      *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterTransferSent is a free log retrieval operation binding the contract event 0xe35dddd4ea75d7e9b3fe93af4f4e40e778c3da4074c9d93e7c6536f1e803c1eb.
//
// Solidity: event TransferSent(bytes32 indexed transferId, uint256 indexed chainId, address indexed recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 index, uint256 amountOutMin, uint256 deadline)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterTransferSent(opts *bind.FilterOpts, transferId [][32]byte, chainId []*big.Int, recipient []common.Address) (*HopL2ArbitrumBridgeTransferSentIterator, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var chainIdRule []interface{}
	for _, chainIdItem := range chainId {
		chainIdRule = append(chainIdRule, chainIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "TransferSent", transferIdRule, chainIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeTransferSentIterator{contract: _HopL2ArbitrumBridge.contract, event: "TransferSent", logs: logs, sub: sub}, nil
}

// WatchTransferSent is a free log subscription operation binding the contract event 0xe35dddd4ea75d7e9b3fe93af4f4e40e778c3da4074c9d93e7c6536f1e803c1eb.
//
// Solidity: event TransferSent(bytes32 indexed transferId, uint256 indexed chainId, address indexed recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 index, uint256 amountOutMin, uint256 deadline)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchTransferSent(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeTransferSent, transferId [][32]byte, chainId []*big.Int, recipient []common.Address) (event.Subscription, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var chainIdRule []interface{}
	for _, chainIdItem := range chainId {
		chainIdRule = append(chainIdRule, chainIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "TransferSent", transferIdRule, chainIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeTransferSent)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "TransferSent", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransferSent is a log parse operation binding the contract event 0xe35dddd4ea75d7e9b3fe93af4f4e40e778c3da4074c9d93e7c6536f1e803c1eb.
//
// Solidity: event TransferSent(bytes32 indexed transferId, uint256 indexed chainId, address indexed recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 index, uint256 amountOutMin, uint256 deadline)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseTransferSent(log types.Log) (*HopL2ArbitrumBridgeTransferSent, error) {
	event := new(HopL2ArbitrumBridgeTransferSent)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "TransferSent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeTransfersCommittedIterator is returned from FilterTransfersCommitted and is used to iterate over the raw logs and unpacked data for TransfersCommitted events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeTransfersCommittedIterator struct {
	Event *HopL2ArbitrumBridgeTransfersCommitted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeTransfersCommittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeTransfersCommitted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeTransfersCommitted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeTransfersCommittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeTransfersCommittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeTransfersCommitted represents a TransfersCommitted event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeTransfersCommitted struct {
	DestinationChainId *big.Int
	RootHash           [32]byte
	TotalAmount        *big.Int
	RootCommittedAt    *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterTransfersCommitted is a free log retrieval operation binding the contract event 0xf52ad20d3b4f50d1c40901dfb95a9ce5270b2fc32694e5c668354721cd87aa74.
//
// Solidity: event TransfersCommitted(uint256 indexed destinationChainId, bytes32 indexed rootHash, uint256 totalAmount, uint256 rootCommittedAt)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterTransfersCommitted(opts *bind.FilterOpts, destinationChainId []*big.Int, rootHash [][32]byte) (*HopL2ArbitrumBridgeTransfersCommittedIterator, error) {

	var destinationChainIdRule []interface{}
	for _, destinationChainIdItem := range destinationChainId {
		destinationChainIdRule = append(destinationChainIdRule, destinationChainIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "TransfersCommitted", destinationChainIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeTransfersCommittedIterator{contract: _HopL2ArbitrumBridge.contract, event: "TransfersCommitted", logs: logs, sub: sub}, nil
}

// WatchTransfersCommitted is a free log subscription operation binding the contract event 0xf52ad20d3b4f50d1c40901dfb95a9ce5270b2fc32694e5c668354721cd87aa74.
//
// Solidity: event TransfersCommitted(uint256 indexed destinationChainId, bytes32 indexed rootHash, uint256 totalAmount, uint256 rootCommittedAt)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchTransfersCommitted(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeTransfersCommitted, destinationChainId []*big.Int, rootHash [][32]byte) (event.Subscription, error) {

	var destinationChainIdRule []interface{}
	for _, destinationChainIdItem := range destinationChainId {
		destinationChainIdRule = append(destinationChainIdRule, destinationChainIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "TransfersCommitted", destinationChainIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeTransfersCommitted)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "TransfersCommitted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfersCommitted is a log parse operation binding the contract event 0xf52ad20d3b4f50d1c40901dfb95a9ce5270b2fc32694e5c668354721cd87aa74.
//
// Solidity: event TransfersCommitted(uint256 indexed destinationChainId, bytes32 indexed rootHash, uint256 totalAmount, uint256 rootCommittedAt)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseTransfersCommitted(log types.Log) (*HopL2ArbitrumBridgeTransfersCommitted, error) {
	event := new(HopL2ArbitrumBridgeTransfersCommitted)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "TransfersCommitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeUnstakeIterator is returned from FilterUnstake and is used to iterate over the raw logs and unpacked data for Unstake events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeUnstakeIterator struct {
	Event *HopL2ArbitrumBridgeUnstake // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeUnstakeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeUnstake)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeUnstake)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeUnstakeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeUnstakeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeUnstake represents a Unstake event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeUnstake struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnstake is a free log retrieval operation binding the contract event 0x85082129d87b2fe11527cb1b3b7a520aeb5aa6913f88a3d8757fe40d1db02fdd.
//
// Solidity: event Unstake(address indexed account, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterUnstake(opts *bind.FilterOpts, account []common.Address) (*HopL2ArbitrumBridgeUnstakeIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "Unstake", accountRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeUnstakeIterator{contract: _HopL2ArbitrumBridge.contract, event: "Unstake", logs: logs, sub: sub}, nil
}

// WatchUnstake is a free log subscription operation binding the contract event 0x85082129d87b2fe11527cb1b3b7a520aeb5aa6913f88a3d8757fe40d1db02fdd.
//
// Solidity: event Unstake(address indexed account, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchUnstake(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeUnstake, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "Unstake", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeUnstake)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "Unstake", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnstake is a log parse operation binding the contract event 0x85082129d87b2fe11527cb1b3b7a520aeb5aa6913f88a3d8757fe40d1db02fdd.
//
// Solidity: event Unstake(address indexed account, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseUnstake(log types.Log) (*HopL2ArbitrumBridgeUnstake, error) {
	event := new(HopL2ArbitrumBridgeUnstake)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "Unstake", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeWithdrawalBondSettledIterator is returned from FilterWithdrawalBondSettled and is used to iterate over the raw logs and unpacked data for WithdrawalBondSettled events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeWithdrawalBondSettledIterator struct {
	Event *HopL2ArbitrumBridgeWithdrawalBondSettled // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeWithdrawalBondSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeWithdrawalBondSettled)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeWithdrawalBondSettled)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeWithdrawalBondSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeWithdrawalBondSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeWithdrawalBondSettled represents a WithdrawalBondSettled event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeWithdrawalBondSettled struct {
	Bonder     common.Address
	TransferId [32]byte
	RootHash   [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalBondSettled is a free log retrieval operation binding the contract event 0x84eb21b24c31b27a3bc67dde4a598aad06db6e9415cd66544492b9616996143c.
//
// Solidity: event WithdrawalBondSettled(address indexed bonder, bytes32 indexed transferId, bytes32 indexed rootHash)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterWithdrawalBondSettled(opts *bind.FilterOpts, bonder []common.Address, transferId [][32]byte, rootHash [][32]byte) (*HopL2ArbitrumBridgeWithdrawalBondSettledIterator, error) {

	var bonderRule []interface{}
	for _, bonderItem := range bonder {
		bonderRule = append(bonderRule, bonderItem)
	}
	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "WithdrawalBondSettled", bonderRule, transferIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeWithdrawalBondSettledIterator{contract: _HopL2ArbitrumBridge.contract, event: "WithdrawalBondSettled", logs: logs, sub: sub}, nil
}

// WatchWithdrawalBondSettled is a free log subscription operation binding the contract event 0x84eb21b24c31b27a3bc67dde4a598aad06db6e9415cd66544492b9616996143c.
//
// Solidity: event WithdrawalBondSettled(address indexed bonder, bytes32 indexed transferId, bytes32 indexed rootHash)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchWithdrawalBondSettled(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeWithdrawalBondSettled, bonder []common.Address, transferId [][32]byte, rootHash [][32]byte) (event.Subscription, error) {

	var bonderRule []interface{}
	for _, bonderItem := range bonder {
		bonderRule = append(bonderRule, bonderItem)
	}
	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "WithdrawalBondSettled", bonderRule, transferIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeWithdrawalBondSettled)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "WithdrawalBondSettled", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrawalBondSettled is a log parse operation binding the contract event 0x84eb21b24c31b27a3bc67dde4a598aad06db6e9415cd66544492b9616996143c.
//
// Solidity: event WithdrawalBondSettled(address indexed bonder, bytes32 indexed transferId, bytes32 indexed rootHash)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseWithdrawalBondSettled(log types.Log) (*HopL2ArbitrumBridgeWithdrawalBondSettled, error) {
	event := new(HopL2ArbitrumBridgeWithdrawalBondSettled)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "WithdrawalBondSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeWithdrawalBondedIterator is returned from FilterWithdrawalBonded and is used to iterate over the raw logs and unpacked data for WithdrawalBonded events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeWithdrawalBondedIterator struct {
	Event *HopL2ArbitrumBridgeWithdrawalBonded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeWithdrawalBondedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeWithdrawalBonded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeWithdrawalBonded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeWithdrawalBondedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeWithdrawalBondedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeWithdrawalBonded represents a WithdrawalBonded event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeWithdrawalBonded struct {
	TransferId [32]byte
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalBonded is a free log retrieval operation binding the contract event 0x0c3d250c7831051e78aa6a56679e590374c7c424415ffe4aa474491def2fe705.
//
// Solidity: event WithdrawalBonded(bytes32 indexed transferId, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterWithdrawalBonded(opts *bind.FilterOpts, transferId [][32]byte) (*HopL2ArbitrumBridgeWithdrawalBondedIterator, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "WithdrawalBonded", transferIdRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeWithdrawalBondedIterator{contract: _HopL2ArbitrumBridge.contract, event: "WithdrawalBonded", logs: logs, sub: sub}, nil
}

// WatchWithdrawalBonded is a free log subscription operation binding the contract event 0x0c3d250c7831051e78aa6a56679e590374c7c424415ffe4aa474491def2fe705.
//
// Solidity: event WithdrawalBonded(bytes32 indexed transferId, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchWithdrawalBonded(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeWithdrawalBonded, transferId [][32]byte) (event.Subscription, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "WithdrawalBonded", transferIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeWithdrawalBonded)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "WithdrawalBonded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrawalBonded is a log parse operation binding the contract event 0x0c3d250c7831051e78aa6a56679e590374c7c424415ffe4aa474491def2fe705.
//
// Solidity: event WithdrawalBonded(bytes32 indexed transferId, uint256 amount)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseWithdrawalBonded(log types.Log) (*HopL2ArbitrumBridgeWithdrawalBonded, error) {
	event := new(HopL2ArbitrumBridgeWithdrawalBonded)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "WithdrawalBonded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2ArbitrumBridgeWithdrewIterator is returned from FilterWithdrew and is used to iterate over the raw logs and unpacked data for Withdrew events raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeWithdrewIterator struct {
	Event *HopL2ArbitrumBridgeWithdrew // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *HopL2ArbitrumBridgeWithdrewIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2ArbitrumBridgeWithdrew)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(HopL2ArbitrumBridgeWithdrew)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *HopL2ArbitrumBridgeWithdrewIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2ArbitrumBridgeWithdrewIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2ArbitrumBridgeWithdrew represents a Withdrew event raised by the HopL2ArbitrumBridge contract.
type HopL2ArbitrumBridgeWithdrew struct {
	TransferId    [32]byte
	Recipient     common.Address
	Amount        *big.Int
	TransferNonce [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterWithdrew is a free log retrieval operation binding the contract event 0x9475cdbde5fc71fe2ccd413c82878ee54d061b9f74f9e2e1a03ff1178821502c.
//
// Solidity: event Withdrew(bytes32 indexed transferId, address indexed recipient, uint256 amount, bytes32 transferNonce)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) FilterWithdrew(opts *bind.FilterOpts, transferId [][32]byte, recipient []common.Address) (*HopL2ArbitrumBridgeWithdrewIterator, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.FilterLogs(opts, "Withdrew", transferIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &HopL2ArbitrumBridgeWithdrewIterator{contract: _HopL2ArbitrumBridge.contract, event: "Withdrew", logs: logs, sub: sub}, nil
}

// WatchWithdrew is a free log subscription operation binding the contract event 0x9475cdbde5fc71fe2ccd413c82878ee54d061b9f74f9e2e1a03ff1178821502c.
//
// Solidity: event Withdrew(bytes32 indexed transferId, address indexed recipient, uint256 amount, bytes32 transferNonce)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) WatchWithdrew(opts *bind.WatchOpts, sink chan<- *HopL2ArbitrumBridgeWithdrew, transferId [][32]byte, recipient []common.Address) (event.Subscription, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL2ArbitrumBridge.contract.WatchLogs(opts, "Withdrew", transferIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2ArbitrumBridgeWithdrew)
				if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "Withdrew", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseWithdrew is a log parse operation binding the contract event 0x9475cdbde5fc71fe2ccd413c82878ee54d061b9f74f9e2e1a03ff1178821502c.
//
// Solidity: event Withdrew(bytes32 indexed transferId, address indexed recipient, uint256 amount, bytes32 transferNonce)
func (_HopL2ArbitrumBridge *HopL2ArbitrumBridgeFilterer) ParseWithdrew(log types.Log) (*HopL2ArbitrumBridgeWithdrew, error) {
	event := new(HopL2ArbitrumBridgeWithdrew)
	if err := _HopL2ArbitrumBridge.contract.UnpackLog(event, "Withdrew", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
