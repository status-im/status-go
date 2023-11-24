package walletconnect

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/transactions"
)

// sendTransactionParams instead of transactions.SendTxArgs to allow parsing of hex Uint64 with leading 0 ("0x01") and empty hex value ("0x")
type sendTransactionParams struct {
	transactions.SendTxArgs
	Nonce                JSONProxyType `json:"nonce"`
	Gas                  JSONProxyType `json:"gas"`
	GasPrice             JSONProxyType `json:"gasPrice"`
	Value                JSONProxyType `json:"value"`
	MaxFeePerGas         JSONProxyType `json:"maxFeePerGas"`
	MaxPriorityFeePerGas JSONProxyType `json:"maxPriorityFeePerGas"`
}

func (n *sendTransactionParams) UnmarshalJSON(data []byte) error {
	// Avoid recursion
	type Alias sendTransactionParams
	var alias Alias
	// Fix hex values with leading 0 or empty
	fixWCHexValues := func(input []byte) ([]byte, error) {
		hexStr := string(input)
		if !strings.HasPrefix(hexStr, "\"0x") {
			return input, nil
		}
		trimmedStr := strings.TrimPrefix(hexStr, "\"0x")
		fixedStrNoPrefix := strings.TrimLeft(trimmedStr, "0")
		fixedStr := "\"0x" + fixedStrNoPrefix
		if fixedStr == "\"0x\"" {
			fixedStr = "\"0x0\""
		}

		return []byte(fixedStr), nil
	}

	alias.Nonce = JSONProxyType{target: &alias.SendTxArgs.Nonce, transform: fixWCHexValues}
	alias.Gas = JSONProxyType{target: &alias.SendTxArgs.Gas, transform: fixWCHexValues}
	alias.GasPrice = JSONProxyType{target: &alias.SendTxArgs.GasPrice, transform: fixWCHexValues}
	alias.Value = JSONProxyType{target: &alias.SendTxArgs.Value, transform: fixWCHexValues}
	alias.MaxFeePerGas = JSONProxyType{target: &alias.SendTxArgs.MaxFeePerGas, transform: fixWCHexValues}
	alias.MaxPriorityFeePerGas = JSONProxyType{target: &alias.SendTxArgs.MaxPriorityFeePerGas, transform: fixWCHexValues}

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*n = sendTransactionParams(alias)
	return nil
}

func (n *sendTransactionParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.SendTxArgs)
}

func (s *Service) buildTransaction(request SessionRequest) (response *SessionRequestResponse, err error) {
	if len(request.Params.Request.Params) != 1 {
		return nil, ErrorInvalidParamsCount
	}

	var params sendTransactionParams
	if err = json.Unmarshal(request.Params.Request.Params[0], &params); err != nil {
		return nil, err
	}

	account, err := s.accountsDB.GetAccountByAddress(params.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get active account: %w", err)
	}

	kp, err := s.accountsDB.GetKeypairByKeyUID(account.KeyUID)
	if err != nil {
		return nil, err
	}

	_, chainID, err := parseCaip2ChainID(request.Params.ChainID)
	if err != nil {
		return nil, err
	}

	// In this case we can ignore `unlock` function received from `ValidateAndBuildTransaction` cause `Nonce`
	// will be always set by the initiator of this transaction (by the dapp).
	// Though we will need sort out completely that part since Nonce kept in the local cache is not the most recent one,
	// instead of that we should always ask network what's the most recent known Nonce for the account.
	// Logged issue to handle that: https://github.com/status-im/status-go/issues/4335
	txBeingSigned, _, err := s.transactor.ValidateAndBuildTransaction(chainID, params.SendTxArgs)
	if err != nil {
		return nil, err
	}

	s.txSignDetails = &txSigningDetails{
		from:          common.Address(account.Address),
		chainID:       chainID,
		txBeingSigned: txBeingSigned,
	}

	signer := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
	return &SessionRequestResponse{
		KeyUID:        account.KeyUID,
		Address:       account.Address,
		AddressPath:   account.Path,
		SignOnKeycard: kp.MigratedToKeycard(),
		MesageToSign:  signer.Hash(txBeingSigned),
	}, nil
}

func (s *Service) addSignatureToTransaction(signature string) (*ethTypes.Transaction, error) {
	if s.txSignDetails.txBeingSigned == nil {
		return nil, errors.New("no tx to sign")
	}

	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return nil, err
	}

	return s.transactor.AddSignatureToTransaction(s.txSignDetails.chainID, s.txSignDetails.txBeingSigned, signatureBytes)
}

func (s *Service) buildMessage(request SessionRequest, addressIndex int, messageIndex int,
	handleTypedData bool) (response *SessionRequestResponse, err error) {
	if len(request.Params.Request.Params) != 2 {
		return nil, ErrorInvalidParamsCount
	}

	if addressIndex > 1 || addressIndex < 0 || messageIndex > 1 || messageIndex < 0 {
		return nil, ErrorInvalidAddressMsgIndex
	}

	var address types.Address
	if err := json.Unmarshal(request.Params.Request.Params[addressIndex], &address); err != nil {
		return nil, err
	}

	account, err := s.accountsDB.GetAccountByAddress(address)
	if err != nil {
		return nil, fmt.Errorf("failed to get active account: %w", err)
	}

	kp, err := s.accountsDB.GetKeypairByKeyUID(account.KeyUID)
	if err != nil {
		return nil, err
	}

	var hash []byte
	if !handleTypedData {
		var dBytes types.HexBytes
		if err := json.Unmarshal(request.Params.Request.Params[messageIndex], &dBytes); err != nil {
			return nil, err
		}
		hash = crypto.TextHash(dBytes)
	} else {
		var typedDataJSON string
		if err := json.Unmarshal(request.Params.Request.Params[messageIndex], &typedDataJSON); err != nil {
			return nil, err
		}

		var typedData apitypes.TypedData
		if err := json.Unmarshal([]byte(typedDataJSON), &typedData); err != nil {
			return nil, err
		}

		hash, _, err = apitypes.TypedDataAndHash(typedData)
		if err != nil {
			return nil, err
		}
	}

	return &SessionRequestResponse{
		KeyUID:        account.KeyUID,
		Address:       account.Address,
		AddressPath:   account.Path,
		SignOnKeycard: kp.MigratedToKeycard(),
		MesageToSign:  types.HexBytes(hash),
	}, nil
}
