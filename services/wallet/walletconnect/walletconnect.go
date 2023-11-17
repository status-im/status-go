package walletconnect

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const ProposeUserPairEvent = walletevent.EventType("WalletConnectProposeUserPair")

var ErrorChainsNotSupported = errors.New("chains not supported")
var ErrorInvalidParamsCount = errors.New("invalid params count")
var ErrorMethodNotSupported = errors.New("method not supported")

type Namespace struct {
	Methods  []string `json:"methods"`
	Chains   []string `json:"chains"` // CAIP-2 format e.g. ["eip155:1"]
	Events   []string `json:"events"`
	Accounts []string `json:"accounts,omitempty"` // CAIP-10 format e.g. ["eip155:1:0x453...228"]
}

type Metadata struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Icons       []string `json:"icons"`
	Name        string   `json:"name"`
	VerifyURL   string   `json:"verifyUrl"`
}

type Proposer struct {
	PublicKey string   `json:"publicKey"`
	Metadata  Metadata `json:"metadata"`
}

type Namespaces struct {
	Eip155 Namespace `json:"eip155"`
	// We ignore non ethereum namespaces
}

type VerifyContext struct {
	Verified struct {
		VerifyURL  string `json:"verifyUrl"`
		Validation string `json:"validation"`
		Origin     string `json:"origin"`
		IsScam     bool   `json:"isScam,omitempty"`
	} `json:"verified"`
}

type Params struct {
	ID                 int64         `json:"id"`
	PairingTopic       string        `json:"pairingTopic"`
	Expiry             int64         `json:"expiry"`
	RequiredNamespaces Namespaces    `json:"requiredNamespaces"`
	OptionalNamespaces Namespaces    `json:"optionalNamespaces"`
	Proposer           Proposer      `json:"proposer"`
	Verify             VerifyContext `json:"verifyContext"`
}

type SessionProposal struct {
	ID     uint64 `json:"id"`
	Params Params `json:"params"`
}

type PairSessionResponse struct {
	SessionProposal     SessionProposal `json:"sessionProposal"`
	SupportedNamespaces Namespaces      `json:"supportedNamespaces"`
}

type RequestParams struct {
	Request struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	} `json:"request"`
	ChainID string `json:"chainId"`
}

type SessionRequest struct {
	ID     int64         `json:"id"`
	Topic  string        `json:"topic"`
	Params RequestParams `json:"params"`
	Verify VerifyContext `json:"verifyContext"`
}

type SessionRequestResponse struct {
	KeyUID        string        `json:"keyUid,omitempty"`
	Address       types.Address `json:"address,omitempty"`
	AddressPath   string        `json:"addressPath,omitempty"`
	SignOnKeycard bool          `json:"signOnKeycard,omitempty"`
	MesageToSign  interface{}   `json:"messageToSign,omitempty"`
	SignedMessage interface{}   `json:"signedMessage,omitempty"`
}

func sessionProposalToSupportedChain(caipChains []string, supportsChain func(uint64) bool) (chains []uint64, eipChains []string) {
	chains = make([]uint64, 0, 1)
	eipChains = make([]string, 0, 1)
	for _, caip2Str := range caipChains {
		chainID, err := parseCaip2ChainID(caip2Str)
		if err != nil {
			log.Warn("Failed parsing CAIP-2", "str", caip2Str, "error", err)
			continue
		}

		if !supportsChain(chainID) {
			continue
		}

		eipChains = append(eipChains, caip2Str)
		chains = append(chains, chainID)
	}
	return
}

func activeToOwnedAccounts(activeAccounts []*accounts.Account) []types.Address {
	addresses := make([]types.Address, 0, 1)
	for _, account := range activeAccounts {
		if account.Type != accounts.AccountTypeWatch {
			addresses = append(addresses, account.Address)
		}
	}
	return addresses
}

func caip10Accounts(addresses []types.Address, chains []uint64) []string {
	accounts := make([]string, 0, len(addresses)*len(chains))
	for _, address := range addresses {
		for _, chainID := range chains {
			accounts = append(accounts, "eip155:"+strconv.FormatUint(chainID, 10)+":"+address.Hex())
		}
	}
	return accounts
}
