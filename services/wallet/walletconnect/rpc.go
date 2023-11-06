package walletconnect

import (
	"encoding/json"
	"strings"

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

func (s *Service) sendTransaction(request SessionRequest, hashedPassword string) (response *SessionRequestResponse, err error) {
	if len(request.Params.Request.Params) != 1 {
		return nil, ErrorInvalidParamsCount
	}

	var params sendTransactionParams
	if err := json.Unmarshal(request.Params.Request.Params[0], &params); err != nil {
		return nil, err
	}

	acc, err := s.gethManager.GetVerifiedWalletAccount(s.accountsDB, params.From.Hex(), hashedPassword)
	if err != nil {
		return nil, err
	}

	// TODO: export it as a JSON parsable type
	chainID, err := parseCaip2ChainID(request.Params.ChainID)
	if err != nil {
		return nil, err
	}

	hash, err := s.transactor.SendTransactionWithChainID(chainID, params.SendTxArgs, acc)
	if err != nil {
		return nil, err
	}

	return &SessionRequestResponse{
		SessionRequest: request,
		Signed:         hash.Bytes(),
	}, nil
}

func (s *Service) personalSign(request SessionRequest, hashedPassword string) (response *SessionRequestResponse, err error) {
	if len(request.Params.Request.Params) != 2 {
		return nil, ErrorInvalidParamsCount
	}

	var address types.Address
	if err := json.Unmarshal(request.Params.Request.Params[1], &address); err != nil {
		return nil, err
	}

	acc, err := s.gethManager.GetVerifiedWalletAccount(s.accountsDB, address.Hex(), hashedPassword)
	if err != nil {
		return nil, err
	}

	var dBytes types.HexBytes
	if err := json.Unmarshal(request.Params.Request.Params[0], &dBytes); err != nil {
		return nil, err
	}

	hash := crypto.TextHash(dBytes)

	sig, err := crypto.Sign(hash, acc.AccountKey.PrivateKey)
	if err != nil {
		return nil, err
	}

	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

	return &SessionRequestResponse{
		SessionRequest: request,
		Signed:         types.HexBytes(sig),
	}, nil
}
