// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package hopL1EthBridge

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

// HopL1EthBridgeMetaData contains all meta data concerning the HopL1EthBridge contract.
var HopL1EthBridgeMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"bonders\",\"type\":\"address[]\"},{\"internalType\":\"address\",\"name\":\"_governance\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newBonder\",\"type\":\"address\"}],\"name\":\"BonderAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousBonder\",\"type\":\"address\"}],\"name\":\"BonderRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferRootId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"originalAmount\",\"type\":\"uint256\"}],\"name\":\"ChallengeResolved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalBondsSettled\",\"type\":\"uint256\"}],\"name\":\"MultipleWithdrawalsSettled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Stake\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferRootId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"originalAmount\",\"type\":\"uint256\"}],\"name\":\"TransferBondChallenged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"root\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"TransferRootBonded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"originChainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"TransferRootConfirmed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"TransferRootSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"relayer\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"relayerFee\",\"type\":\"uint256\"}],\"name\":\"TransferSentToL2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"Unstake\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"}],\"name\":\"WithdrawalBondSettled\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"WithdrawalBonded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"}],\"name\":\"Withdrew\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"CHALLENGE_AMOUNT_DIVISOR\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TIME_SLOT_SIZE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"addBonder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"bondTransferRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"}],\"name\":\"bondWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"chainBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengePeriod\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"challengeResolutionPeriod\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"originalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"}],\"name\":\"challengeTransferBond\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"originChainId\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"rootCommittedAt\",\"type\":\"uint256\"}],\"name\":\"confirmTransferRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"crossDomainMessengerWrappers\",\"outputs\":[{\"internalType\":\"contractIMessengerWrapper\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"getBondForTransferAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"}],\"name\":\"getBondedWithdrawalAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getChainId\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"getChallengeAmountForTransferAmount\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getCredit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getDebitAndAdditionalDebit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"maybeBonder\",\"type\":\"address\"}],\"name\":\"getIsBonder\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"getRawDebit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"time\",\"type\":\"uint256\"}],\"name\":\"getTimeSlot\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"}],\"name\":\"getTransferId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"getTransferRoot\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"total\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountWithdrawn\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"createdAt\",\"type\":\"uint256\"}],\"internalType\":\"structBridge.TransferRoot\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"getTransferRootId\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"governance\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"isChainIdPaused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"}],\"name\":\"isTransferIdSpent\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"minTransferRootBondDelay\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"}],\"name\":\"removeBonder\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"originalAmount\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"rescueTransferRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"originalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"destinationChainId\",\"type\":\"uint256\"}],\"name\":\"resolveChallenge\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"relayer\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"relayerFee\",\"type\":\"uint256\"}],\"name\":\"sendToL2\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isPaused\",\"type\":\"bool\"}],\"name\":\"setChainIdDepositsPaused\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_challengePeriod\",\"type\":\"uint256\"}],\"name\":\"setChallengePeriod\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_challengeResolutionPeriod\",\"type\":\"uint256\"}],\"name\":\"setChallengeResolutionPeriod\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"contractIMessengerWrapper\",\"name\":\"_crossDomainMessengerWrapper\",\"type\":\"address\"}],\"name\":\"setCrossDomainMessengerWrapper\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newGovernance\",\"type\":\"address\"}],\"name\":\"setGovernance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_minTransferRootBondDelay\",\"type\":\"uint256\"}],\"name\":\"setMinTransferRootBondDelay\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"transferId\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"transferRootTotalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"transferIdTreeIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalLeaves\",\"type\":\"uint256\"}],\"name\":\"settleBondedWithdrawal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"bytes32[]\",\"name\":\"transferIds\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"}],\"name\":\"settleBondedWithdrawals\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"stake\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"timeSlotToAmountBonded\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"transferBonds\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"bonder\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"createdAt\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"totalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"challengeStartTime\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"challenger\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"challengeResolved\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"transferRootCommittedAt\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"unstake\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"transferNonce\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"bonderFee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"amountOutMin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"rootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"transferRootTotalAmount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"transferIdTreeIndex\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"siblings\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"totalLeaves\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// HopL1EthBridgeABI is the input ABI used to generate the binding from.
// Deprecated: Use HopL1EthBridgeMetaData.ABI instead.
var HopL1EthBridgeABI = HopL1EthBridgeMetaData.ABI

// HopL1EthBridge is an auto generated Go binding around an Ethereum contract.
type HopL1EthBridge struct {
	HopL1EthBridgeCaller     // Read-only binding to the contract
	HopL1EthBridgeTransactor // Write-only binding to the contract
	HopL1EthBridgeFilterer   // Log filterer for contract events
}

// HopL1EthBridgeCaller is an auto generated read-only Go binding around an Ethereum contract.
type HopL1EthBridgeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL1EthBridgeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HopL1EthBridgeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL1EthBridgeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HopL1EthBridgeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HopL1EthBridgeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HopL1EthBridgeSession struct {
	Contract     *HopL1EthBridge   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// HopL1EthBridgeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HopL1EthBridgeCallerSession struct {
	Contract *HopL1EthBridgeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// HopL1EthBridgeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HopL1EthBridgeTransactorSession struct {
	Contract     *HopL1EthBridgeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// HopL1EthBridgeRaw is an auto generated low-level Go binding around an Ethereum contract.
type HopL1EthBridgeRaw struct {
	Contract *HopL1EthBridge // Generic contract binding to access the raw methods on
}

// HopL1EthBridgeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HopL1EthBridgeCallerRaw struct {
	Contract *HopL1EthBridgeCaller // Generic read-only contract binding to access the raw methods on
}

// HopL1EthBridgeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HopL1EthBridgeTransactorRaw struct {
	Contract *HopL1EthBridgeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHopL1EthBridge creates a new instance of HopL1EthBridge, bound to a specific deployed contract.
func NewHopL1EthBridge(address common.Address, backend bind.ContractBackend) (*HopL1EthBridge, error) {
	contract, err := bindHopL1EthBridge(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridge{HopL1EthBridgeCaller: HopL1EthBridgeCaller{contract: contract}, HopL1EthBridgeTransactor: HopL1EthBridgeTransactor{contract: contract}, HopL1EthBridgeFilterer: HopL1EthBridgeFilterer{contract: contract}}, nil
}

// NewHopL1EthBridgeCaller creates a new read-only instance of HopL1EthBridge, bound to a specific deployed contract.
func NewHopL1EthBridgeCaller(address common.Address, caller bind.ContractCaller) (*HopL1EthBridgeCaller, error) {
	contract, err := bindHopL1EthBridge(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeCaller{contract: contract}, nil
}

// NewHopL1EthBridgeTransactor creates a new write-only instance of HopL1EthBridge, bound to a specific deployed contract.
func NewHopL1EthBridgeTransactor(address common.Address, transactor bind.ContractTransactor) (*HopL1EthBridgeTransactor, error) {
	contract, err := bindHopL1EthBridge(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeTransactor{contract: contract}, nil
}

// NewHopL1EthBridgeFilterer creates a new log filterer instance of HopL1EthBridge, bound to a specific deployed contract.
func NewHopL1EthBridgeFilterer(address common.Address, filterer bind.ContractFilterer) (*HopL1EthBridgeFilterer, error) {
	contract, err := bindHopL1EthBridge(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeFilterer{contract: contract}, nil
}

// bindHopL1EthBridge binds a generic wrapper to an already deployed contract.
func bindHopL1EthBridge(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HopL1EthBridgeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL1EthBridge *HopL1EthBridgeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL1EthBridge.Contract.HopL1EthBridgeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL1EthBridge *HopL1EthBridgeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.HopL1EthBridgeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL1EthBridge *HopL1EthBridgeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.HopL1EthBridgeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HopL1EthBridge *HopL1EthBridgeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HopL1EthBridge.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HopL1EthBridge *HopL1EthBridgeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HopL1EthBridge *HopL1EthBridgeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.contract.Transact(opts, method, params...)
}

// CHALLENGEAMOUNTDIVISOR is a free data retrieval call binding the contract method 0x98c4f76d.
//
// Solidity: function CHALLENGE_AMOUNT_DIVISOR() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) CHALLENGEAMOUNTDIVISOR(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "CHALLENGE_AMOUNT_DIVISOR")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CHALLENGEAMOUNTDIVISOR is a free data retrieval call binding the contract method 0x98c4f76d.
//
// Solidity: function CHALLENGE_AMOUNT_DIVISOR() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) CHALLENGEAMOUNTDIVISOR() (*big.Int, error) {
	return _HopL1EthBridge.Contract.CHALLENGEAMOUNTDIVISOR(&_HopL1EthBridge.CallOpts)
}

// CHALLENGEAMOUNTDIVISOR is a free data retrieval call binding the contract method 0x98c4f76d.
//
// Solidity: function CHALLENGE_AMOUNT_DIVISOR() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) CHALLENGEAMOUNTDIVISOR() (*big.Int, error) {
	return _HopL1EthBridge.Contract.CHALLENGEAMOUNTDIVISOR(&_HopL1EthBridge.CallOpts)
}

// TIMESLOTSIZE is a free data retrieval call binding the contract method 0x4de8c6e6.
//
// Solidity: function TIME_SLOT_SIZE() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) TIMESLOTSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "TIME_SLOT_SIZE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TIMESLOTSIZE is a free data retrieval call binding the contract method 0x4de8c6e6.
//
// Solidity: function TIME_SLOT_SIZE() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) TIMESLOTSIZE() (*big.Int, error) {
	return _HopL1EthBridge.Contract.TIMESLOTSIZE(&_HopL1EthBridge.CallOpts)
}

// TIMESLOTSIZE is a free data retrieval call binding the contract method 0x4de8c6e6.
//
// Solidity: function TIME_SLOT_SIZE() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) TIMESLOTSIZE() (*big.Int, error) {
	return _HopL1EthBridge.Contract.TIMESLOTSIZE(&_HopL1EthBridge.CallOpts)
}

// ChainBalance is a free data retrieval call binding the contract method 0xfc110b67.
//
// Solidity: function chainBalance(uint256 ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) ChainBalance(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "chainBalance", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChainBalance is a free data retrieval call binding the contract method 0xfc110b67.
//
// Solidity: function chainBalance(uint256 ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) ChainBalance(arg0 *big.Int) (*big.Int, error) {
	return _HopL1EthBridge.Contract.ChainBalance(&_HopL1EthBridge.CallOpts, arg0)
}

// ChainBalance is a free data retrieval call binding the contract method 0xfc110b67.
//
// Solidity: function chainBalance(uint256 ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) ChainBalance(arg0 *big.Int) (*big.Int, error) {
	return _HopL1EthBridge.Contract.ChainBalance(&_HopL1EthBridge.CallOpts, arg0)
}

// ChallengePeriod is a free data retrieval call binding the contract method 0xf3f480d9.
//
// Solidity: function challengePeriod() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) ChallengePeriod(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "challengePeriod")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengePeriod is a free data retrieval call binding the contract method 0xf3f480d9.
//
// Solidity: function challengePeriod() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) ChallengePeriod() (*big.Int, error) {
	return _HopL1EthBridge.Contract.ChallengePeriod(&_HopL1EthBridge.CallOpts)
}

// ChallengePeriod is a free data retrieval call binding the contract method 0xf3f480d9.
//
// Solidity: function challengePeriod() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) ChallengePeriod() (*big.Int, error) {
	return _HopL1EthBridge.Contract.ChallengePeriod(&_HopL1EthBridge.CallOpts)
}

// ChallengeResolutionPeriod is a free data retrieval call binding the contract method 0x767631d5.
//
// Solidity: function challengeResolutionPeriod() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) ChallengeResolutionPeriod(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "challengeResolutionPeriod")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ChallengeResolutionPeriod is a free data retrieval call binding the contract method 0x767631d5.
//
// Solidity: function challengeResolutionPeriod() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) ChallengeResolutionPeriod() (*big.Int, error) {
	return _HopL1EthBridge.Contract.ChallengeResolutionPeriod(&_HopL1EthBridge.CallOpts)
}

// ChallengeResolutionPeriod is a free data retrieval call binding the contract method 0x767631d5.
//
// Solidity: function challengeResolutionPeriod() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) ChallengeResolutionPeriod() (*big.Int, error) {
	return _HopL1EthBridge.Contract.ChallengeResolutionPeriod(&_HopL1EthBridge.CallOpts)
}

// CrossDomainMessengerWrappers is a free data retrieval call binding the contract method 0xa35962f3.
//
// Solidity: function crossDomainMessengerWrappers(uint256 ) view returns(address)
func (_HopL1EthBridge *HopL1EthBridgeCaller) CrossDomainMessengerWrappers(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "crossDomainMessengerWrappers", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// CrossDomainMessengerWrappers is a free data retrieval call binding the contract method 0xa35962f3.
//
// Solidity: function crossDomainMessengerWrappers(uint256 ) view returns(address)
func (_HopL1EthBridge *HopL1EthBridgeSession) CrossDomainMessengerWrappers(arg0 *big.Int) (common.Address, error) {
	return _HopL1EthBridge.Contract.CrossDomainMessengerWrappers(&_HopL1EthBridge.CallOpts, arg0)
}

// CrossDomainMessengerWrappers is a free data retrieval call binding the contract method 0xa35962f3.
//
// Solidity: function crossDomainMessengerWrappers(uint256 ) view returns(address)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) CrossDomainMessengerWrappers(arg0 *big.Int) (common.Address, error) {
	return _HopL1EthBridge.Contract.CrossDomainMessengerWrappers(&_HopL1EthBridge.CallOpts, arg0)
}

// GetBondForTransferAmount is a free data retrieval call binding the contract method 0xe19be150.
//
// Solidity: function getBondForTransferAmount(uint256 amount) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetBondForTransferAmount(opts *bind.CallOpts, amount *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getBondForTransferAmount", amount)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBondForTransferAmount is a free data retrieval call binding the contract method 0xe19be150.
//
// Solidity: function getBondForTransferAmount(uint256 amount) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetBondForTransferAmount(amount *big.Int) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetBondForTransferAmount(&_HopL1EthBridge.CallOpts, amount)
}

// GetBondForTransferAmount is a free data retrieval call binding the contract method 0xe19be150.
//
// Solidity: function getBondForTransferAmount(uint256 amount) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetBondForTransferAmount(amount *big.Int) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetBondForTransferAmount(&_HopL1EthBridge.CallOpts, amount)
}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetBondedWithdrawalAmount(opts *bind.CallOpts, bonder common.Address, transferId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getBondedWithdrawalAmount", bonder, transferId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetBondedWithdrawalAmount(bonder common.Address, transferId [32]byte) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetBondedWithdrawalAmount(&_HopL1EthBridge.CallOpts, bonder, transferId)
}

// GetBondedWithdrawalAmount is a free data retrieval call binding the contract method 0x302830ab.
//
// Solidity: function getBondedWithdrawalAmount(address bonder, bytes32 transferId) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetBondedWithdrawalAmount(bonder common.Address, transferId [32]byte) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetBondedWithdrawalAmount(&_HopL1EthBridge.CallOpts, bonder, transferId)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetChainId(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getChainId")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetChainId() (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetChainId(&_HopL1EthBridge.CallOpts)
}

// GetChainId is a free data retrieval call binding the contract method 0x3408e470.
//
// Solidity: function getChainId() view returns(uint256 chainId)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetChainId() (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetChainId(&_HopL1EthBridge.CallOpts)
}

// GetChallengeAmountForTransferAmount is a free data retrieval call binding the contract method 0xa239f5ee.
//
// Solidity: function getChallengeAmountForTransferAmount(uint256 amount) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetChallengeAmountForTransferAmount(opts *bind.CallOpts, amount *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getChallengeAmountForTransferAmount", amount)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetChallengeAmountForTransferAmount is a free data retrieval call binding the contract method 0xa239f5ee.
//
// Solidity: function getChallengeAmountForTransferAmount(uint256 amount) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetChallengeAmountForTransferAmount(amount *big.Int) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetChallengeAmountForTransferAmount(&_HopL1EthBridge.CallOpts, amount)
}

// GetChallengeAmountForTransferAmount is a free data retrieval call binding the contract method 0xa239f5ee.
//
// Solidity: function getChallengeAmountForTransferAmount(uint256 amount) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetChallengeAmountForTransferAmount(amount *big.Int) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetChallengeAmountForTransferAmount(&_HopL1EthBridge.CallOpts, amount)
}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetCredit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getCredit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetCredit(bonder common.Address) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetCredit(&_HopL1EthBridge.CallOpts, bonder)
}

// GetCredit is a free data retrieval call binding the contract method 0x57344e6f.
//
// Solidity: function getCredit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetCredit(bonder common.Address) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetCredit(&_HopL1EthBridge.CallOpts, bonder)
}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetDebitAndAdditionalDebit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getDebitAndAdditionalDebit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetDebitAndAdditionalDebit(bonder common.Address) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetDebitAndAdditionalDebit(&_HopL1EthBridge.CallOpts, bonder)
}

// GetDebitAndAdditionalDebit is a free data retrieval call binding the contract method 0xffa9286c.
//
// Solidity: function getDebitAndAdditionalDebit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetDebitAndAdditionalDebit(bonder common.Address) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetDebitAndAdditionalDebit(&_HopL1EthBridge.CallOpts, bonder)
}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetIsBonder(opts *bind.CallOpts, maybeBonder common.Address) (bool, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getIsBonder", maybeBonder)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetIsBonder(maybeBonder common.Address) (bool, error) {
	return _HopL1EthBridge.Contract.GetIsBonder(&_HopL1EthBridge.CallOpts, maybeBonder)
}

// GetIsBonder is a free data retrieval call binding the contract method 0xd5ef7551.
//
// Solidity: function getIsBonder(address maybeBonder) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetIsBonder(maybeBonder common.Address) (bool, error) {
	return _HopL1EthBridge.Contract.GetIsBonder(&_HopL1EthBridge.CallOpts, maybeBonder)
}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetRawDebit(opts *bind.CallOpts, bonder common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getRawDebit", bonder)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetRawDebit(bonder common.Address) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetRawDebit(&_HopL1EthBridge.CallOpts, bonder)
}

// GetRawDebit is a free data retrieval call binding the contract method 0x13948c76.
//
// Solidity: function getRawDebit(address bonder) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetRawDebit(bonder common.Address) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetRawDebit(&_HopL1EthBridge.CallOpts, bonder)
}

// GetTimeSlot is a free data retrieval call binding the contract method 0x2b85dcc9.
//
// Solidity: function getTimeSlot(uint256 time) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetTimeSlot(opts *bind.CallOpts, time *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getTimeSlot", time)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTimeSlot is a free data retrieval call binding the contract method 0x2b85dcc9.
//
// Solidity: function getTimeSlot(uint256 time) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetTimeSlot(time *big.Int) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetTimeSlot(&_HopL1EthBridge.CallOpts, time)
}

// GetTimeSlot is a free data retrieval call binding the contract method 0x2b85dcc9.
//
// Solidity: function getTimeSlot(uint256 time) pure returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetTimeSlot(time *big.Int) (*big.Int, error) {
	return _HopL1EthBridge.Contract.GetTimeSlot(&_HopL1EthBridge.CallOpts, time)
}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetTransferId(opts *bind.CallOpts, chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getTransferId", chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetTransferId(chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	return _HopL1EthBridge.Contract.GetTransferId(&_HopL1EthBridge.CallOpts, chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// GetTransferId is a free data retrieval call binding the contract method 0xaf215f94.
//
// Solidity: function getTransferId(uint256 chainId, address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline) pure returns(bytes32)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetTransferId(chainId *big.Int, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int) ([32]byte, error) {
	return _HopL1EthBridge.Contract.GetTransferId(&_HopL1EthBridge.CallOpts, chainId, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline)
}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetTransferRoot(opts *bind.CallOpts, rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getTransferRoot", rootHash, totalAmount)

	if err != nil {
		return *new(BridgeTransferRoot), err
	}

	out0 := *abi.ConvertType(out[0], new(BridgeTransferRoot)).(*BridgeTransferRoot)

	return out0, err

}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL1EthBridge *HopL1EthBridgeSession) GetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	return _HopL1EthBridge.Contract.GetTransferRoot(&_HopL1EthBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRoot is a free data retrieval call binding the contract method 0xce803b4f.
//
// Solidity: function getTransferRoot(bytes32 rootHash, uint256 totalAmount) view returns((uint256,uint256,uint256))
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetTransferRoot(rootHash [32]byte, totalAmount *big.Int) (BridgeTransferRoot, error) {
	return _HopL1EthBridge.Contract.GetTransferRoot(&_HopL1EthBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL1EthBridge *HopL1EthBridgeCaller) GetTransferRootId(opts *bind.CallOpts, rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "getTransferRootId", rootHash, totalAmount)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL1EthBridge *HopL1EthBridgeSession) GetTransferRootId(rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	return _HopL1EthBridge.Contract.GetTransferRootId(&_HopL1EthBridge.CallOpts, rootHash, totalAmount)
}

// GetTransferRootId is a free data retrieval call binding the contract method 0x960a7afa.
//
// Solidity: function getTransferRootId(bytes32 rootHash, uint256 totalAmount) pure returns(bytes32)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) GetTransferRootId(rootHash [32]byte, totalAmount *big.Int) ([32]byte, error) {
	return _HopL1EthBridge.Contract.GetTransferRootId(&_HopL1EthBridge.CallOpts, rootHash, totalAmount)
}

// Governance is a free data retrieval call binding the contract method 0x5aa6e675.
//
// Solidity: function governance() view returns(address)
func (_HopL1EthBridge *HopL1EthBridgeCaller) Governance(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "governance")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Governance is a free data retrieval call binding the contract method 0x5aa6e675.
//
// Solidity: function governance() view returns(address)
func (_HopL1EthBridge *HopL1EthBridgeSession) Governance() (common.Address, error) {
	return _HopL1EthBridge.Contract.Governance(&_HopL1EthBridge.CallOpts)
}

// Governance is a free data retrieval call binding the contract method 0x5aa6e675.
//
// Solidity: function governance() view returns(address)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) Governance() (common.Address, error) {
	return _HopL1EthBridge.Contract.Governance(&_HopL1EthBridge.CallOpts)
}

// IsChainIdPaused is a free data retrieval call binding the contract method 0xfa2a69a3.
//
// Solidity: function isChainIdPaused(uint256 ) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeCaller) IsChainIdPaused(opts *bind.CallOpts, arg0 *big.Int) (bool, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "isChainIdPaused", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsChainIdPaused is a free data retrieval call binding the contract method 0xfa2a69a3.
//
// Solidity: function isChainIdPaused(uint256 ) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeSession) IsChainIdPaused(arg0 *big.Int) (bool, error) {
	return _HopL1EthBridge.Contract.IsChainIdPaused(&_HopL1EthBridge.CallOpts, arg0)
}

// IsChainIdPaused is a free data retrieval call binding the contract method 0xfa2a69a3.
//
// Solidity: function isChainIdPaused(uint256 ) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) IsChainIdPaused(arg0 *big.Int) (bool, error) {
	return _HopL1EthBridge.Contract.IsChainIdPaused(&_HopL1EthBridge.CallOpts, arg0)
}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeCaller) IsTransferIdSpent(opts *bind.CallOpts, transferId [32]byte) (bool, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "isTransferIdSpent", transferId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeSession) IsTransferIdSpent(transferId [32]byte) (bool, error) {
	return _HopL1EthBridge.Contract.IsTransferIdSpent(&_HopL1EthBridge.CallOpts, transferId)
}

// IsTransferIdSpent is a free data retrieval call binding the contract method 0x3a7af631.
//
// Solidity: function isTransferIdSpent(bytes32 transferId) view returns(bool)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) IsTransferIdSpent(transferId [32]byte) (bool, error) {
	return _HopL1EthBridge.Contract.IsTransferIdSpent(&_HopL1EthBridge.CallOpts, transferId)
}

// MinTransferRootBondDelay is a free data retrieval call binding the contract method 0x6cff06a7.
//
// Solidity: function minTransferRootBondDelay() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) MinTransferRootBondDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "minTransferRootBondDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinTransferRootBondDelay is a free data retrieval call binding the contract method 0x6cff06a7.
//
// Solidity: function minTransferRootBondDelay() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) MinTransferRootBondDelay() (*big.Int, error) {
	return _HopL1EthBridge.Contract.MinTransferRootBondDelay(&_HopL1EthBridge.CallOpts)
}

// MinTransferRootBondDelay is a free data retrieval call binding the contract method 0x6cff06a7.
//
// Solidity: function minTransferRootBondDelay() view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) MinTransferRootBondDelay() (*big.Int, error) {
	return _HopL1EthBridge.Contract.MinTransferRootBondDelay(&_HopL1EthBridge.CallOpts)
}

// TimeSlotToAmountBonded is a free data retrieval call binding the contract method 0x7398d282.
//
// Solidity: function timeSlotToAmountBonded(uint256 , address ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) TimeSlotToAmountBonded(opts *bind.CallOpts, arg0 *big.Int, arg1 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "timeSlotToAmountBonded", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TimeSlotToAmountBonded is a free data retrieval call binding the contract method 0x7398d282.
//
// Solidity: function timeSlotToAmountBonded(uint256 , address ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) TimeSlotToAmountBonded(arg0 *big.Int, arg1 common.Address) (*big.Int, error) {
	return _HopL1EthBridge.Contract.TimeSlotToAmountBonded(&_HopL1EthBridge.CallOpts, arg0, arg1)
}

// TimeSlotToAmountBonded is a free data retrieval call binding the contract method 0x7398d282.
//
// Solidity: function timeSlotToAmountBonded(uint256 , address ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) TimeSlotToAmountBonded(arg0 *big.Int, arg1 common.Address) (*big.Int, error) {
	return _HopL1EthBridge.Contract.TimeSlotToAmountBonded(&_HopL1EthBridge.CallOpts, arg0, arg1)
}

// TransferBonds is a free data retrieval call binding the contract method 0x5a7e1083.
//
// Solidity: function transferBonds(bytes32 ) view returns(address bonder, uint256 createdAt, uint256 totalAmount, uint256 challengeStartTime, address challenger, bool challengeResolved)
func (_HopL1EthBridge *HopL1EthBridgeCaller) TransferBonds(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Bonder             common.Address
	CreatedAt          *big.Int
	TotalAmount        *big.Int
	ChallengeStartTime *big.Int
	Challenger         common.Address
	ChallengeResolved  bool
}, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "transferBonds", arg0)

	outstruct := new(struct {
		Bonder             common.Address
		CreatedAt          *big.Int
		TotalAmount        *big.Int
		ChallengeStartTime *big.Int
		Challenger         common.Address
		ChallengeResolved  bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Bonder = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.CreatedAt = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalAmount = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.ChallengeStartTime = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.Challenger = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.ChallengeResolved = *abi.ConvertType(out[5], new(bool)).(*bool)

	return *outstruct, err

}

// TransferBonds is a free data retrieval call binding the contract method 0x5a7e1083.
//
// Solidity: function transferBonds(bytes32 ) view returns(address bonder, uint256 createdAt, uint256 totalAmount, uint256 challengeStartTime, address challenger, bool challengeResolved)
func (_HopL1EthBridge *HopL1EthBridgeSession) TransferBonds(arg0 [32]byte) (struct {
	Bonder             common.Address
	CreatedAt          *big.Int
	TotalAmount        *big.Int
	ChallengeStartTime *big.Int
	Challenger         common.Address
	ChallengeResolved  bool
}, error) {
	return _HopL1EthBridge.Contract.TransferBonds(&_HopL1EthBridge.CallOpts, arg0)
}

// TransferBonds is a free data retrieval call binding the contract method 0x5a7e1083.
//
// Solidity: function transferBonds(bytes32 ) view returns(address bonder, uint256 createdAt, uint256 totalAmount, uint256 challengeStartTime, address challenger, bool challengeResolved)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) TransferBonds(arg0 [32]byte) (struct {
	Bonder             common.Address
	CreatedAt          *big.Int
	TotalAmount        *big.Int
	ChallengeStartTime *big.Int
	Challenger         common.Address
	ChallengeResolved  bool
}, error) {
	return _HopL1EthBridge.Contract.TransferBonds(&_HopL1EthBridge.CallOpts, arg0)
}

// TransferRootCommittedAt is a free data retrieval call binding the contract method 0x3b8fea28.
//
// Solidity: function transferRootCommittedAt(uint256 , bytes32 ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCaller) TransferRootCommittedAt(opts *bind.CallOpts, arg0 *big.Int, arg1 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _HopL1EthBridge.contract.Call(opts, &out, "transferRootCommittedAt", arg0, arg1)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TransferRootCommittedAt is a free data retrieval call binding the contract method 0x3b8fea28.
//
// Solidity: function transferRootCommittedAt(uint256 , bytes32 ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeSession) TransferRootCommittedAt(arg0 *big.Int, arg1 [32]byte) (*big.Int, error) {
	return _HopL1EthBridge.Contract.TransferRootCommittedAt(&_HopL1EthBridge.CallOpts, arg0, arg1)
}

// TransferRootCommittedAt is a free data retrieval call binding the contract method 0x3b8fea28.
//
// Solidity: function transferRootCommittedAt(uint256 , bytes32 ) view returns(uint256)
func (_HopL1EthBridge *HopL1EthBridgeCallerSession) TransferRootCommittedAt(arg0 *big.Int, arg1 [32]byte) (*big.Int, error) {
	return _HopL1EthBridge.Contract.TransferRootCommittedAt(&_HopL1EthBridge.CallOpts, arg0, arg1)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) AddBonder(opts *bind.TransactOpts, bonder common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "addBonder", bonder)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) AddBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.AddBonder(&_HopL1EthBridge.TransactOpts, bonder)
}

// AddBonder is a paid mutator transaction binding the contract method 0x5325937f.
//
// Solidity: function addBonder(address bonder) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) AddBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.AddBonder(&_HopL1EthBridge.TransactOpts, bonder)
}

// BondTransferRoot is a paid mutator transaction binding the contract method 0x8d8798bf.
//
// Solidity: function bondTransferRoot(bytes32 rootHash, uint256 destinationChainId, uint256 totalAmount) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) BondTransferRoot(opts *bind.TransactOpts, rootHash [32]byte, destinationChainId *big.Int, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "bondTransferRoot", rootHash, destinationChainId, totalAmount)
}

// BondTransferRoot is a paid mutator transaction binding the contract method 0x8d8798bf.
//
// Solidity: function bondTransferRoot(bytes32 rootHash, uint256 destinationChainId, uint256 totalAmount) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) BondTransferRoot(rootHash [32]byte, destinationChainId *big.Int, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.BondTransferRoot(&_HopL1EthBridge.TransactOpts, rootHash, destinationChainId, totalAmount)
}

// BondTransferRoot is a paid mutator transaction binding the contract method 0x8d8798bf.
//
// Solidity: function bondTransferRoot(bytes32 rootHash, uint256 destinationChainId, uint256 totalAmount) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) BondTransferRoot(rootHash [32]byte, destinationChainId *big.Int, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.BondTransferRoot(&_HopL1EthBridge.TransactOpts, rootHash, destinationChainId, totalAmount)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) BondWithdrawal(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "bondWithdrawal", recipient, amount, transferNonce, bonderFee)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) BondWithdrawal(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.BondWithdrawal(&_HopL1EthBridge.TransactOpts, recipient, amount, transferNonce, bonderFee)
}

// BondWithdrawal is a paid mutator transaction binding the contract method 0x23c452cd.
//
// Solidity: function bondWithdrawal(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) BondWithdrawal(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.BondWithdrawal(&_HopL1EthBridge.TransactOpts, recipient, amount, transferNonce, bonderFee)
}

// ChallengeTransferBond is a paid mutator transaction binding the contract method 0x1bbe15ea.
//
// Solidity: function challengeTransferBond(bytes32 rootHash, uint256 originalAmount, uint256 destinationChainId) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) ChallengeTransferBond(opts *bind.TransactOpts, rootHash [32]byte, originalAmount *big.Int, destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "challengeTransferBond", rootHash, originalAmount, destinationChainId)
}

// ChallengeTransferBond is a paid mutator transaction binding the contract method 0x1bbe15ea.
//
// Solidity: function challengeTransferBond(bytes32 rootHash, uint256 originalAmount, uint256 destinationChainId) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) ChallengeTransferBond(rootHash [32]byte, originalAmount *big.Int, destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.ChallengeTransferBond(&_HopL1EthBridge.TransactOpts, rootHash, originalAmount, destinationChainId)
}

// ChallengeTransferBond is a paid mutator transaction binding the contract method 0x1bbe15ea.
//
// Solidity: function challengeTransferBond(bytes32 rootHash, uint256 originalAmount, uint256 destinationChainId) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) ChallengeTransferBond(rootHash [32]byte, originalAmount *big.Int, destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.ChallengeTransferBond(&_HopL1EthBridge.TransactOpts, rootHash, originalAmount, destinationChainId)
}

// ConfirmTransferRoot is a paid mutator transaction binding the contract method 0xef6ebe5e.
//
// Solidity: function confirmTransferRoot(uint256 originChainId, bytes32 rootHash, uint256 destinationChainId, uint256 totalAmount, uint256 rootCommittedAt) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) ConfirmTransferRoot(opts *bind.TransactOpts, originChainId *big.Int, rootHash [32]byte, destinationChainId *big.Int, totalAmount *big.Int, rootCommittedAt *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "confirmTransferRoot", originChainId, rootHash, destinationChainId, totalAmount, rootCommittedAt)
}

// ConfirmTransferRoot is a paid mutator transaction binding the contract method 0xef6ebe5e.
//
// Solidity: function confirmTransferRoot(uint256 originChainId, bytes32 rootHash, uint256 destinationChainId, uint256 totalAmount, uint256 rootCommittedAt) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) ConfirmTransferRoot(originChainId *big.Int, rootHash [32]byte, destinationChainId *big.Int, totalAmount *big.Int, rootCommittedAt *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.ConfirmTransferRoot(&_HopL1EthBridge.TransactOpts, originChainId, rootHash, destinationChainId, totalAmount, rootCommittedAt)
}

// ConfirmTransferRoot is a paid mutator transaction binding the contract method 0xef6ebe5e.
//
// Solidity: function confirmTransferRoot(uint256 originChainId, bytes32 rootHash, uint256 destinationChainId, uint256 totalAmount, uint256 rootCommittedAt) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) ConfirmTransferRoot(originChainId *big.Int, rootHash [32]byte, destinationChainId *big.Int, totalAmount *big.Int, rootCommittedAt *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.ConfirmTransferRoot(&_HopL1EthBridge.TransactOpts, originChainId, rootHash, destinationChainId, totalAmount, rootCommittedAt)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) RemoveBonder(opts *bind.TransactOpts, bonder common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "removeBonder", bonder)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) RemoveBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.RemoveBonder(&_HopL1EthBridge.TransactOpts, bonder)
}

// RemoveBonder is a paid mutator transaction binding the contract method 0x04e6c2c0.
//
// Solidity: function removeBonder(address bonder) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) RemoveBonder(bonder common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.RemoveBonder(&_HopL1EthBridge.TransactOpts, bonder)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) RescueTransferRoot(opts *bind.TransactOpts, rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "rescueTransferRoot", rootHash, originalAmount, recipient)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) RescueTransferRoot(rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.RescueTransferRoot(&_HopL1EthBridge.TransactOpts, rootHash, originalAmount, recipient)
}

// RescueTransferRoot is a paid mutator transaction binding the contract method 0xcbd1642e.
//
// Solidity: function rescueTransferRoot(bytes32 rootHash, uint256 originalAmount, address recipient) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) RescueTransferRoot(rootHash [32]byte, originalAmount *big.Int, recipient common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.RescueTransferRoot(&_HopL1EthBridge.TransactOpts, rootHash, originalAmount, recipient)
}

// ResolveChallenge is a paid mutator transaction binding the contract method 0x81707b80.
//
// Solidity: function resolveChallenge(bytes32 rootHash, uint256 originalAmount, uint256 destinationChainId) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) ResolveChallenge(opts *bind.TransactOpts, rootHash [32]byte, originalAmount *big.Int, destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "resolveChallenge", rootHash, originalAmount, destinationChainId)
}

// ResolveChallenge is a paid mutator transaction binding the contract method 0x81707b80.
//
// Solidity: function resolveChallenge(bytes32 rootHash, uint256 originalAmount, uint256 destinationChainId) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) ResolveChallenge(rootHash [32]byte, originalAmount *big.Int, destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.ResolveChallenge(&_HopL1EthBridge.TransactOpts, rootHash, originalAmount, destinationChainId)
}

// ResolveChallenge is a paid mutator transaction binding the contract method 0x81707b80.
//
// Solidity: function resolveChallenge(bytes32 rootHash, uint256 originalAmount, uint256 destinationChainId) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) ResolveChallenge(rootHash [32]byte, originalAmount *big.Int, destinationChainId *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.ResolveChallenge(&_HopL1EthBridge.TransactOpts, rootHash, originalAmount, destinationChainId)
}

// SendToL2 is a paid mutator transaction binding the contract method 0xdeace8f5.
//
// Solidity: function sendToL2(uint256 chainId, address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SendToL2(opts *bind.TransactOpts, chainId *big.Int, recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "sendToL2", chainId, recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// SendToL2 is a paid mutator transaction binding the contract method 0xdeace8f5.
//
// Solidity: function sendToL2(uint256 chainId, address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SendToL2(chainId *big.Int, recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SendToL2(&_HopL1EthBridge.TransactOpts, chainId, recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// SendToL2 is a paid mutator transaction binding the contract method 0xdeace8f5.
//
// Solidity: function sendToL2(uint256 chainId, address recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address relayer, uint256 relayerFee) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SendToL2(chainId *big.Int, recipient common.Address, amount *big.Int, amountOutMin *big.Int, deadline *big.Int, relayer common.Address, relayerFee *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SendToL2(&_HopL1EthBridge.TransactOpts, chainId, recipient, amount, amountOutMin, deadline, relayer, relayerFee)
}

// SetChainIdDepositsPaused is a paid mutator transaction binding the contract method 0x14942024.
//
// Solidity: function setChainIdDepositsPaused(uint256 chainId, bool isPaused) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SetChainIdDepositsPaused(opts *bind.TransactOpts, chainId *big.Int, isPaused bool) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "setChainIdDepositsPaused", chainId, isPaused)
}

// SetChainIdDepositsPaused is a paid mutator transaction binding the contract method 0x14942024.
//
// Solidity: function setChainIdDepositsPaused(uint256 chainId, bool isPaused) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SetChainIdDepositsPaused(chainId *big.Int, isPaused bool) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetChainIdDepositsPaused(&_HopL1EthBridge.TransactOpts, chainId, isPaused)
}

// SetChainIdDepositsPaused is a paid mutator transaction binding the contract method 0x14942024.
//
// Solidity: function setChainIdDepositsPaused(uint256 chainId, bool isPaused) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SetChainIdDepositsPaused(chainId *big.Int, isPaused bool) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetChainIdDepositsPaused(&_HopL1EthBridge.TransactOpts, chainId, isPaused)
}

// SetChallengePeriod is a paid mutator transaction binding the contract method 0x5d475fdd.
//
// Solidity: function setChallengePeriod(uint256 _challengePeriod) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SetChallengePeriod(opts *bind.TransactOpts, _challengePeriod *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "setChallengePeriod", _challengePeriod)
}

// SetChallengePeriod is a paid mutator transaction binding the contract method 0x5d475fdd.
//
// Solidity: function setChallengePeriod(uint256 _challengePeriod) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SetChallengePeriod(_challengePeriod *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetChallengePeriod(&_HopL1EthBridge.TransactOpts, _challengePeriod)
}

// SetChallengePeriod is a paid mutator transaction binding the contract method 0x5d475fdd.
//
// Solidity: function setChallengePeriod(uint256 _challengePeriod) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SetChallengePeriod(_challengePeriod *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetChallengePeriod(&_HopL1EthBridge.TransactOpts, _challengePeriod)
}

// SetChallengeResolutionPeriod is a paid mutator transaction binding the contract method 0xeecd57e6.
//
// Solidity: function setChallengeResolutionPeriod(uint256 _challengeResolutionPeriod) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SetChallengeResolutionPeriod(opts *bind.TransactOpts, _challengeResolutionPeriod *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "setChallengeResolutionPeriod", _challengeResolutionPeriod)
}

// SetChallengeResolutionPeriod is a paid mutator transaction binding the contract method 0xeecd57e6.
//
// Solidity: function setChallengeResolutionPeriod(uint256 _challengeResolutionPeriod) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SetChallengeResolutionPeriod(_challengeResolutionPeriod *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetChallengeResolutionPeriod(&_HopL1EthBridge.TransactOpts, _challengeResolutionPeriod)
}

// SetChallengeResolutionPeriod is a paid mutator transaction binding the contract method 0xeecd57e6.
//
// Solidity: function setChallengeResolutionPeriod(uint256 _challengeResolutionPeriod) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SetChallengeResolutionPeriod(_challengeResolutionPeriod *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetChallengeResolutionPeriod(&_HopL1EthBridge.TransactOpts, _challengeResolutionPeriod)
}

// SetCrossDomainMessengerWrapper is a paid mutator transaction binding the contract method 0xd4448163.
//
// Solidity: function setCrossDomainMessengerWrapper(uint256 chainId, address _crossDomainMessengerWrapper) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SetCrossDomainMessengerWrapper(opts *bind.TransactOpts, chainId *big.Int, _crossDomainMessengerWrapper common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "setCrossDomainMessengerWrapper", chainId, _crossDomainMessengerWrapper)
}

// SetCrossDomainMessengerWrapper is a paid mutator transaction binding the contract method 0xd4448163.
//
// Solidity: function setCrossDomainMessengerWrapper(uint256 chainId, address _crossDomainMessengerWrapper) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SetCrossDomainMessengerWrapper(chainId *big.Int, _crossDomainMessengerWrapper common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetCrossDomainMessengerWrapper(&_HopL1EthBridge.TransactOpts, chainId, _crossDomainMessengerWrapper)
}

// SetCrossDomainMessengerWrapper is a paid mutator transaction binding the contract method 0xd4448163.
//
// Solidity: function setCrossDomainMessengerWrapper(uint256 chainId, address _crossDomainMessengerWrapper) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SetCrossDomainMessengerWrapper(chainId *big.Int, _crossDomainMessengerWrapper common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetCrossDomainMessengerWrapper(&_HopL1EthBridge.TransactOpts, chainId, _crossDomainMessengerWrapper)
}

// SetGovernance is a paid mutator transaction binding the contract method 0xab033ea9.
//
// Solidity: function setGovernance(address _newGovernance) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SetGovernance(opts *bind.TransactOpts, _newGovernance common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "setGovernance", _newGovernance)
}

// SetGovernance is a paid mutator transaction binding the contract method 0xab033ea9.
//
// Solidity: function setGovernance(address _newGovernance) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SetGovernance(_newGovernance common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetGovernance(&_HopL1EthBridge.TransactOpts, _newGovernance)
}

// SetGovernance is a paid mutator transaction binding the contract method 0xab033ea9.
//
// Solidity: function setGovernance(address _newGovernance) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SetGovernance(_newGovernance common.Address) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetGovernance(&_HopL1EthBridge.TransactOpts, _newGovernance)
}

// SetMinTransferRootBondDelay is a paid mutator transaction binding the contract method 0x39ada669.
//
// Solidity: function setMinTransferRootBondDelay(uint256 _minTransferRootBondDelay) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SetMinTransferRootBondDelay(opts *bind.TransactOpts, _minTransferRootBondDelay *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "setMinTransferRootBondDelay", _minTransferRootBondDelay)
}

// SetMinTransferRootBondDelay is a paid mutator transaction binding the contract method 0x39ada669.
//
// Solidity: function setMinTransferRootBondDelay(uint256 _minTransferRootBondDelay) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SetMinTransferRootBondDelay(_minTransferRootBondDelay *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetMinTransferRootBondDelay(&_HopL1EthBridge.TransactOpts, _minTransferRootBondDelay)
}

// SetMinTransferRootBondDelay is a paid mutator transaction binding the contract method 0x39ada669.
//
// Solidity: function setMinTransferRootBondDelay(uint256 _minTransferRootBondDelay) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SetMinTransferRootBondDelay(_minTransferRootBondDelay *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SetMinTransferRootBondDelay(&_HopL1EthBridge.TransactOpts, _minTransferRootBondDelay)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SettleBondedWithdrawal(opts *bind.TransactOpts, bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "settleBondedWithdrawal", bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SettleBondedWithdrawal(bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SettleBondedWithdrawal(&_HopL1EthBridge.TransactOpts, bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawal is a paid mutator transaction binding the contract method 0xc7525dd3.
//
// Solidity: function settleBondedWithdrawal(address bonder, bytes32 transferId, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SettleBondedWithdrawal(bonder common.Address, transferId [32]byte, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SettleBondedWithdrawal(&_HopL1EthBridge.TransactOpts, bonder, transferId, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) SettleBondedWithdrawals(opts *bind.TransactOpts, bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "settleBondedWithdrawals", bonder, transferIds, totalAmount)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) SettleBondedWithdrawals(bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SettleBondedWithdrawals(&_HopL1EthBridge.TransactOpts, bonder, transferIds, totalAmount)
}

// SettleBondedWithdrawals is a paid mutator transaction binding the contract method 0xb162717e.
//
// Solidity: function settleBondedWithdrawals(address bonder, bytes32[] transferIds, uint256 totalAmount) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) SettleBondedWithdrawals(bonder common.Address, transferIds [][32]byte, totalAmount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.SettleBondedWithdrawals(&_HopL1EthBridge.TransactOpts, bonder, transferIds, totalAmount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) Stake(opts *bind.TransactOpts, bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "stake", bonder, amount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) Stake(bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.Stake(&_HopL1EthBridge.TransactOpts, bonder, amount)
}

// Stake is a paid mutator transaction binding the contract method 0xadc9772e.
//
// Solidity: function stake(address bonder, uint256 amount) payable returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) Stake(bonder common.Address, amount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.Stake(&_HopL1EthBridge.TransactOpts, bonder, amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) Unstake(opts *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "unstake", amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) Unstake(amount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.Unstake(&_HopL1EthBridge.TransactOpts, amount)
}

// Unstake is a paid mutator transaction binding the contract method 0x2e17de78.
//
// Solidity: function unstake(uint256 amount) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) Unstake(amount *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.Unstake(&_HopL1EthBridge.TransactOpts, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactor) Withdraw(opts *bind.TransactOpts, recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.contract.Transact(opts, "withdraw", recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL1EthBridge *HopL1EthBridgeSession) Withdraw(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.Withdraw(&_HopL1EthBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// Withdraw is a paid mutator transaction binding the contract method 0x0f7aadb7.
//
// Solidity: function withdraw(address recipient, uint256 amount, bytes32 transferNonce, uint256 bonderFee, uint256 amountOutMin, uint256 deadline, bytes32 rootHash, uint256 transferRootTotalAmount, uint256 transferIdTreeIndex, bytes32[] siblings, uint256 totalLeaves) returns()
func (_HopL1EthBridge *HopL1EthBridgeTransactorSession) Withdraw(recipient common.Address, amount *big.Int, transferNonce [32]byte, bonderFee *big.Int, amountOutMin *big.Int, deadline *big.Int, rootHash [32]byte, transferRootTotalAmount *big.Int, transferIdTreeIndex *big.Int, siblings [][32]byte, totalLeaves *big.Int) (*types.Transaction, error) {
	return _HopL1EthBridge.Contract.Withdraw(&_HopL1EthBridge.TransactOpts, recipient, amount, transferNonce, bonderFee, amountOutMin, deadline, rootHash, transferRootTotalAmount, transferIdTreeIndex, siblings, totalLeaves)
}

// HopL1EthBridgeBonderAddedIterator is returned from FilterBonderAdded and is used to iterate over the raw logs and unpacked data for BonderAdded events raised by the HopL1EthBridge contract.
type HopL1EthBridgeBonderAddedIterator struct {
	Event *HopL1EthBridgeBonderAdded // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeBonderAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeBonderAdded)
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
		it.Event = new(HopL1EthBridgeBonderAdded)
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
func (it *HopL1EthBridgeBonderAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeBonderAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeBonderAdded represents a BonderAdded event raised by the HopL1EthBridge contract.
type HopL1EthBridgeBonderAdded struct {
	NewBonder common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterBonderAdded is a free log retrieval operation binding the contract event 0x2cec73b7434d3b91198ad1a618f63e6a0761ce281af5ec9ec76606d948d03e23.
//
// Solidity: event BonderAdded(address indexed newBonder)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterBonderAdded(opts *bind.FilterOpts, newBonder []common.Address) (*HopL1EthBridgeBonderAddedIterator, error) {

	var newBonderRule []interface{}
	for _, newBonderItem := range newBonder {
		newBonderRule = append(newBonderRule, newBonderItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "BonderAdded", newBonderRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeBonderAddedIterator{contract: _HopL1EthBridge.contract, event: "BonderAdded", logs: logs, sub: sub}, nil
}

// WatchBonderAdded is a free log subscription operation binding the contract event 0x2cec73b7434d3b91198ad1a618f63e6a0761ce281af5ec9ec76606d948d03e23.
//
// Solidity: event BonderAdded(address indexed newBonder)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchBonderAdded(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeBonderAdded, newBonder []common.Address) (event.Subscription, error) {

	var newBonderRule []interface{}
	for _, newBonderItem := range newBonder {
		newBonderRule = append(newBonderRule, newBonderItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "BonderAdded", newBonderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeBonderAdded)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "BonderAdded", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseBonderAdded(log types.Log) (*HopL1EthBridgeBonderAdded, error) {
	event := new(HopL1EthBridgeBonderAdded)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "BonderAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeBonderRemovedIterator is returned from FilterBonderRemoved and is used to iterate over the raw logs and unpacked data for BonderRemoved events raised by the HopL1EthBridge contract.
type HopL1EthBridgeBonderRemovedIterator struct {
	Event *HopL1EthBridgeBonderRemoved // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeBonderRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeBonderRemoved)
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
		it.Event = new(HopL1EthBridgeBonderRemoved)
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
func (it *HopL1EthBridgeBonderRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeBonderRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeBonderRemoved represents a BonderRemoved event raised by the HopL1EthBridge contract.
type HopL1EthBridgeBonderRemoved struct {
	PreviousBonder common.Address
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterBonderRemoved is a free log retrieval operation binding the contract event 0x4234ba611d325b3ba434c4e1b037967b955b1274d4185ee9847b7491111a48ff.
//
// Solidity: event BonderRemoved(address indexed previousBonder)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterBonderRemoved(opts *bind.FilterOpts, previousBonder []common.Address) (*HopL1EthBridgeBonderRemovedIterator, error) {

	var previousBonderRule []interface{}
	for _, previousBonderItem := range previousBonder {
		previousBonderRule = append(previousBonderRule, previousBonderItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "BonderRemoved", previousBonderRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeBonderRemovedIterator{contract: _HopL1EthBridge.contract, event: "BonderRemoved", logs: logs, sub: sub}, nil
}

// WatchBonderRemoved is a free log subscription operation binding the contract event 0x4234ba611d325b3ba434c4e1b037967b955b1274d4185ee9847b7491111a48ff.
//
// Solidity: event BonderRemoved(address indexed previousBonder)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchBonderRemoved(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeBonderRemoved, previousBonder []common.Address) (event.Subscription, error) {

	var previousBonderRule []interface{}
	for _, previousBonderItem := range previousBonder {
		previousBonderRule = append(previousBonderRule, previousBonderItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "BonderRemoved", previousBonderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeBonderRemoved)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "BonderRemoved", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseBonderRemoved(log types.Log) (*HopL1EthBridgeBonderRemoved, error) {
	event := new(HopL1EthBridgeBonderRemoved)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "BonderRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeChallengeResolvedIterator is returned from FilterChallengeResolved and is used to iterate over the raw logs and unpacked data for ChallengeResolved events raised by the HopL1EthBridge contract.
type HopL1EthBridgeChallengeResolvedIterator struct {
	Event *HopL1EthBridgeChallengeResolved // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeChallengeResolvedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeChallengeResolved)
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
		it.Event = new(HopL1EthBridgeChallengeResolved)
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
func (it *HopL1EthBridgeChallengeResolvedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeChallengeResolvedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeChallengeResolved represents a ChallengeResolved event raised by the HopL1EthBridge contract.
type HopL1EthBridgeChallengeResolved struct {
	TransferRootId [32]byte
	RootHash       [32]byte
	OriginalAmount *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterChallengeResolved is a free log retrieval operation binding the contract event 0x4a99228a8a6d774d261be57ab0ed833bb1bae1f22bbbd3d4767b75ad03fdddf7.
//
// Solidity: event ChallengeResolved(bytes32 indexed transferRootId, bytes32 indexed rootHash, uint256 originalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterChallengeResolved(opts *bind.FilterOpts, transferRootId [][32]byte, rootHash [][32]byte) (*HopL1EthBridgeChallengeResolvedIterator, error) {

	var transferRootIdRule []interface{}
	for _, transferRootIdItem := range transferRootId {
		transferRootIdRule = append(transferRootIdRule, transferRootIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "ChallengeResolved", transferRootIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeChallengeResolvedIterator{contract: _HopL1EthBridge.contract, event: "ChallengeResolved", logs: logs, sub: sub}, nil
}

// WatchChallengeResolved is a free log subscription operation binding the contract event 0x4a99228a8a6d774d261be57ab0ed833bb1bae1f22bbbd3d4767b75ad03fdddf7.
//
// Solidity: event ChallengeResolved(bytes32 indexed transferRootId, bytes32 indexed rootHash, uint256 originalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchChallengeResolved(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeChallengeResolved, transferRootId [][32]byte, rootHash [][32]byte) (event.Subscription, error) {

	var transferRootIdRule []interface{}
	for _, transferRootIdItem := range transferRootId {
		transferRootIdRule = append(transferRootIdRule, transferRootIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "ChallengeResolved", transferRootIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeChallengeResolved)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "ChallengeResolved", log); err != nil {
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

// ParseChallengeResolved is a log parse operation binding the contract event 0x4a99228a8a6d774d261be57ab0ed833bb1bae1f22bbbd3d4767b75ad03fdddf7.
//
// Solidity: event ChallengeResolved(bytes32 indexed transferRootId, bytes32 indexed rootHash, uint256 originalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseChallengeResolved(log types.Log) (*HopL1EthBridgeChallengeResolved, error) {
	event := new(HopL1EthBridgeChallengeResolved)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "ChallengeResolved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeMultipleWithdrawalsSettledIterator is returned from FilterMultipleWithdrawalsSettled and is used to iterate over the raw logs and unpacked data for MultipleWithdrawalsSettled events raised by the HopL1EthBridge contract.
type HopL1EthBridgeMultipleWithdrawalsSettledIterator struct {
	Event *HopL1EthBridgeMultipleWithdrawalsSettled // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeMultipleWithdrawalsSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeMultipleWithdrawalsSettled)
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
		it.Event = new(HopL1EthBridgeMultipleWithdrawalsSettled)
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
func (it *HopL1EthBridgeMultipleWithdrawalsSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeMultipleWithdrawalsSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeMultipleWithdrawalsSettled represents a MultipleWithdrawalsSettled event raised by the HopL1EthBridge contract.
type HopL1EthBridgeMultipleWithdrawalsSettled struct {
	Bonder            common.Address
	RootHash          [32]byte
	TotalBondsSettled *big.Int
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterMultipleWithdrawalsSettled is a free log retrieval operation binding the contract event 0x78e830d08be9d5f957414c84d685c061ecbd8467be98b42ebb64f0118b57d2ff.
//
// Solidity: event MultipleWithdrawalsSettled(address indexed bonder, bytes32 indexed rootHash, uint256 totalBondsSettled)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterMultipleWithdrawalsSettled(opts *bind.FilterOpts, bonder []common.Address, rootHash [][32]byte) (*HopL1EthBridgeMultipleWithdrawalsSettledIterator, error) {

	var bonderRule []interface{}
	for _, bonderItem := range bonder {
		bonderRule = append(bonderRule, bonderItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "MultipleWithdrawalsSettled", bonderRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeMultipleWithdrawalsSettledIterator{contract: _HopL1EthBridge.contract, event: "MultipleWithdrawalsSettled", logs: logs, sub: sub}, nil
}

// WatchMultipleWithdrawalsSettled is a free log subscription operation binding the contract event 0x78e830d08be9d5f957414c84d685c061ecbd8467be98b42ebb64f0118b57d2ff.
//
// Solidity: event MultipleWithdrawalsSettled(address indexed bonder, bytes32 indexed rootHash, uint256 totalBondsSettled)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchMultipleWithdrawalsSettled(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeMultipleWithdrawalsSettled, bonder []common.Address, rootHash [][32]byte) (event.Subscription, error) {

	var bonderRule []interface{}
	for _, bonderItem := range bonder {
		bonderRule = append(bonderRule, bonderItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "MultipleWithdrawalsSettled", bonderRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeMultipleWithdrawalsSettled)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "MultipleWithdrawalsSettled", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseMultipleWithdrawalsSettled(log types.Log) (*HopL1EthBridgeMultipleWithdrawalsSettled, error) {
	event := new(HopL1EthBridgeMultipleWithdrawalsSettled)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "MultipleWithdrawalsSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeStakeIterator is returned from FilterStake and is used to iterate over the raw logs and unpacked data for Stake events raised by the HopL1EthBridge contract.
type HopL1EthBridgeStakeIterator struct {
	Event *HopL1EthBridgeStake // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeStakeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeStake)
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
		it.Event = new(HopL1EthBridgeStake)
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
func (it *HopL1EthBridgeStakeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeStakeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeStake represents a Stake event raised by the HopL1EthBridge contract.
type HopL1EthBridgeStake struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterStake is a free log retrieval operation binding the contract event 0xebedb8b3c678666e7f36970bc8f57abf6d8fa2e828c0da91ea5b75bf68ed101a.
//
// Solidity: event Stake(address indexed account, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterStake(opts *bind.FilterOpts, account []common.Address) (*HopL1EthBridgeStakeIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "Stake", accountRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeStakeIterator{contract: _HopL1EthBridge.contract, event: "Stake", logs: logs, sub: sub}, nil
}

// WatchStake is a free log subscription operation binding the contract event 0xebedb8b3c678666e7f36970bc8f57abf6d8fa2e828c0da91ea5b75bf68ed101a.
//
// Solidity: event Stake(address indexed account, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchStake(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeStake, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "Stake", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeStake)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "Stake", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseStake(log types.Log) (*HopL1EthBridgeStake, error) {
	event := new(HopL1EthBridgeStake)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "Stake", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeTransferBondChallengedIterator is returned from FilterTransferBondChallenged and is used to iterate over the raw logs and unpacked data for TransferBondChallenged events raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferBondChallengedIterator struct {
	Event *HopL1EthBridgeTransferBondChallenged // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeTransferBondChallengedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeTransferBondChallenged)
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
		it.Event = new(HopL1EthBridgeTransferBondChallenged)
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
func (it *HopL1EthBridgeTransferBondChallengedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeTransferBondChallengedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeTransferBondChallenged represents a TransferBondChallenged event raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferBondChallenged struct {
	TransferRootId [32]byte
	RootHash       [32]byte
	OriginalAmount *big.Int
	Raw            types.Log // Blockchain specific contextual infos
}

// FilterTransferBondChallenged is a free log retrieval operation binding the contract event 0xec2697dcba539a0ac947cdf1f6d0b6314c065429eca8be2435859b10209d4c27.
//
// Solidity: event TransferBondChallenged(bytes32 indexed transferRootId, bytes32 indexed rootHash, uint256 originalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterTransferBondChallenged(opts *bind.FilterOpts, transferRootId [][32]byte, rootHash [][32]byte) (*HopL1EthBridgeTransferBondChallengedIterator, error) {

	var transferRootIdRule []interface{}
	for _, transferRootIdItem := range transferRootId {
		transferRootIdRule = append(transferRootIdRule, transferRootIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "TransferBondChallenged", transferRootIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeTransferBondChallengedIterator{contract: _HopL1EthBridge.contract, event: "TransferBondChallenged", logs: logs, sub: sub}, nil
}

// WatchTransferBondChallenged is a free log subscription operation binding the contract event 0xec2697dcba539a0ac947cdf1f6d0b6314c065429eca8be2435859b10209d4c27.
//
// Solidity: event TransferBondChallenged(bytes32 indexed transferRootId, bytes32 indexed rootHash, uint256 originalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchTransferBondChallenged(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeTransferBondChallenged, transferRootId [][32]byte, rootHash [][32]byte) (event.Subscription, error) {

	var transferRootIdRule []interface{}
	for _, transferRootIdItem := range transferRootId {
		transferRootIdRule = append(transferRootIdRule, transferRootIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "TransferBondChallenged", transferRootIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeTransferBondChallenged)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferBondChallenged", log); err != nil {
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

// ParseTransferBondChallenged is a log parse operation binding the contract event 0xec2697dcba539a0ac947cdf1f6d0b6314c065429eca8be2435859b10209d4c27.
//
// Solidity: event TransferBondChallenged(bytes32 indexed transferRootId, bytes32 indexed rootHash, uint256 originalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseTransferBondChallenged(log types.Log) (*HopL1EthBridgeTransferBondChallenged, error) {
	event := new(HopL1EthBridgeTransferBondChallenged)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferBondChallenged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeTransferRootBondedIterator is returned from FilterTransferRootBonded and is used to iterate over the raw logs and unpacked data for TransferRootBonded events raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferRootBondedIterator struct {
	Event *HopL1EthBridgeTransferRootBonded // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeTransferRootBondedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeTransferRootBonded)
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
		it.Event = new(HopL1EthBridgeTransferRootBonded)
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
func (it *HopL1EthBridgeTransferRootBondedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeTransferRootBondedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeTransferRootBonded represents a TransferRootBonded event raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferRootBonded struct {
	Root   [32]byte
	Amount *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTransferRootBonded is a free log retrieval operation binding the contract event 0xa57b3e1f3af9eca02201028629700658608222c365064584cfe65d9630ef4f7b.
//
// Solidity: event TransferRootBonded(bytes32 indexed root, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterTransferRootBonded(opts *bind.FilterOpts, root [][32]byte) (*HopL1EthBridgeTransferRootBondedIterator, error) {

	var rootRule []interface{}
	for _, rootItem := range root {
		rootRule = append(rootRule, rootItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "TransferRootBonded", rootRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeTransferRootBondedIterator{contract: _HopL1EthBridge.contract, event: "TransferRootBonded", logs: logs, sub: sub}, nil
}

// WatchTransferRootBonded is a free log subscription operation binding the contract event 0xa57b3e1f3af9eca02201028629700658608222c365064584cfe65d9630ef4f7b.
//
// Solidity: event TransferRootBonded(bytes32 indexed root, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchTransferRootBonded(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeTransferRootBonded, root [][32]byte) (event.Subscription, error) {

	var rootRule []interface{}
	for _, rootItem := range root {
		rootRule = append(rootRule, rootItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "TransferRootBonded", rootRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeTransferRootBonded)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferRootBonded", log); err != nil {
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

// ParseTransferRootBonded is a log parse operation binding the contract event 0xa57b3e1f3af9eca02201028629700658608222c365064584cfe65d9630ef4f7b.
//
// Solidity: event TransferRootBonded(bytes32 indexed root, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseTransferRootBonded(log types.Log) (*HopL1EthBridgeTransferRootBonded, error) {
	event := new(HopL1EthBridgeTransferRootBonded)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferRootBonded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeTransferRootConfirmedIterator is returned from FilterTransferRootConfirmed and is used to iterate over the raw logs and unpacked data for TransferRootConfirmed events raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferRootConfirmedIterator struct {
	Event *HopL1EthBridgeTransferRootConfirmed // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeTransferRootConfirmedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeTransferRootConfirmed)
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
		it.Event = new(HopL1EthBridgeTransferRootConfirmed)
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
func (it *HopL1EthBridgeTransferRootConfirmedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeTransferRootConfirmedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeTransferRootConfirmed represents a TransferRootConfirmed event raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferRootConfirmed struct {
	OriginChainId      *big.Int
	DestinationChainId *big.Int
	RootHash           [32]byte
	TotalAmount        *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterTransferRootConfirmed is a free log retrieval operation binding the contract event 0xfdfb0eefa96935b8a8c0edf528e125dc6f3934fdbbfce31b38967e8ff413dccd.
//
// Solidity: event TransferRootConfirmed(uint256 indexed originChainId, uint256 indexed destinationChainId, bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterTransferRootConfirmed(opts *bind.FilterOpts, originChainId []*big.Int, destinationChainId []*big.Int, rootHash [][32]byte) (*HopL1EthBridgeTransferRootConfirmedIterator, error) {

	var originChainIdRule []interface{}
	for _, originChainIdItem := range originChainId {
		originChainIdRule = append(originChainIdRule, originChainIdItem)
	}
	var destinationChainIdRule []interface{}
	for _, destinationChainIdItem := range destinationChainId {
		destinationChainIdRule = append(destinationChainIdRule, destinationChainIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "TransferRootConfirmed", originChainIdRule, destinationChainIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeTransferRootConfirmedIterator{contract: _HopL1EthBridge.contract, event: "TransferRootConfirmed", logs: logs, sub: sub}, nil
}

// WatchTransferRootConfirmed is a free log subscription operation binding the contract event 0xfdfb0eefa96935b8a8c0edf528e125dc6f3934fdbbfce31b38967e8ff413dccd.
//
// Solidity: event TransferRootConfirmed(uint256 indexed originChainId, uint256 indexed destinationChainId, bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchTransferRootConfirmed(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeTransferRootConfirmed, originChainId []*big.Int, destinationChainId []*big.Int, rootHash [][32]byte) (event.Subscription, error) {

	var originChainIdRule []interface{}
	for _, originChainIdItem := range originChainId {
		originChainIdRule = append(originChainIdRule, originChainIdItem)
	}
	var destinationChainIdRule []interface{}
	for _, destinationChainIdItem := range destinationChainId {
		destinationChainIdRule = append(destinationChainIdRule, destinationChainIdItem)
	}
	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "TransferRootConfirmed", originChainIdRule, destinationChainIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeTransferRootConfirmed)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferRootConfirmed", log); err != nil {
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

// ParseTransferRootConfirmed is a log parse operation binding the contract event 0xfdfb0eefa96935b8a8c0edf528e125dc6f3934fdbbfce31b38967e8ff413dccd.
//
// Solidity: event TransferRootConfirmed(uint256 indexed originChainId, uint256 indexed destinationChainId, bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseTransferRootConfirmed(log types.Log) (*HopL1EthBridgeTransferRootConfirmed, error) {
	event := new(HopL1EthBridgeTransferRootConfirmed)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferRootConfirmed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeTransferRootSetIterator is returned from FilterTransferRootSet and is used to iterate over the raw logs and unpacked data for TransferRootSet events raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferRootSetIterator struct {
	Event *HopL1EthBridgeTransferRootSet // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeTransferRootSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeTransferRootSet)
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
		it.Event = new(HopL1EthBridgeTransferRootSet)
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
func (it *HopL1EthBridgeTransferRootSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeTransferRootSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeTransferRootSet represents a TransferRootSet event raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferRootSet struct {
	RootHash    [32]byte
	TotalAmount *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterTransferRootSet is a free log retrieval operation binding the contract event 0xb33d2162aead99dab59e77a7a67ea025b776bf8ca8079e132afdf9b23e03bd42.
//
// Solidity: event TransferRootSet(bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterTransferRootSet(opts *bind.FilterOpts, rootHash [][32]byte) (*HopL1EthBridgeTransferRootSetIterator, error) {

	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "TransferRootSet", rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeTransferRootSetIterator{contract: _HopL1EthBridge.contract, event: "TransferRootSet", logs: logs, sub: sub}, nil
}

// WatchTransferRootSet is a free log subscription operation binding the contract event 0xb33d2162aead99dab59e77a7a67ea025b776bf8ca8079e132afdf9b23e03bd42.
//
// Solidity: event TransferRootSet(bytes32 indexed rootHash, uint256 totalAmount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchTransferRootSet(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeTransferRootSet, rootHash [][32]byte) (event.Subscription, error) {

	var rootHashRule []interface{}
	for _, rootHashItem := range rootHash {
		rootHashRule = append(rootHashRule, rootHashItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "TransferRootSet", rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeTransferRootSet)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferRootSet", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseTransferRootSet(log types.Log) (*HopL1EthBridgeTransferRootSet, error) {
	event := new(HopL1EthBridgeTransferRootSet)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferRootSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeTransferSentToL2Iterator is returned from FilterTransferSentToL2 and is used to iterate over the raw logs and unpacked data for TransferSentToL2 events raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferSentToL2Iterator struct {
	Event *HopL1EthBridgeTransferSentToL2 // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeTransferSentToL2Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeTransferSentToL2)
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
		it.Event = new(HopL1EthBridgeTransferSentToL2)
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
func (it *HopL1EthBridgeTransferSentToL2Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeTransferSentToL2Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeTransferSentToL2 represents a TransferSentToL2 event raised by the HopL1EthBridge contract.
type HopL1EthBridgeTransferSentToL2 struct {
	ChainId      *big.Int
	Recipient    common.Address
	Amount       *big.Int
	AmountOutMin *big.Int
	Deadline     *big.Int
	Relayer      common.Address
	RelayerFee   *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterTransferSentToL2 is a free log retrieval operation binding the contract event 0x0a0607688c86ec1775abcdbab7b33a3a35a6c9cde677c9be880150c231cc6b0b.
//
// Solidity: event TransferSentToL2(uint256 indexed chainId, address indexed recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address indexed relayer, uint256 relayerFee)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterTransferSentToL2(opts *bind.FilterOpts, chainId []*big.Int, recipient []common.Address, relayer []common.Address) (*HopL1EthBridgeTransferSentToL2Iterator, error) {

	var chainIdRule []interface{}
	for _, chainIdItem := range chainId {
		chainIdRule = append(chainIdRule, chainIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	var relayerRule []interface{}
	for _, relayerItem := range relayer {
		relayerRule = append(relayerRule, relayerItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "TransferSentToL2", chainIdRule, recipientRule, relayerRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeTransferSentToL2Iterator{contract: _HopL1EthBridge.contract, event: "TransferSentToL2", logs: logs, sub: sub}, nil
}

// WatchTransferSentToL2 is a free log subscription operation binding the contract event 0x0a0607688c86ec1775abcdbab7b33a3a35a6c9cde677c9be880150c231cc6b0b.
//
// Solidity: event TransferSentToL2(uint256 indexed chainId, address indexed recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address indexed relayer, uint256 relayerFee)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchTransferSentToL2(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeTransferSentToL2, chainId []*big.Int, recipient []common.Address, relayer []common.Address) (event.Subscription, error) {

	var chainIdRule []interface{}
	for _, chainIdItem := range chainId {
		chainIdRule = append(chainIdRule, chainIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	var relayerRule []interface{}
	for _, relayerItem := range relayer {
		relayerRule = append(relayerRule, relayerItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "TransferSentToL2", chainIdRule, recipientRule, relayerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeTransferSentToL2)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferSentToL2", log); err != nil {
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

// ParseTransferSentToL2 is a log parse operation binding the contract event 0x0a0607688c86ec1775abcdbab7b33a3a35a6c9cde677c9be880150c231cc6b0b.
//
// Solidity: event TransferSentToL2(uint256 indexed chainId, address indexed recipient, uint256 amount, uint256 amountOutMin, uint256 deadline, address indexed relayer, uint256 relayerFee)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseTransferSentToL2(log types.Log) (*HopL1EthBridgeTransferSentToL2, error) {
	event := new(HopL1EthBridgeTransferSentToL2)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "TransferSentToL2", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeUnstakeIterator is returned from FilterUnstake and is used to iterate over the raw logs and unpacked data for Unstake events raised by the HopL1EthBridge contract.
type HopL1EthBridgeUnstakeIterator struct {
	Event *HopL1EthBridgeUnstake // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeUnstakeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeUnstake)
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
		it.Event = new(HopL1EthBridgeUnstake)
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
func (it *HopL1EthBridgeUnstakeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeUnstakeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeUnstake represents a Unstake event raised by the HopL1EthBridge contract.
type HopL1EthBridgeUnstake struct {
	Account common.Address
	Amount  *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterUnstake is a free log retrieval operation binding the contract event 0x85082129d87b2fe11527cb1b3b7a520aeb5aa6913f88a3d8757fe40d1db02fdd.
//
// Solidity: event Unstake(address indexed account, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterUnstake(opts *bind.FilterOpts, account []common.Address) (*HopL1EthBridgeUnstakeIterator, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "Unstake", accountRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeUnstakeIterator{contract: _HopL1EthBridge.contract, event: "Unstake", logs: logs, sub: sub}, nil
}

// WatchUnstake is a free log subscription operation binding the contract event 0x85082129d87b2fe11527cb1b3b7a520aeb5aa6913f88a3d8757fe40d1db02fdd.
//
// Solidity: event Unstake(address indexed account, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchUnstake(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeUnstake, account []common.Address) (event.Subscription, error) {

	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "Unstake", accountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeUnstake)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "Unstake", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseUnstake(log types.Log) (*HopL1EthBridgeUnstake, error) {
	event := new(HopL1EthBridgeUnstake)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "Unstake", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeWithdrawalBondSettledIterator is returned from FilterWithdrawalBondSettled and is used to iterate over the raw logs and unpacked data for WithdrawalBondSettled events raised by the HopL1EthBridge contract.
type HopL1EthBridgeWithdrawalBondSettledIterator struct {
	Event *HopL1EthBridgeWithdrawalBondSettled // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeWithdrawalBondSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeWithdrawalBondSettled)
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
		it.Event = new(HopL1EthBridgeWithdrawalBondSettled)
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
func (it *HopL1EthBridgeWithdrawalBondSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeWithdrawalBondSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeWithdrawalBondSettled represents a WithdrawalBondSettled event raised by the HopL1EthBridge contract.
type HopL1EthBridgeWithdrawalBondSettled struct {
	Bonder     common.Address
	TransferId [32]byte
	RootHash   [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalBondSettled is a free log retrieval operation binding the contract event 0x84eb21b24c31b27a3bc67dde4a598aad06db6e9415cd66544492b9616996143c.
//
// Solidity: event WithdrawalBondSettled(address indexed bonder, bytes32 indexed transferId, bytes32 indexed rootHash)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterWithdrawalBondSettled(opts *bind.FilterOpts, bonder []common.Address, transferId [][32]byte, rootHash [][32]byte) (*HopL1EthBridgeWithdrawalBondSettledIterator, error) {

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

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "WithdrawalBondSettled", bonderRule, transferIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeWithdrawalBondSettledIterator{contract: _HopL1EthBridge.contract, event: "WithdrawalBondSettled", logs: logs, sub: sub}, nil
}

// WatchWithdrawalBondSettled is a free log subscription operation binding the contract event 0x84eb21b24c31b27a3bc67dde4a598aad06db6e9415cd66544492b9616996143c.
//
// Solidity: event WithdrawalBondSettled(address indexed bonder, bytes32 indexed transferId, bytes32 indexed rootHash)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchWithdrawalBondSettled(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeWithdrawalBondSettled, bonder []common.Address, transferId [][32]byte, rootHash [][32]byte) (event.Subscription, error) {

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

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "WithdrawalBondSettled", bonderRule, transferIdRule, rootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeWithdrawalBondSettled)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "WithdrawalBondSettled", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseWithdrawalBondSettled(log types.Log) (*HopL1EthBridgeWithdrawalBondSettled, error) {
	event := new(HopL1EthBridgeWithdrawalBondSettled)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "WithdrawalBondSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeWithdrawalBondedIterator is returned from FilterWithdrawalBonded and is used to iterate over the raw logs and unpacked data for WithdrawalBonded events raised by the HopL1EthBridge contract.
type HopL1EthBridgeWithdrawalBondedIterator struct {
	Event *HopL1EthBridgeWithdrawalBonded // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeWithdrawalBondedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeWithdrawalBonded)
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
		it.Event = new(HopL1EthBridgeWithdrawalBonded)
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
func (it *HopL1EthBridgeWithdrawalBondedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeWithdrawalBondedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeWithdrawalBonded represents a WithdrawalBonded event raised by the HopL1EthBridge contract.
type HopL1EthBridgeWithdrawalBonded struct {
	TransferId [32]byte
	Amount     *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterWithdrawalBonded is a free log retrieval operation binding the contract event 0x0c3d250c7831051e78aa6a56679e590374c7c424415ffe4aa474491def2fe705.
//
// Solidity: event WithdrawalBonded(bytes32 indexed transferId, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterWithdrawalBonded(opts *bind.FilterOpts, transferId [][32]byte) (*HopL1EthBridgeWithdrawalBondedIterator, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "WithdrawalBonded", transferIdRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeWithdrawalBondedIterator{contract: _HopL1EthBridge.contract, event: "WithdrawalBonded", logs: logs, sub: sub}, nil
}

// WatchWithdrawalBonded is a free log subscription operation binding the contract event 0x0c3d250c7831051e78aa6a56679e590374c7c424415ffe4aa474491def2fe705.
//
// Solidity: event WithdrawalBonded(bytes32 indexed transferId, uint256 amount)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchWithdrawalBonded(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeWithdrawalBonded, transferId [][32]byte) (event.Subscription, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "WithdrawalBonded", transferIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeWithdrawalBonded)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "WithdrawalBonded", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseWithdrawalBonded(log types.Log) (*HopL1EthBridgeWithdrawalBonded, error) {
	event := new(HopL1EthBridgeWithdrawalBonded)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "WithdrawalBonded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HopL1EthBridgeWithdrewIterator is returned from FilterWithdrew and is used to iterate over the raw logs and unpacked data for Withdrew events raised by the HopL1EthBridge contract.
type HopL1EthBridgeWithdrewIterator struct {
	Event *HopL1EthBridgeWithdrew // Event containing the contract specifics and raw log

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
func (it *HopL1EthBridgeWithdrewIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HopL1EthBridgeWithdrew)
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
		it.Event = new(HopL1EthBridgeWithdrew)
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
func (it *HopL1EthBridgeWithdrewIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HopL1EthBridgeWithdrewIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HopL1EthBridgeWithdrew represents a Withdrew event raised by the HopL1EthBridge contract.
type HopL1EthBridgeWithdrew struct {
	TransferId    [32]byte
	Recipient     common.Address
	Amount        *big.Int
	TransferNonce [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterWithdrew is a free log retrieval operation binding the contract event 0x9475cdbde5fc71fe2ccd413c82878ee54d061b9f74f9e2e1a03ff1178821502c.
//
// Solidity: event Withdrew(bytes32 indexed transferId, address indexed recipient, uint256 amount, bytes32 transferNonce)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) FilterWithdrew(opts *bind.FilterOpts, transferId [][32]byte, recipient []common.Address) (*HopL1EthBridgeWithdrewIterator, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.FilterLogs(opts, "Withdrew", transferIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &HopL1EthBridgeWithdrewIterator{contract: _HopL1EthBridge.contract, event: "Withdrew", logs: logs, sub: sub}, nil
}

// WatchWithdrew is a free log subscription operation binding the contract event 0x9475cdbde5fc71fe2ccd413c82878ee54d061b9f74f9e2e1a03ff1178821502c.
//
// Solidity: event Withdrew(bytes32 indexed transferId, address indexed recipient, uint256 amount, bytes32 transferNonce)
func (_HopL1EthBridge *HopL1EthBridgeFilterer) WatchWithdrew(opts *bind.WatchOpts, sink chan<- *HopL1EthBridgeWithdrew, transferId [][32]byte, recipient []common.Address) (event.Subscription, error) {

	var transferIdRule []interface{}
	for _, transferIdItem := range transferId {
		transferIdRule = append(transferIdRule, transferIdItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HopL1EthBridge.contract.WatchLogs(opts, "Withdrew", transferIdRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HopL1EthBridgeWithdrew)
				if err := _HopL1EthBridge.contract.UnpackLog(event, "Withdrew", log); err != nil {
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
func (_HopL1EthBridge *HopL1EthBridgeFilterer) ParseWithdrew(log types.Log) (*HopL1EthBridgeWithdrew, error) {
	event := new(HopL1EthBridgeWithdrew)
	if err := _HopL1EthBridge.contract.UnpackLog(event, "Withdrew", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
