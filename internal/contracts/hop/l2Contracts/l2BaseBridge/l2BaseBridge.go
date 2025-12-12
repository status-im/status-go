// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package hopL2BaseBridge

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

// HopL2BaseBridgeMetaData contains all meta data concerning the HopL2BaseBridge contract.
var HopL2BaseBridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"contractiOVM_L2CrossDomainMessenger\",\"name\":\"_messenger\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1Governance\",\"type\":\"address\"},{\"internalType\":\"contractHopBridgeToken\",\"name\":\"hToken\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"l1BridgeAddress\",\"type\":\"address\"},{\"internalType\":\"uint256[]\",\"name\":\"activeChainIds\",\"type\":\"uint256[]\"},{\"internalType\":\"address[]\",\"name\":\"bonders\",\"type\":\"address[]\"},{\"internalType\":\"uint32\",\"name\":\"_defaultGasLimit\",\"type\":\"uint32\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newBonder\",\"type\":\"address\"}],\"name\":\"BonderAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousBonder\",\"type\":\"address\"}],\"name\":\"BonderRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalBondsSettled\",\"type\":\"uint256\"}],\"name\":\"MultipleWithdrawalsSettled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Stake\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"relayer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"relayerFee\",\"type\":\"uint256\"}],\"name\":\"TransferFromL1Completed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"TransferRootSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"TransferSent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"rootCommittedAt\",\"type\":\"uint256\"}],\"name\":\"TransfersCommitted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Unstake\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"}],\"name\":\"WithdrawalBondSettled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"WithdrawalBonded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"}],\"name\":\"Withdrew\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"activeChainIds\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"chainIds\",\"type\":\"uint256[]\"}],\"name\":\"addActiveChainIds\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"addBonder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ammWrapper\",\"outputs\":[{\"internalType\":\"contractI_L2_AmmWrapper\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"}],\"name\":\"bondWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"bondWithdrawalAndDistribute\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"}],\"name\":\"commitTransfers\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"defaultGasLimit\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"relayer\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"relayerFee\",\"type\":\"uint256\"}],\"name\":\"distribute\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"}],\"name\":\"getBondedWithdrawalAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getCredit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getDebitAndAdditionalDebit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"maybeBonder\",\"type\":\"address\"}],\"name\":\"getIsBonder\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getNextTransferNonce\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getRawDebit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"getTransferId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"getTransferRoot\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountWithdrawn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdAt\",\"type\":\"uint256\"}],\"internalType\":\"structBridge.TransferRoot\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"getTransferRootId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"hToken\",\"outputs\":[{\"internalType\":\"contractHopBridgeToken\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"}],\"name\":\"isTransferIdSpent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BridgeAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1BridgeCaller\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"l1Governance\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"lastCommitTimeForChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"maxPendingTransfers\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"messenger\",\"outputs\":[{\"internalType\":\"contractiOVM_L2CrossDomainMessenger\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minBonderBps\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minBonderFeeAbsolute\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minimumForceCommitDelay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"pendingAmountForChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"pendingTransferIdsForChainId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"chainIds\",\"type\":\"uint256[]\"}],\"name\":\"removeActiveChainIds\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"removeBonder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"originalAmount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"rescueTransferRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"send\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractI_L2_AmmWrapper\",\"name\":\"_ammWrapper\",\"type\":\"address\"}],\"name\":\"setAmmWrapper\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"_defaultGasLimit\",\"type\":\"uint32\"}],\"name\":\"setDefaultGasLimit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"setHopBridgeTokenOwner\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1BridgeAddress\",\"type\":\"address\"}],\"name\":\"setL1BridgeAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1BridgeCaller\",\"type\":\"address\"}],\"name\":\"setL1BridgeCaller\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_l1Governance\",\"type\":\"address\"}],\"name\":\"setL1Governance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_maxPendingTransfers\",\"type\":\"uint256\"}],\"name\":\"setMaxPendingTransfers\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"contractiOVM_L2CrossDomainMessenger\",\"name\":\"_messenger\",\"type\":\"address\"}],\"name\":\"setMessenger\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minBonderBps\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_minBonderFeeAbsolute\",\"type\":\"uint256\"}],\"name\":\"setMinimumBonderFeeRequirements\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minimumForceCommitDelay\",\"type\":\"uint256\"}],\"name\":\"setMinimumForceCommitDelay\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"setTransferRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"transferRootTotalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"transferIdTreeIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalLeaves\",\"type\":\"uint256\"}],\"name\":\"settleBondedWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32[]\",\"name\":\"transferIds\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"settleBondedWithdrawals\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"stake\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"transferNonceIncrementer\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"unstake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"transferRootTotalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"transferIdTreeIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalLeaves\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// HopL2BaseBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use HopL2BaseBridgeMetaData.ABI instead.
var HopL2BaseBridgeABI = HopL2BaseBridgeMetaData.ABI

// HopL2BaseBridge is an auto generated Go binding around an Ethereum contract.
type HopL2BaseBridge struct {
	HopL2BaseBridgeCaller     // Read-only binding to the contract
	HopL2BaseBridgeTransactor // Write-only binding to the contract
	HopL2BaseBridgeFilterer   // Log filterer for contract events
}

// HopL2BaseBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type HopL2BaseBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2BaseBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HopL2BaseBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2BaseBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HopL2BaseBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL2BaseBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HopL2BaseBridgeSession struct {
	Contract     *HopL2BaseBridge  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// HopL2BaseBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HopL2BaseBridgeCallerSession struct {
	Contract *HopL2BaseBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// HopL2BaseBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HopL2BaseBridgeTransactorSession struct {
	Contract     *HopL2BaseBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// HopL2BaseBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type HopL2BaseBridgeRaw struct {
	Contract *HopL2BaseBridge // Generic contract binding to access the raw methods on
}

// HopL2BaseBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HopL2BaseBridgeCallerRaw struct {
	Contract *HopL2BaseBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// HopL2BaseBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HopL2BaseBridgeTransactorRaw struct {
	Contract *HopL2BaseBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHopL2BaseBridge creates a new instance of HopL2BaseBridge, bound to a specific deployed contract.
func NewHopL2BaseBridge(address common.Address, backend bind.ContractBackend) (*HopL2BaseBridge, error) {
	contract, err := bindHopL2BaseBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridge{HopL2BaseBridgeCaller: HopL2BaseBridgeCaller{contract: contract}, HopL2BaseBridgeTransactor: HopL2BaseBridgeTransactor{contract: contract}, HopL2BaseBridgeFilterer: HopL2BaseBridgeFilterer{contract: contract}}, nil
}

// NewHopL2BaseBridgeCaller creates a new read-only instance of HopL2BaseBridge, bound to a specific deployed contract.
func NewHopL2BaseBridgeCaller(address common.Address, caller bind.ContractCaller) (*HopL2BaseBridgeCaller, error) {
	contract, err := bindHopL2BaseBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeCaller{contract: contract}, nil
}

// NewHopL2BaseBridgeTransactor creates a new write-only instance of HopL2BaseBridge, bound to a specific deployed contract.
func NewHopL2BaseBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*HopL2BaseBridgeTransactor, error) {
	contract, err := bindHopL2BaseBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeTransactor{contract: contract}, nil
}

// NewHopL2BaseBridgeFilterer creates a new log filterer instance of HopL2BaseBridge, bound to a specific deployed contract.
func NewHopL2BaseBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*HopL2BaseBridgeFilterer, error) {
	contract, err := bindHopL2BaseBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeFilterer{contract: contract}, nil
}

// bindHopL2BaseBridge binds a generic wrapper to an already deployed contract.
func bindHopL2BaseBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HopL2BaseBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL2BaseBridge *HopL2BaseBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL2BaseBridge.Contract.HopL2BaseBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL2BaseBridge *HopL2BaseBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.HopL2BaseBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL2BaseBridge *HopL2BaseBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.HopL2BaseBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL2BaseBridge *HopL2BaseBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL2BaseBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.contract.Transact(opts, method, params...)
}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) ActiveChainIds(opts *bind.CallOpts, arg0 *big.Int) (bool, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "activeChainIds", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) ActiveChainIds(arg0 *big.Int) (bool, error) {
	return _HopL2BaseBridge.Contract.ActiveChainIds(&_HopL2BaseBridge.CallOpts, arg0)
}

// ActiveChainIds is a free data retrieval call binding the contract method 0xc97d172e.
//
// Solidity: function activeChainIds(uint256 ) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) ActiveChainIds(arg0 *big.Int) (bool, error) {
	return _HopL2BaseBridge.Contract.ActiveChainIds(&_HopL2BaseBridge.CallOpts, arg0)
}

// AmmWrapper is a free data retrieval call binding the contract method 0xe9cdfe51.
//
// Solidity: function ammWrapper() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) AmmWrapper(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "ammWrapper")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AmmWrapper is a free data retrieval call binding the contract method 0xe9cdfe51.
//
// Solidity: function ammWrapper() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) AmmWrapper() (common.Address, error) {
	return _HopL2BaseBridge.Contract.AmmWrapper(&_HopL2BaseBridge.CallOpts)
}

// AmmWrapper is a free data retrieval call binding the contract method 0xe9cdfe51.
//
// Solidity: function ammWrapper() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) AmmWrapper() (common.Address, error) {
	return _HopL2BaseBridge.Contract.AmmWrapper(&_HopL2BaseBridge.CallOpts)
}

// DefaultGasLimit is a free data retrieval call binding the contract method 0x95368d2e.
//
// Solidity: function defaultGasLimit() view returns(uint32)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) DefaultGasLimit(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "defaultGasLimit")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// DefaultGasLimit is a free data retrieval call binding the contract method 0x95368d2e.
//
// Solidity: function defaultGasLimit() view returns(uint32)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) DefaultGasLimit() (uint32, error) {
	return _HopL2BaseBridge.Contract.DefaultGasLimit(&_HopL2BaseBridge.CallOpts)
}

// DefaultGasLimit is a free data retrieval call binding the contract method 0x95368d2e.
//
// Solidity: function defaultGasLimit() view returns(uint32)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) DefaultGasLimit() (uint32, error) {
	return _HopL2BaseBridge.Contract.DefaultGasLimit(&_HopL2BaseBridge.CallOpts)
}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetBondedWithdrawalAmount(opts *bind.CallOpts, bonder common.Address, transferId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getBondedWithdrawalAmount", bonder, transferId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetBondedWithdrawalAmount(bonder common.Address, transferId [32]byte) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetBondedWithdrawalAmount(&_HopL2BaseBridge.CallOpts, bonder, transferId)
}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetBondedWithdrawalAmount(bonder common.Address, transferId [32]byte) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetBondedWithdrawalAmount(&_HopL2BaseBridge.CallOpts, bonder, transferId)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getChainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetChainId() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetChainId(&_HopL2BaseBridge.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetChainId() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetChainId(&_HopL2BaseBridge.CallOpts)
}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetCredit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getCredit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetCredit(bonder common.Address) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetCredit(&_HopL2BaseBridge.CallOpts, bonder)
}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetCredit(bonder common.Address) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetCredit(&_HopL2BaseBridge.CallOpts, bonder)
}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetDebitAndAdditionalDebit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getDebitAndAdditionalDebit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetDebitAndAdditionalDebit(bonder common.Address) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetDebitAndAdditionalDebit(&_HopL2BaseBridge.CallOpts, bonder)
}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetDebitAndAdditionalDebit(bonder common.Address) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetDebitAndAdditionalDebit(&_HopL2BaseBridge.CallOpts, bonder)
}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetIsBonder(opts *bind.CallOpts, maybeBonder common.Address) (bool, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getIsBonder", maybeBonder)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetIsBonder(maybeBonder common.Address) (bool, error) {
	return _HopL2BaseBridge.Contract.GetIsBonder(&_HopL2BaseBridge.CallOpts, maybeBonder)
}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetIsBonder(maybeBonder common.Address) (bool, error) {
	return _HopL2BaseBridge.Contract.GetIsBonder(&_HopL2BaseBridge.CallOpts, maybeBonder)
}

// GetNextTransferNonce is a free data retrieval call binding the contract method 0x051e7216.
//
// Solidity: function getNextTransferNonce() view returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetNextTransferNonce(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getNextTransferNonce")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetNextTransferNonce is a free data retrieval call binding the contract method 0x051e7216.
//
// Solidity: function getNextTransferNonce() view returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetNextTransferNonce() ([32]byte, error) {
	return _HopL2BaseBridge.Contract.GetNextTransferNonce(&_HopL2BaseBridge.CallOpts)
}

// GetNextTransferNonce is a free data retrieval call binding the contract method 0x051e7216.
//
// Solidity: function getNextTransferNonce() view returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetNextTransferNonce() ([32]byte, error) {
	return _HopL2BaseBridge.Contract.GetNextTransferNonce(&_HopL2BaseBridge.CallOpts)
}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetRawDebit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getRawDebit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetRawDebit(bonder common.Address) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetRawDebit(&_HopL2BaseBridge.CallOpts, bonder)
}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetRawDebit(bonder common.Address) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.GetRawDebit(&_HopL2BaseBridge.CallOpts, bonder)
}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetTransferId(opts *bind.CallOpts, chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getTransferId", chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetTransferId(chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	return _HopL2BaseBridge.Contract.GetTransferId(&_HopL2BaseBridge.CallOpts, chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetTransferId(chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	return _HopL2BaseBridge.Contract.GetTransferId(&_HopL2BaseBridge.CallOpts, chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetTransferRoot(opts *bind.CallOpts, rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getTransferRoot", rootHash, totalAmount)

	if err != nil {
		return *new(BridgeTransferRoot), err
	}

	out0 := *abi.ConvertType(out[0], new(BridgeTransferRoot)).(*BridgeTransferRoot)

	return out0, err

}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	return _HopL2BaseBridge.Contract.GetTransferRoot(&_HopL2BaseBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	return _HopL2BaseBridge.Contract.GetTransferRoot(&_HopL2BaseBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) GetTransferRootId(opts *bind.CallOpts, rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "getTransferRootId", rootHash, totalAmount)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) GetTransferRootId(rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	return _HopL2BaseBridge.Contract.GetTransferRootId(&_HopL2BaseBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) GetTransferRootId(rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	return _HopL2BaseBridge.Contract.GetTransferRootId(&_HopL2BaseBridge.CallOpts, rootHash, totalAmount)
}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) HToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "hToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) HToken() (common.Address, error) {
	return _HopL2BaseBridge.Contract.HToken(&_HopL2BaseBridge.CallOpts)
}

// HToken is a free data retrieval call binding the contract method 0xfc6e3b3b.
//
// Solidity: function hToken() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) HToken() (common.Address, error) {
	return _HopL2BaseBridge.Contract.HToken(&_HopL2BaseBridge.CallOpts)
}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) IsTransferIdSpent(opts *bind.CallOpts, transferId [32]byte) (bool, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "isTransferIdSpent", transferId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) IsTransferIdSpent(transferId [32]byte) (bool, error) {
	return _HopL2BaseBridge.Contract.IsTransferIdSpent(&_HopL2BaseBridge.CallOpts, transferId)
}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) IsTransferIdSpent(transferId [32]byte) (bool, error) {
	return _HopL2BaseBridge.Contract.IsTransferIdSpent(&_HopL2BaseBridge.CallOpts, transferId)
}

// L1BridgeAddress is a free data retrieval call binding the contract method 0x5ab2a558.
//
// Solidity: function l1BridgeAddress() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) L1BridgeAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "l1BridgeAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1BridgeAddress is a free data retrieval call binding the contract method 0x5ab2a558.
//
// Solidity: function l1BridgeAddress() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) L1BridgeAddress() (common.Address, error) {
	return _HopL2BaseBridge.Contract.L1BridgeAddress(&_HopL2BaseBridge.CallOpts)
}

// L1BridgeAddress is a free data retrieval call binding the contract method 0x5ab2a558.
//
// Solidity: function l1BridgeAddress() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) L1BridgeAddress() (common.Address, error) {
	return _HopL2BaseBridge.Contract.L1BridgeAddress(&_HopL2BaseBridge.CallOpts)
}

// L1BridgeCaller is a free data retrieval call binding the contract method 0xd2442783.
//
// Solidity: function l1BridgeCaller() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) L1BridgeCaller(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "l1BridgeCaller")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1BridgeCaller is a free data retrieval call binding the contract method 0xd2442783.
//
// Solidity: function l1BridgeCaller() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) L1BridgeCaller() (common.Address, error) {
	return _HopL2BaseBridge.Contract.L1BridgeCaller(&_HopL2BaseBridge.CallOpts)
}

// L1BridgeCaller is a free data retrieval call binding the contract method 0xd2442783.
//
// Solidity: function l1BridgeCaller() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) L1BridgeCaller() (common.Address, error) {
	return _HopL2BaseBridge.Contract.L1BridgeCaller(&_HopL2BaseBridge.CallOpts)
}

// L1Governance is a free data retrieval call binding the contract method 0x3ef23f7f.
//
// Solidity: function l1Governance() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) L1Governance(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "l1Governance")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// L1Governance is a free data retrieval call binding the contract method 0x3ef23f7f.
//
// Solidity: function l1Governance() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) L1Governance() (common.Address, error) {
	return _HopL2BaseBridge.Contract.L1Governance(&_HopL2BaseBridge.CallOpts)
}

// L1Governance is a free data retrieval call binding the contract method 0x3ef23f7f.
//
// Solidity: function l1Governance() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) L1Governance() (common.Address, error) {
	return _HopL2BaseBridge.Contract.L1Governance(&_HopL2BaseBridge.CallOpts)
}

// LastCommitTimeForChainId is a free data retrieval call binding the contract method 0xd4e54c47.
//
// Solidity: function lastCommitTimeForChainId(uint256 ) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) LastCommitTimeForChainId(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "lastCommitTimeForChainId", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LastCommitTimeForChainId is a free data retrieval call binding the contract method 0xd4e54c47.
//
// Solidity: function lastCommitTimeForChainId(uint256 ) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) LastCommitTimeForChainId(arg0 *big.Int) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.LastCommitTimeForChainId(&_HopL2BaseBridge.CallOpts, arg0)
}

// LastCommitTimeForChainId is a free data retrieval call binding the contract method 0xd4e54c47.
//
// Solidity: function lastCommitTimeForChainId(uint256 ) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) LastCommitTimeForChainId(arg0 *big.Int) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.LastCommitTimeForChainId(&_HopL2BaseBridge.CallOpts, arg0)
}

// MaxPendingTransfers is a free data retrieval call binding the contract method 0xbed93c84.
//
// Solidity: function maxPendingTransfers() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) MaxPendingTransfers(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "maxPendingTransfers")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxPendingTransfers is a free data retrieval call binding the contract method 0xbed93c84.
//
// Solidity: function maxPendingTransfers() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) MaxPendingTransfers() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.MaxPendingTransfers(&_HopL2BaseBridge.CallOpts)
}

// MaxPendingTransfers is a free data retrieval call binding the contract method 0xbed93c84.
//
// Solidity: function maxPendingTransfers() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) MaxPendingTransfers() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.MaxPendingTransfers(&_HopL2BaseBridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) Messenger(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "messenger")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) Messenger() (common.Address, error) {
	return _HopL2BaseBridge.Contract.Messenger(&_HopL2BaseBridge.CallOpts)
}

// Messenger is a free data retrieval call binding the contract method 0x3cb747bf.
//
// Solidity: function messenger() view returns(address)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) Messenger() (common.Address, error) {
	return _HopL2BaseBridge.Contract.Messenger(&_HopL2BaseBridge.CallOpts)
}

// MinBonderBps is a free data retrieval call binding the contract method 0x35e2c4af.
//
// Solidity: function minBonderBps() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) MinBonderBps(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "minBonderBps")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinBonderBps is a free data retrieval call binding the contract method 0x35e2c4af.
//
// Solidity: function minBonderBps() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) MinBonderBps() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.MinBonderBps(&_HopL2BaseBridge.CallOpts)
}

// MinBonderBps is a free data retrieval call binding the contract method 0x35e2c4af.
//
// Solidity: function minBonderBps() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) MinBonderBps() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.MinBonderBps(&_HopL2BaseBridge.CallOpts)
}

// MinBonderFeeAbsolute is a free data retrieval call binding the contract method 0xc3035261.
//
// Solidity: function minBonderFeeAbsolute() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) MinBonderFeeAbsolute(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "minBonderFeeAbsolute")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinBonderFeeAbsolute is a free data retrieval call binding the contract method 0xc3035261.
//
// Solidity: function minBonderFeeAbsolute() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) MinBonderFeeAbsolute() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.MinBonderFeeAbsolute(&_HopL2BaseBridge.CallOpts)
}

// MinBonderFeeAbsolute is a free data retrieval call binding the contract method 0xc3035261.
//
// Solidity: function minBonderFeeAbsolute() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) MinBonderFeeAbsolute() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.MinBonderFeeAbsolute(&_HopL2BaseBridge.CallOpts)
}

// MinimumForceCommitDelay is a free data retrieval call binding the contract method 0x8f658198.
//
// Solidity: function minimumForceCommitDelay() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) MinimumForceCommitDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "minimumForceCommitDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinimumForceCommitDelay is a free data retrieval call binding the contract method 0x8f658198.
//
// Solidity: function minimumForceCommitDelay() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) MinimumForceCommitDelay() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.MinimumForceCommitDelay(&_HopL2BaseBridge.CallOpts)
}

// MinimumForceCommitDelay is a free data retrieval call binding the contract method 0x8f658198.
//
// Solidity: function minimumForceCommitDelay() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) MinimumForceCommitDelay() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.MinimumForceCommitDelay(&_HopL2BaseBridge.CallOpts)
}

// PendingAmountForChainId is a free data retrieval call binding the contract method 0x0f5e09e7.
//
// Solidity: function pendingAmountForChainId(uint256 ) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) PendingAmountForChainId(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "pendingAmountForChainId", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PendingAmountForChainId is a free data retrieval call binding the contract method 0x0f5e09e7.
//
// Solidity: function pendingAmountForChainId(uint256 ) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) PendingAmountForChainId(arg0 *big.Int) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.PendingAmountForChainId(&_HopL2BaseBridge.CallOpts, arg0)
}

// PendingAmountForChainId is a free data retrieval call binding the contract method 0x0f5e09e7.
//
// Solidity: function pendingAmountForChainId(uint256 ) view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) PendingAmountForChainId(arg0 *big.Int) (*big.Int, error) {
	return _HopL2BaseBridge.Contract.PendingAmountForChainId(&_HopL2BaseBridge.CallOpts, arg0)
}

// PendingTransferIdsForChainId is a free data retrieval call binding the contract method 0x98445caf.
//
// Solidity: function pendingTransferIdsForChainId(uint256 , uint256 ) view returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) PendingTransferIdsForChainId(opts *bind.CallOpts, arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "pendingTransferIdsForChainId", arg0, arg1)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PendingTransferIdsForChainId is a free data retrieval call binding the contract method 0x98445caf.
//
// Solidity: function pendingTransferIdsForChainId(uint256 , uint256 ) view returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) PendingTransferIdsForChainId(arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	return _HopL2BaseBridge.Contract.PendingTransferIdsForChainId(&_HopL2BaseBridge.CallOpts, arg0, arg1)
}

// PendingTransferIdsForChainId is a free data retrieval call binding the contract method 0x98445caf.
//
// Solidity: function pendingTransferIdsForChainId(uint256 , uint256 ) view returns(bytes32)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) PendingTransferIdsForChainId(arg0 *big.Int, arg1 *big.Int) ([32]byte, error) {
	return _HopL2BaseBridge.Contract.PendingTransferIdsForChainId(&_HopL2BaseBridge.CallOpts, arg0, arg1)
}

// TransferNonceIncrementer is a free data retrieval call binding the contract method 0x82c69f9d.
//
// Solidity: function transferNonceIncrementer() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCaller) TransferNonceIncrementer(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL2BaseBridge.contract.Call(opts, &out, "transferNonceIncrementer")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TransferNonceIncrementer is a free data retrieval call binding the contract method 0x82c69f9d.
//
// Solidity: function transferNonceIncrementer() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeSession) TransferNonceIncrementer() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.TransferNonceIncrementer(&_HopL2BaseBridge.CallOpts)
}

// TransferNonceIncrementer is a free data retrieval call binding the contract method 0x82c69f9d.
//
// Solidity: function transferNonceIncrementer() view returns(uint256)
func (_HopL2BaseBridge *HopL2BaseBridgeCallerSession) TransferNonceIncrementer() (*big.Int, error) {
	return _HopL2BaseBridge.Contract.TransferNonceIncrementer(&_HopL2BaseBridge.CallOpts)
}

// AddActiveChainIds is a paid mutator transaction binding the contract method 0xf8398fa4.
//
// Solidity: function addActiveChainIds(uint256[] chainIds) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) AddActiveChainIds(opts *bind.TransactOpts, chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "addActiveChainIds", chainIds)
}

// AddActiveChainIds is a paid mutator transaction binding the contract method 0xf8398fa4.
//
// Solidity: function addActiveChainIds(uint256[] chainIds) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) AddActiveChainIds(chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.AddActiveChainIds(&_HopL2BaseBridge.TransactOpts, chainIds)
}

// AddActiveChainIds is a paid mutator transaction binding the contract method 0xf8398fa4.
//
// Solidity: function addActiveChainIds(uint256[] chainIds) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) AddActiveChainIds(chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.AddActiveChainIds(&_HopL2BaseBridge.TransactOpts, chainIds)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) AddBonder(opts *bind.TransactOpts, bonder common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "addBonder", bonder)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) AddBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.AddBonder(&_HopL2BaseBridge.TransactOpts, bonder)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) AddBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.AddBonder(&_HopL2BaseBridge.TransactOpts, bonder)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) BondWithdrawal(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "bondWithdrawal", recipient, amount, transferNonce, bonderFee)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) BondWithdrawal(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.BondWithdrawal(&_HopL2BaseBridge.TransactOpts, recipient, amount, transferNonce, bonderFee)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) BondWithdrawal(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.BondWithdrawal(&_HopL2BaseBridge.TransactOpts, recipient, amount, transferNonce, bonderFee)
}

// BondWithdrawalAndDistribute is a paid mutator transaction binding the contract method 0x3d12a85a.
//
// Solidity: function bondWithdrawalAndDistribute(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) BondWithdrawalAndDistribute(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "bondWithdrawalAndDistribute", recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// BondWithdrawalAndDistribute is a paid mutator transaction binding the contract method 0x3d12a85a.
//
// Solidity: function bondWithdrawalAndDistribute(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) BondWithdrawalAndDistribute(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.BondWithdrawalAndDistribute(&_HopL2BaseBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// BondWithdrawalAndDistribute is a paid mutator transaction binding the contract method 0x3d12a85a.
//
// Solidity: function bondWithdrawalAndDistribute(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) BondWithdrawalAndDistribute(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.BondWithdrawalAndDistribute(&_HopL2BaseBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// CommitTransfers is a paid mutator transaction binding the contract method 0x32b949a2.
//
// Solidity: function commitTransfers(uint256 destinationChainId) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) CommitTransfers(opts *bind.TransactOpts, destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "commitTransfers", destinationChainId)
}

// CommitTransfers is a paid mutator transaction binding the contract method 0x32b949a2.
//
// Solidity: function commitTransfers(uint256 destinationChainId) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) CommitTransfers(destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.CommitTransfers(&_HopL2BaseBridge.TransactOpts, destinationChainId)
}

// CommitTransfers is a paid mutator transaction binding the contract method 0x32b949a2.
//
// Solidity: function commitTransfers(uint256 destinationChainId) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) CommitTransfers(destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.CommitTransfers(&_HopL2BaseBridge.TransactOpts, destinationChainId)
}

// Distribute is a paid mutator transaction binding the contract method 0xcc29a306.
//
// Solidity: function distribute(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) Distribute(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "distribute", recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// Distribute is a paid mutator transaction binding the contract method 0xcc29a306.
//
// Solidity: function distribute(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) Distribute(recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Distribute(&_HopL2BaseBridge.TransactOpts, recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// Distribute is a paid mutator transaction binding the contract method 0xcc29a306.
//
// Solidity: function distribute(address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) Distribute(recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Distribute(&_HopL2BaseBridge.TransactOpts, recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// RemoveActiveChainIds is a paid mutator transaction binding the contract method 0x9f600a0b.
//
// Solidity: function removeActiveChainIds(uint256[] chainIds) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) RemoveActiveChainIds(opts *bind.TransactOpts, chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "removeActiveChainIds", chainIds)
}

// RemoveActiveChainIds is a paid mutator transaction binding the contract method 0x9f600a0b.
//
// Solidity: function removeActiveChainIds(uint256[] chainIds) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) RemoveActiveChainIds(chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.RemoveActiveChainIds(&_HopL2BaseBridge.TransactOpts, chainIds)
}

// RemoveActiveChainIds is a paid mutator transaction binding the contract method 0x9f600a0b.
//
// Solidity: function removeActiveChainIds(uint256[] chainIds) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) RemoveActiveChainIds(chainIds []*big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.RemoveActiveChainIds(&_HopL2BaseBridge.TransactOpts, chainIds)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) RemoveBonder(opts *bind.TransactOpts, bonder common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "removeBonder", bonder)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) RemoveBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.RemoveBonder(&_HopL2BaseBridge.TransactOpts, bonder)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) RemoveBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.RemoveBonder(&_HopL2BaseBridge.TransactOpts, bonder)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) RescueTransferRoot(opts *bind.TransactOpts, rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "rescueTransferRoot", rootHash, originalAmount, recipient)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) RescueTransferRoot(rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.RescueTransferRoot(&_HopL2BaseBridge.TransactOpts, rootHash, originalAmount, recipient)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) RescueTransferRoot(rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.RescueTransferRoot(&_HopL2BaseBridge.TransactOpts, rootHash, originalAmount, recipient)
}

// Send is a paid mutator transaction binding the contract method 0xa6bd1b33.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) Send(opts *bind.TransactOpts, chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "send", chainId, recipient, amount, bonderFee, amountOutMin, deadline)
}

// Send is a paid mutator transaction binding the contract method 0xa6bd1b33.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) Send(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Send(&_HopL2BaseBridge.TransactOpts, chainId, recipient, amount, bonderFee, amountOutMin, deadline)
}

// Send is a paid mutator transaction binding the contract method 0xa6bd1b33.
//
// Solidity: function send(uint256 chainId, address recipient, uint256 amount, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) Send(chainId *big.Int, recipient common.Address, amount *big.Int, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Send(&_HopL2BaseBridge.TransactOpts, chainId, recipient, amount, bonderFee, amountOutMin, deadline)
}

// SetAmmWrapper is a paid mutator transaction binding the contract method 0x64c6fdb4.
//
// Solidity: function setAmmWrapper(address _ammWrapper) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetAmmWrapper(opts *bind.TransactOpts, _ammWrapper common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setAmmWrapper", _ammWrapper)
}

// SetAmmWrapper is a paid mutator transaction binding the contract method 0x64c6fdb4.
//
// Solidity: function setAmmWrapper(address _ammWrapper) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetAmmWrapper(_ammWrapper common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetAmmWrapper(&_HopL2BaseBridge.TransactOpts, _ammWrapper)
}

// SetAmmWrapper is a paid mutator transaction binding the contract method 0x64c6fdb4.
//
// Solidity: function setAmmWrapper(address _ammWrapper) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetAmmWrapper(_ammWrapper common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetAmmWrapper(&_HopL2BaseBridge.TransactOpts, _ammWrapper)
}

// SetDefaultGasLimit is a paid mutator transaction binding the contract method 0x524b6f70.
//
// Solidity: function setDefaultGasLimit(uint32 _defaultGasLimit) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetDefaultGasLimit(opts *bind.TransactOpts, _defaultGasLimit uint32) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setDefaultGasLimit", _defaultGasLimit)
}

// SetDefaultGasLimit is a paid mutator transaction binding the contract method 0x524b6f70.
//
// Solidity: function setDefaultGasLimit(uint32 _defaultGasLimit) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetDefaultGasLimit(_defaultGasLimit uint32) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetDefaultGasLimit(&_HopL2BaseBridge.TransactOpts, _defaultGasLimit)
}

// SetDefaultGasLimit is a paid mutator transaction binding the contract method 0x524b6f70.
//
// Solidity: function setDefaultGasLimit(uint32 _defaultGasLimit) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetDefaultGasLimit(_defaultGasLimit uint32) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetDefaultGasLimit(&_HopL2BaseBridge.TransactOpts, _defaultGasLimit)
}

// SetHopBridgeTokenOwner is a paid mutator transaction binding the contract method 0x8295f258.
//
// Solidity: function setHopBridgeTokenOwner(address newOwner) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetHopBridgeTokenOwner(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setHopBridgeTokenOwner", newOwner)
}

// SetHopBridgeTokenOwner is a paid mutator transaction binding the contract method 0x8295f258.
//
// Solidity: function setHopBridgeTokenOwner(address newOwner) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetHopBridgeTokenOwner(newOwner common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetHopBridgeTokenOwner(&_HopL2BaseBridge.TransactOpts, newOwner)
}

// SetHopBridgeTokenOwner is a paid mutator transaction binding the contract method 0x8295f258.
//
// Solidity: function setHopBridgeTokenOwner(address newOwner) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetHopBridgeTokenOwner(newOwner common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetHopBridgeTokenOwner(&_HopL2BaseBridge.TransactOpts, newOwner)
}

// SetL1BridgeAddress is a paid mutator transaction binding the contract method 0xe1825d06.
//
// Solidity: function setL1BridgeAddress(address _l1BridgeAddress) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetL1BridgeAddress(opts *bind.TransactOpts, _l1BridgeAddress common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setL1BridgeAddress", _l1BridgeAddress)
}

// SetL1BridgeAddress is a paid mutator transaction binding the contract method 0xe1825d06.
//
// Solidity: function setL1BridgeAddress(address _l1BridgeAddress) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetL1BridgeAddress(_l1BridgeAddress common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetL1BridgeAddress(&_HopL2BaseBridge.TransactOpts, _l1BridgeAddress)
}

// SetL1BridgeAddress is a paid mutator transaction binding the contract method 0xe1825d06.
//
// Solidity: function setL1BridgeAddress(address _l1BridgeAddress) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetL1BridgeAddress(_l1BridgeAddress common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetL1BridgeAddress(&_HopL2BaseBridge.TransactOpts, _l1BridgeAddress)
}

// SetL1BridgeCaller is a paid mutator transaction binding the contract method 0xaf33ae69.
//
// Solidity: function setL1BridgeCaller(address _l1BridgeCaller) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetL1BridgeCaller(opts *bind.TransactOpts, _l1BridgeCaller common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setL1BridgeCaller", _l1BridgeCaller)
}

// SetL1BridgeCaller is a paid mutator transaction binding the contract method 0xaf33ae69.
//
// Solidity: function setL1BridgeCaller(address _l1BridgeCaller) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetL1BridgeCaller(_l1BridgeCaller common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetL1BridgeCaller(&_HopL2BaseBridge.TransactOpts, _l1BridgeCaller)
}

// SetL1BridgeCaller is a paid mutator transaction binding the contract method 0xaf33ae69.
//
// Solidity: function setL1BridgeCaller(address _l1BridgeCaller) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetL1BridgeCaller(_l1BridgeCaller common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetL1BridgeCaller(&_HopL2BaseBridge.TransactOpts, _l1BridgeCaller)
}

// SetL1Governance is a paid mutator transaction binding the contract method 0xe40272d7.
//
// Solidity: function setL1Governance(address _l1Governance) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetL1Governance(opts *bind.TransactOpts, _l1Governance common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setL1Governance", _l1Governance)
}

// SetL1Governance is a paid mutator transaction binding the contract method 0xe40272d7.
//
// Solidity: function setL1Governance(address _l1Governance) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetL1Governance(_l1Governance common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetL1Governance(&_HopL2BaseBridge.TransactOpts, _l1Governance)
}

// SetL1Governance is a paid mutator transaction binding the contract method 0xe40272d7.
//
// Solidity: function setL1Governance(address _l1Governance) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetL1Governance(_l1Governance common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetL1Governance(&_HopL2BaseBridge.TransactOpts, _l1Governance)
}

// SetMaxPendingTransfers is a paid mutator transaction binding the contract method 0x4742bbfb.
//
// Solidity: function setMaxPendingTransfers(uint256 _maxPendingTransfers) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetMaxPendingTransfers(opts *bind.TransactOpts, _maxPendingTransfers *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setMaxPendingTransfers", _maxPendingTransfers)
}

// SetMaxPendingTransfers is a paid mutator transaction binding the contract method 0x4742bbfb.
//
// Solidity: function setMaxPendingTransfers(uint256 _maxPendingTransfers) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetMaxPendingTransfers(_maxPendingTransfers *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetMaxPendingTransfers(&_HopL2BaseBridge.TransactOpts, _maxPendingTransfers)
}

// SetMaxPendingTransfers is a paid mutator transaction binding the contract method 0x4742bbfb.
//
// Solidity: function setMaxPendingTransfers(uint256 _maxPendingTransfers) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetMaxPendingTransfers(_maxPendingTransfers *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetMaxPendingTransfers(&_HopL2BaseBridge.TransactOpts, _maxPendingTransfers)
}

// SetMessenger is a paid mutator transaction binding the contract method 0x66285967.
//
// Solidity: function setMessenger(address _messenger) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetMessenger(opts *bind.TransactOpts, _messenger common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setMessenger", _messenger)
}

// SetMessenger is a paid mutator transaction binding the contract method 0x66285967.
//
// Solidity: function setMessenger(address _messenger) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetMessenger(_messenger common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetMessenger(&_HopL2BaseBridge.TransactOpts, _messenger)
}

// SetMessenger is a paid mutator transaction binding the contract method 0x66285967.
//
// Solidity: function setMessenger(address _messenger) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetMessenger(_messenger common.Address) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetMessenger(&_HopL2BaseBridge.TransactOpts, _messenger)
}

// SetMinimumBonderFeeRequirements is a paid mutator transaction binding the contract method 0xa9fa4ed5.
//
// Solidity: function setMinimumBonderFeeRequirements(uint256 _minBonderBps, uint256 _minBonderFeeAbsolute) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetMinimumBonderFeeRequirements(opts *bind.TransactOpts, _minBonderBps *big.Int, _minBonderFeeAbsolute *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setMinimumBonderFeeRequirements", _minBonderBps, _minBonderFeeAbsolute)
}

// SetMinimumBonderFeeRequirements is a paid mutator transaction binding the contract method 0xa9fa4ed5.
//
// Solidity: function setMinimumBonderFeeRequirements(uint256 _minBonderBps, uint256 _minBonderFeeAbsolute) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetMinimumBonderFeeRequirements(_minBonderBps *big.Int, _minBonderFeeAbsolute *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetMinimumBonderFeeRequirements(&_HopL2BaseBridge.TransactOpts, _minBonderBps, _minBonderFeeAbsolute)
}

// SetMinimumBonderFeeRequirements is a paid mutator transaction binding the contract method 0xa9fa4ed5.
//
// Solidity: function setMinimumBonderFeeRequirements(uint256 _minBonderBps, uint256 _minBonderFeeAbsolute) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetMinimumBonderFeeRequirements(_minBonderBps *big.Int, _minBonderFeeAbsolute *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetMinimumBonderFeeRequirements(&_HopL2BaseBridge.TransactOpts, _minBonderBps, _minBonderFeeAbsolute)
}

// SetMinimumForceCommitDelay is a paid mutator transaction binding the contract method 0x9bf43028.
//
// Solidity: function setMinimumForceCommitDelay(uint256 _minimumForceCommitDelay) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetMinimumForceCommitDelay(opts *bind.TransactOpts, _minimumForceCommitDelay *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setMinimumForceCommitDelay", _minimumForceCommitDelay)
}

// SetMinimumForceCommitDelay is a paid mutator transaction binding the contract method 0x9bf43028.
//
// Solidity: function setMinimumForceCommitDelay(uint256 _minimumForceCommitDelay) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetMinimumForceCommitDelay(_minimumForceCommitDelay *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetMinimumForceCommitDelay(&_HopL2BaseBridge.TransactOpts, _minimumForceCommitDelay)
}

// SetMinimumForceCommitDelay is a paid mutator transaction binding the contract method 0x9bf43028.
//
// Solidity: function setMinimumForceCommitDelay(uint256 _minimumForceCommitDelay) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetMinimumForceCommitDelay(_minimumForceCommitDelay *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetMinimumForceCommitDelay(&_HopL2BaseBridge.TransactOpts, _minimumForceCommitDelay)
}

// SetTransferRoot is a paid mutator transaction binding the contract method 0xfd31c5ba.
//
// Solidity: function setTransferRoot(bytes32 rootHash, uint256 totalAmount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SetTransferRoot(opts *bind.TransactOpts, rootHash [32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "setTransferRoot", rootHash, totalAmount)
}

// SetTransferRoot is a paid mutator transaction binding the contract method 0xfd31c5ba.
//
// Solidity: function setTransferRoot(bytes32 rootHash, uint256 totalAmount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetTransferRoot(&_HopL2BaseBridge.TransactOpts, rootHash, totalAmount)
}

// SetTransferRoot is a paid mutator transaction binding the contract method 0xfd31c5ba.
//
// Solidity: function setTransferRoot(bytes32 rootHash, uint256 totalAmount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SetTransferRoot(&_HopL2BaseBridge.TransactOpts, rootHash, totalAmount)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SettleBondedWithdrawal(opts *bind.TransactOpts, bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "settleBondedWithdrawal", bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SettleBondedWithdrawal(bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SettleBondedWithdrawal(&_HopL2BaseBridge.TransactOpts, bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SettleBondedWithdrawal(bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SettleBondedWithdrawal(&_HopL2BaseBridge.TransactOpts, bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) SettleBondedWithdrawals(opts *bind.TransactOpts, bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "settleBondedWithdrawals", bonder, transferIds, totalAmount)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) SettleBondedWithdrawals(bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SettleBondedWithdrawals(&_HopL2BaseBridge.TransactOpts, bonder, transferIds, totalAmount)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) SettleBondedWithdrawals(bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.SettleBondedWithdrawals(&_HopL2BaseBridge.TransactOpts, bonder, transferIds, totalAmount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) Stake(opts *bind.TransactOpts, bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "stake", bonder, amount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) Stake(bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Stake(&_HopL2BaseBridge.TransactOpts, bonder, amount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) Stake(bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Stake(&_HopL2BaseBridge.TransactOpts, bonder, amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) Unstake(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "unstake", amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) Unstake(amount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Unstake(&_HopL2BaseBridge.TransactOpts, amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) Unstake(amount *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Unstake(&_HopL2BaseBridge.TransactOpts, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactor) Withdraw(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.contract.Transact(opts, "withdraw", recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeSession) Withdraw(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Withdraw(&_HopL2BaseBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL2BaseBridge *HopL2BaseBridgeTransactorSession) Withdraw(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL2BaseBridge.Contract.Withdraw(&_HopL2BaseBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// HopL2BaseBridgeBonderAddedIterator is returned from FilterBonderAdded and is used to iterate over the raw logs and unpacked data for BonderAdded events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeBonderAddedIterator struct {
	Event *HopL2BaseBridgeBonderAdded // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeBonderAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeBonderAdded)
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
		it.Event = new(HopL2BaseBridgeBonderAdded)
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
func (it *HopL2BaseBridgeBonderAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeBonderAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeBonderAdded represents a BonderAdded event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeBonderAdded struct {
	NewBonder common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterBonderAdded is a free log retrieval operation binding the contract event 0x2cec73b7434d3b91198ad1a618f63e6a0761ce281af5ec9ec76606d948d03e23.
//
// Solidity: event BonderAdded(address indexed newBonder)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterBonderAdded(opts *bind.FilterOpts, newBonder []common.Address) (*HopL2BaseBridgeBonderAddedIterator, error) {

	var newBonderRule []interface{}
	for _, newBonderItem := range newBonder {
		newBonderRule = append(newBonderRule, newBonderItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "BonderAdded", newBonderRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeBonderAddedIterator{contract: _HopL2BaseBridge.contract, event: "BonderAdded", logs: logs, sub: sub}, nil
}

// WatchBonderAdded is a free log subscription operation binding the contract event 0x2cec73b7434d3b91198ad1a618f63e6a0761ce281af5ec9ec76606d948d03e23.
//
// Solidity: event BonderAdded(address indexed newBonder)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchBonderAdded(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeBonderAdded, newBonder []common.Address) (event.Subscription, error) {

	var newBonderRule []interface{}
	for _, newBonderItem := range newBonder {
		newBonderRule = append(newBonderRule, newBonderItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "BonderAdded", newBonderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeBonderAdded)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "BonderAdded", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseBonderAdded(log types.Log) (*HopL2BaseBridgeBonderAdded, error) {
	event := new(HopL2BaseBridgeBonderAdded)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "BonderAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeBonderRemovedIterator is returned from FilterBonderRemoved and is used to iterate over the raw logs and unpacked data for BonderRemoved events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeBonderRemovedIterator struct {
	Event *HopL2BaseBridgeBonderRemoved // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeBonderRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeBonderRemoved)
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
		it.Event = new(HopL2BaseBridgeBonderRemoved)
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
func (it *HopL2BaseBridgeBonderRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeBonderRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeBonderRemoved represents a BonderRemoved event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeBonderRemoved struct {
	PreviousBonder common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterBonderRemoved is a free log retrieval operation binding the contract event 0x4234ba611d325b3ba434c4e1b037967b955b1274d4185ee9847b7491111a48ff.
//
// Solidity: event BonderRemoved(address indexed previousBonder)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterBonderRemoved(opts *bind.FilterOpts, previousBonder []common.Address) (*HopL2BaseBridgeBonderRemovedIterator, error) {

	var previousBonderRule []interface{}
	for _, previousBonderItem := range previousBonder {
		previousBonderRule = append(previousBonderRule, previousBonderItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "BonderRemoved", previousBonderRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeBonderRemovedIterator{contract: _HopL2BaseBridge.contract, event: "BonderRemoved", logs: logs, sub: sub}, nil
}

// WatchBonderRemoved is a free log subscription operation binding the contract event 0x4234ba611d325b3ba434c4e1b037967b955b1274d4185ee9847b7491111a48ff.
//
// Solidity: event BonderRemoved(address indexed previousBonder)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchBonderRemoved(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeBonderRemoved, previousBonder []common.Address) (event.Subscription, error) {

	var previousBonderRule []interface{}
	for _, previousBonderItem := range previousBonder {
		previousBonderRule = append(previousBonderRule, previousBonderItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "BonderRemoved", previousBonderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeBonderRemoved)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "BonderRemoved", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseBonderRemoved(log types.Log) (*HopL2BaseBridgeBonderRemoved, error) {
	event := new(HopL2BaseBridgeBonderRemoved)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "BonderRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeMultipleWithdrawalsSettledIterator is returned from FilterMultipleWithdrawalsSettled and is used to iterate over the raw logs and unpacked data for MultipleWithdrawalsSettled events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeMultipleWithdrawalsSettledIterator struct {
	Event *HopL2BaseBridgeMultipleWithdrawalsSettled // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeMultipleWithdrawalsSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeMultipleWithdrawalsSettled)
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
		it.Event = new(HopL2BaseBridgeMultipleWithdrawalsSettled)
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
func (it *HopL2BaseBridgeMultipleWithdrawalsSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeMultipleWithdrawalsSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeMultipleWithdrawalsSettled represents a MultipleWithdrawalsSettled event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeMultipleWithdrawalsSettled struct {
	Bonder            common.Address
	RootHash          [32]byte
	TotalBondsSettled *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterMultipleWithdrawalsSettled is a free log retrieval operation binding the contract event 0x78e830d08be9d5f957414c84d685c061ecbd8467be98b42ebb64f0118b57d2ff.
//
// Solidity: event MultipleWithdrawalsSettled(address indexed bonder, bytes32 indexed rootHash, uint256 totalBondsSettled)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterMultipleWithdrawalsSettled(opts *bind.FilterOpts, bonder []common.Address, rootHash [][32]byte) (*HopL2BaseBridgeMultipleWithdrawalsSettledIterator, error) {

	var bonderRule []interface{}
	for _, bonderItem := range bonder {
		bonderRule = append(bonderRule, bonderItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "MultipleWithdrawalsSettled", bonderRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeMultipleWithdrawalsSettledIterator{contract: _HopL2BaseBridge.contract, event: "MultipleWithdrawalsSettled", logs: logs, sub: sub}, nil
}

// WatchMultipleWithdrawalsSettled is a free log subscription operation binding the contract event 0x78e830d08be9d5f957414c84d685c061ecbd8467be98b42ebb64f0118b57d2ff.
//
// Solidity: event MultipleWithdrawalsSettled(address indexed bonder, bytes32 indexed rootHash, uint256 totalBondsSettled)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchMultipleWithdrawalsSettled(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeMultipleWithdrawalsSettled, bonder []common.Address, rootHash [][32]byte) (event.Subscription, error) {

	var bonderRule []interface{}
	for _, bonderItem := range bonder {
		bonderRule = append(bonderRule, bonderItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "MultipleWithdrawalsSettled", bonderRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeMultipleWithdrawalsSettled)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "MultipleWithdrawalsSettled", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseMultipleWithdrawalsSettled(log types.Log) (*HopL2BaseBridgeMultipleWithdrawalsSettled, error) {
	event := new(HopL2BaseBridgeMultipleWithdrawalsSettled)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "MultipleWithdrawalsSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeStakeIterator is returned from FilterStake and is used to iterate over the raw logs and unpacked data for Stake events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeStakeIterator struct {
	Event *HopL2BaseBridgeStake // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeStakeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeStake)
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
		it.Event = new(HopL2BaseBridgeStake)
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
func (it *HopL2BaseBridgeStakeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeStakeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeStake represents a Stake event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeStake struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterStake is a free log retrieval operation binding the contract event 0xebedb8b3c678666e7f36970bc8f57abf6d8fa2e828c0da91ea5b75bf68ed101a.
//
// Solidity: event Stake(address indexed account, uint256 amount)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterStake(opts *bind.FilterOpts, account []common.Address) (*HopL2BaseBridgeStakeIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "Stake", accountRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeStakeIterator{contract: _HopL2BaseBridge.contract, event: "Stake", logs: logs, sub: sub}, nil
}

// WatchStake is a free log subscription operation binding the contract event 0xebedb8b3c678666e7f36970bc8f57abf6d8fa2e828c0da91ea5b75bf68ed101a.
//
// Solidity: event Stake(address indexed account, uint256 amount)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchStake(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeStake, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "Stake", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeStake)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "Stake", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseStake(log types.Log) (*HopL2BaseBridgeStake, error) {
	event := new(HopL2BaseBridgeStake)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "Stake", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeTransferFromL1CompletedIterator is returned from FilterTransferFromL1Completed and is used to iterate over the raw logs and unpacked data for TransferFromL1Completed events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeTransferFromL1CompletedIterator struct {
	Event *HopL2BaseBridgeTransferFromL1Completed // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeTransferFromL1CompletedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeTransferFromL1Completed)
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
		it.Event = new(HopL2BaseBridgeTransferFromL1Completed)
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
func (it *HopL2BaseBridgeTransferFromL1CompletedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeTransferFromL1CompletedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeTransferFromL1Completed represents a TransferFromL1Completed event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeTransferFromL1Completed struct {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterTransferFromL1Completed(opts *bind.FilterOpts, recipient []common.Address, relayer []common.Address) (*HopL2BaseBridgeTransferFromL1CompletedIterator, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	var relayerRule []interface{}
	for _, relayerItem := range relayer {
		relayerRule = append(relayerRule, relayerItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "TransferFromL1Completed", recipientRule, relayerRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeTransferFromL1CompletedIterator{contract: _HopL2BaseBridge.contract, event: "TransferFromL1Completed", logs: logs, sub: sub}, nil
}

// WatchTransferFromL1Completed is a free log subscription operation binding the contract event 0x320958176930804eb66c2343c7343fc0367dc16249590c0f195783bee199d094.
//
// Solidity: event TransferFromL1Completed(address indexed recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address indexed relayer, uint256 relayerFee)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchTransferFromL1Completed(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeTransferFromL1Completed, recipient []common.Address, relayer []common.Address) (event.Subscription, error) {

	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	var relayerRule []interface{}
	for _, relayerItem := range relayer {
		relayerRule = append(relayerRule, relayerItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "TransferFromL1Completed", recipientRule, relayerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeTransferFromL1Completed)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "TransferFromL1Completed", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseTransferFromL1Completed(log types.Log) (*HopL2BaseBridgeTransferFromL1Completed, error) {
	event := new(HopL2BaseBridgeTransferFromL1Completed)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "TransferFromL1Completed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeTransferRootSetIterator is returned from FilterTransferRootSet and is used to iterate over the raw logs and unpacked data for TransferRootSet events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeTransferRootSetIterator struct {
	Event *HopL2BaseBridgeTransferRootSet // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeTransferRootSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeTransferRootSet)
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
		it.Event = new(HopL2BaseBridgeTransferRootSet)
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
func (it *HopL2BaseBridgeTransferRootSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeTransferRootSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeTransferRootSet represents a TransferRootSet event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeTransferRootSet struct {
	RootHash    [32]byte
	TotalAmount *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTransferRootSet is a free log retrieval operation binding the contract event 0xb33d2162aead99dab59e77a7a67ea025b776bf8ca8079e132afdf9b23e03bd42.
//
// Solidity: event TransferRootSet(bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterTransferRootSet(opts *bind.FilterOpts, rootHash [][32]byte) (*HopL2BaseBridgeTransferRootSetIterator, error) {

	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "TransferRootSet", rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeTransferRootSetIterator{contract: _HopL2BaseBridge.contract, event: "TransferRootSet", logs: logs, sub: sub}, nil
}

// WatchTransferRootSet is a free log subscription operation binding the contract event 0xb33d2162aead99dab59e77a7a67ea025b776bf8ca8079e132afdf9b23e03bd42.
//
// Solidity: event TransferRootSet(bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchTransferRootSet(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeTransferRootSet, rootHash [][32]byte) (event.Subscription, error) {

	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "TransferRootSet", rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeTransferRootSet)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "TransferRootSet", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseTransferRootSet(log types.Log) (*HopL2BaseBridgeTransferRootSet, error) {
	event := new(HopL2BaseBridgeTransferRootSet)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "TransferRootSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeTransferSentIterator is returned from FilterTransferSent and is used to iterate over the raw logs and unpacked data for TransferSent events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeTransferSentIterator struct {
	Event *HopL2BaseBridgeTransferSent // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeTransferSentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeTransferSent)
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
		it.Event = new(HopL2BaseBridgeTransferSent)
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
func (it *HopL2BaseBridgeTransferSentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeTransferSentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeTransferSent represents a TransferSent event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeTransferSent struct {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterTransferSent(opts *bind.FilterOpts, transferId [][32]byte, chainId []*big.Int, recipient []common.Address) (*HopL2BaseBridgeTransferSentIterator, error) {

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

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "TransferSent", transferIdRule, chainIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeTransferSentIterator{contract: _HopL2BaseBridge.contract, event: "TransferSent", logs: logs, sub: sub}, nil
}

// WatchTransferSent is a free log subscription operation binding the contract event 0xe35dddd4ea75d7e9b3fe93af4f4e40e778c3da4074c9d93e7c6536f1e803c1eb.
//
// Solidity: event TransferSent(bytes32 indexed transferId, uint256 indexed chainId, address indexed recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 index, uint256 amountOutMin, uint256 deadline)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchTransferSent(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeTransferSent, transferId [][32]byte, chainId []*big.Int, recipient []common.Address) (event.Subscription, error) {

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

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "TransferSent", transferIdRule, chainIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeTransferSent)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "TransferSent", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseTransferSent(log types.Log) (*HopL2BaseBridgeTransferSent, error) {
	event := new(HopL2BaseBridgeTransferSent)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "TransferSent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeTransfersCommittedIterator is returned from FilterTransfersCommitted and is used to iterate over the raw logs and unpacked data for TransfersCommitted events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeTransfersCommittedIterator struct {
	Event *HopL2BaseBridgeTransfersCommitted // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeTransfersCommittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeTransfersCommitted)
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
		it.Event = new(HopL2BaseBridgeTransfersCommitted)
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
func (it *HopL2BaseBridgeTransfersCommittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeTransfersCommittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeTransfersCommitted represents a TransfersCommitted event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeTransfersCommitted struct {
	DestinationChainId *big.Int
	RootHash           [32]byte
	TotalAmount        *big.Int
	RootCommittedAt    *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterTransfersCommitted is a free log retrieval operation binding the contract event 0xf52ad20d3b4f50d1c40901dfb95a9ce5270b2fc32694e5c668354721cd87aa74.
//
// Solidity: event TransfersCommitted(uint256 indexed destinationChainId, bytes32 indexed rootHash, uint256 totalAmount, uint256 rootCommittedAt)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterTransfersCommitted(opts *bind.FilterOpts, destinationChainId []*big.Int, rootHash [][32]byte) (*HopL2BaseBridgeTransfersCommittedIterator, error) {

	var destinationChainIdRule []interface{}
	for _, destinationChainIdItem := range destinationChainId {
		destinationChainIdRule = append(destinationChainIdRule, destinationChainIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "TransfersCommitted", destinationChainIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeTransfersCommittedIterator{contract: _HopL2BaseBridge.contract, event: "TransfersCommitted", logs: logs, sub: sub}, nil
}

// WatchTransfersCommitted is a free log subscription operation binding the contract event 0xf52ad20d3b4f50d1c40901dfb95a9ce5270b2fc32694e5c668354721cd87aa74.
//
// Solidity: event TransfersCommitted(uint256 indexed destinationChainId, bytes32 indexed rootHash, uint256 totalAmount, uint256 rootCommittedAt)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchTransfersCommitted(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeTransfersCommitted, destinationChainId []*big.Int, rootHash [][32]byte) (event.Subscription, error) {

	var destinationChainIdRule []interface{}
	for _, destinationChainIdItem := range destinationChainId {
		destinationChainIdRule = append(destinationChainIdRule, destinationChainIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "TransfersCommitted", destinationChainIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeTransfersCommitted)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "TransfersCommitted", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseTransfersCommitted(log types.Log) (*HopL2BaseBridgeTransfersCommitted, error) {
	event := new(HopL2BaseBridgeTransfersCommitted)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "TransfersCommitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeUnstakeIterator is returned from FilterUnstake and is used to iterate over the raw logs and unpacked data for Unstake events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeUnstakeIterator struct {
	Event *HopL2BaseBridgeUnstake // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeUnstakeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeUnstake)
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
		it.Event = new(HopL2BaseBridgeUnstake)
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
func (it *HopL2BaseBridgeUnstakeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeUnstakeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeUnstake represents a Unstake event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeUnstake struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnstake is a free log retrieval operation binding the contract event 0x85082129d87b2fe11527cb1b3b7a520aeb5aa6913f88a3d8757fe40d1db02fdd.
//
// Solidity: event Unstake(address indexed account, uint256 amount)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterUnstake(opts *bind.FilterOpts, account []common.Address) (*HopL2BaseBridgeUnstakeIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "Unstake", accountRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeUnstakeIterator{contract: _HopL2BaseBridge.contract, event: "Unstake", logs: logs, sub: sub}, nil
}

// WatchUnstake is a free log subscription operation binding the contract event 0x85082129d87b2fe11527cb1b3b7a520aeb5aa6913f88a3d8757fe40d1db02fdd.
//
// Solidity: event Unstake(address indexed account, uint256 amount)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchUnstake(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeUnstake, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "Unstake", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeUnstake)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "Unstake", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseUnstake(log types.Log) (*HopL2BaseBridgeUnstake, error) {
	event := new(HopL2BaseBridgeUnstake)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "Unstake", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeWithdrawalBondSettledIterator is returned from FilterWithdrawalBondSettled and is used to iterate over the raw logs and unpacked data for WithdrawalBondSettled events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeWithdrawalBondSettledIterator struct {
	Event *HopL2BaseBridgeWithdrawalBondSettled // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeWithdrawalBondSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeWithdrawalBondSettled)
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
		it.Event = new(HopL2BaseBridgeWithdrawalBondSettled)
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
func (it *HopL2BaseBridgeWithdrawalBondSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeWithdrawalBondSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeWithdrawalBondSettled represents a WithdrawalBondSettled event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeWithdrawalBondSettled struct {
	Bonder     common.Address
	TransferId [32]byte
	RootHash   [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalBondSettled is a free log retrieval operation binding the contract event 0x84eb21b24c31b27a3bc67dde4a598aad06db6e9415cd66544492b9616996143c.
//
// Solidity: event WithdrawalBondSettled(address indexed bonder, bytes32 indexed transferId, bytes32 indexed rootHash)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterWithdrawalBondSettled(opts *bind.FilterOpts, bonder []common.Address, transferId [][32]byte, rootHash [][32]byte) (*HopL2BaseBridgeWithdrawalBondSettledIterator, error) {

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

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "WithdrawalBondSettled", bonderRule, transferIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeWithdrawalBondSettledIterator{contract: _HopL2BaseBridge.contract, event: "WithdrawalBondSettled", logs: logs, sub: sub}, nil
}

// WatchWithdrawalBondSettled is a free log subscription operation binding the contract event 0x84eb21b24c31b27a3bc67dde4a598aad06db6e9415cd66544492b9616996143c.
//
// Solidity: event WithdrawalBondSettled(address indexed bonder, bytes32 indexed transferId, bytes32 indexed rootHash)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchWithdrawalBondSettled(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeWithdrawalBondSettled, bonder []common.Address, transferId [][32]byte, rootHash [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "WithdrawalBondSettled", bonderRule, transferIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeWithdrawalBondSettled)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "WithdrawalBondSettled", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseWithdrawalBondSettled(log types.Log) (*HopL2BaseBridgeWithdrawalBondSettled, error) {
	event := new(HopL2BaseBridgeWithdrawalBondSettled)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "WithdrawalBondSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeWithdrawalBondedIterator is returned from FilterWithdrawalBonded and is used to iterate over the raw logs and unpacked data for WithdrawalBonded events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeWithdrawalBondedIterator struct {
	Event *HopL2BaseBridgeWithdrawalBonded // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeWithdrawalBondedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeWithdrawalBonded)
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
		it.Event = new(HopL2BaseBridgeWithdrawalBonded)
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
func (it *HopL2BaseBridgeWithdrawalBondedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeWithdrawalBondedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeWithdrawalBonded represents a WithdrawalBonded event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeWithdrawalBonded struct {
	TransferId [32]byte
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalBonded is a free log retrieval operation binding the contract event 0x0c3d250c7831051e78aa6a56679e590374c7c424415ffe4aa474491def2fe705.
//
// Solidity: event WithdrawalBonded(bytes32 indexed transferId, uint256 amount)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterWithdrawalBonded(opts *bind.FilterOpts, transferId [][32]byte) (*HopL2BaseBridgeWithdrawalBondedIterator, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "WithdrawalBonded", transferIdRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeWithdrawalBondedIterator{contract: _HopL2BaseBridge.contract, event: "WithdrawalBonded", logs: logs, sub: sub}, nil
}

// WatchWithdrawalBonded is a free log subscription operation binding the contract event 0x0c3d250c7831051e78aa6a56679e590374c7c424415ffe4aa474491def2fe705.
//
// Solidity: event WithdrawalBonded(bytes32 indexed transferId, uint256 amount)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchWithdrawalBonded(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeWithdrawalBonded, transferId [][32]byte) (event.Subscription, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "WithdrawalBonded", transferIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeWithdrawalBonded)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "WithdrawalBonded", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseWithdrawalBonded(log types.Log) (*HopL2BaseBridgeWithdrawalBonded, error) {
	event := new(HopL2BaseBridgeWithdrawalBonded)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "WithdrawalBonded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL2BaseBridgeWithdrewIterator is returned from FilterWithdrew and is used to iterate over the raw logs and unpacked data for Withdrew events raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeWithdrewIterator struct {
	Event *HopL2BaseBridgeWithdrew // Event containing the contract specifics and raw log

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
func (it *HopL2BaseBridgeWithdrewIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL2BaseBridgeWithdrew)
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
		it.Event = new(HopL2BaseBridgeWithdrew)
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
func (it *HopL2BaseBridgeWithdrewIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL2BaseBridgeWithdrewIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL2BaseBridgeWithdrew represents a Withdrew event raised by the HopL2BaseBridge contract.
type HopL2BaseBridgeWithdrew struct {
	TransferId    [32]byte
	Recipient     common.Address
	Amount        *big.Int
	TransferNonce [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterWithdrew is a free log retrieval operation binding the contract event 0x9475cdbde5fc71fe2ccd413c82878ee54d061b9f74f9e2e1a03ff1178821502c.
//
// Solidity: event Withdrew(bytes32 indexed transferId, address indexed recipient, uint256 amount, bytes32 transferNonce)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) FilterWithdrew(opts *bind.FilterOpts, transferId [][32]byte, recipient []common.Address) (*HopL2BaseBridgeWithdrewIterator, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.FilterLogs(opts, "Withdrew", transferIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &HopL2BaseBridgeWithdrewIterator{contract: _HopL2BaseBridge.contract, event: "Withdrew", logs: logs, sub: sub}, nil
}

// WatchWithdrew is a free log subscription operation binding the contract event 0x9475cdbde5fc71fe2ccd413c82878ee54d061b9f74f9e2e1a03ff1178821502c.
//
// Solidity: event Withdrew(bytes32 indexed transferId, address indexed recipient, uint256 amount, bytes32 transferNonce)
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) WatchWithdrew(opts *bind.WatchOpts, sink chan<- *HopL2BaseBridgeWithdrew, transferId [][32]byte, recipient []common.Address) (event.Subscription, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL2BaseBridge.contract.WatchLogs(opts, "Withdrew", transferIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL2BaseBridgeWithdrew)
				if err := _HopL2BaseBridge.contract.UnpackLog(event, "Withdrew", log); err != nil {
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
func (_HopL2BaseBridge *HopL2BaseBridgeFilterer) ParseWithdrew(log types.Log) (*HopL2BaseBridgeWithdrew, error) {
	event := new(HopL2BaseBridgeWithdrew)
	if err := _HopL2BaseBridge.contract.UnpackLog(event, "Withdrew", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
