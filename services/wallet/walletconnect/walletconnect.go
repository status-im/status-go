package walletconnect

import (
	"crypto/ecdsa"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"
	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/typeddata"
	"github.com/status-im/status-go/services/wallet/walletevent"
)

const (
	SupportedEip155Namespace = "eip155"

	ProposeUserPairEvent = walletevent.EventType("WalletConnectProposeUserPair")
)

var (
	ErrorInvalidSessionProposal = errors.New("invalid session proposal")
	ErrorNamespaceNotSupported  = errors.New("namespace not supported")
	ErrorChainsNotSupported     = errors.New("chains not supported")
	ErrorInvalidParamsCount     = errors.New("invalid params count")
	ErrorInvalidAddressMsgIndex = errors.New("invalid address and/or msg index (must be 0 or 1)")
	ErrorMethodNotSupported     = errors.New("method not supported")
)

type Topic string

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

type Verified struct {
	VerifyURL  string `json:"verifyUrl"`
	Validation string `json:"validation"`
	Origin     string `json:"origin"`
	IsScam     bool   `json:"isScam,omitempty"`
}

type VerifyContext struct {
	Verified Verified `json:"verified"`
}

// Params has RequiredNamespaces entries if part of "proposal namespace" and Namespaces entries if part of "session namespace"
// see https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#controller-side-validation-of-incoming-proposal-namespaces-wallet
type Params struct {
	ID                 int64                `json:"id"`
	PairingTopic       Topic                `json:"pairingTopic"`
	Expiry             int64                `json:"expiry"`
	RequiredNamespaces map[string]Namespace `json:"requiredNamespaces"`
	OptionalNamespaces map[string]Namespace `json:"optionalNamespaces"`
	Proposer           Proposer             `json:"proposer"`
	Verify             VerifyContext        `json:"verifyContext"`
}

type SessionProposal struct {
	ID     int64  `json:"id"`
	Params Params `json:"params"`
}

type PairSessionResponse struct {
	SessionProposal     SessionProposal      `json:"sessionProposal"`
	SupportedNamespaces map[string]Namespace `json:"supportedNamespaces"`
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
	Topic  Topic         `json:"topic"`
	Params RequestParams `json:"params"`
	Verify VerifyContext `json:"verifyContext"`
}

type SessionDelete struct {
	ID    int64 `json:"id"`
	Topic Topic `json:"topic"`
}

type Session struct {
	Acknowledged       bool                 `json:"acknowledged"`
	Controller         string               `json:"controller"`
	Expiry             int64                `json:"expiry"`
	Namespaces         map[string]Namespace `json:"namespaces"`
	OptionalNamespaces map[string]Namespace `json:"optionalNamespaces"`
	PairingTopic       Topic                `json:"pairingTopic"`
	Peer               Proposer             `json:"peer"`
	Relay              json.RawMessage      `json:"relay"`
	RequiredNamespaces map[string]Namespace `json:"requiredNamespaces"`
	Self               Proposer             `json:"self"`
	Topic              Topic                `json:"topic"`
}

// Valid namespace
func (n *Namespace) Valid(namespaceName string, chainID *uint64) bool {
	if chainID == nil {
		if len(n.Chains) == 0 {
			logutils.ZapLogger().Warn("namespace doesn't refer to any chain")
			return false
		}
		for _, caip2Str := range n.Chains {
			resolvedNamespaceName, _, err := parseCaip2ChainID(caip2Str)
			if err != nil {
				logutils.ZapLogger().Warn("namespace chain not in caip2 format",
					zap.String("chain", caip2Str),
					zap.Error(err),
				)
				return false
			}

			if resolvedNamespaceName != namespaceName {
				logutils.ZapLogger().Warn("namespace name doesn't match",
					zap.String("namespace", namespaceName),
					zap.String("chain", caip2Str),
				)
				return false
			}
		}
	}
	return true
}

// ValidateForProposal validates params part of the Proposal Namespace
func (p *Params) ValidateForProposal() bool {
	for key, ns := range p.RequiredNamespaces {
		var chainID *uint64
		if strings.Contains(key, ":") {
			resolvedNamespaceName, cID, err := parseCaip2ChainID(key)
			if err != nil {
				logutils.ZapLogger().Warn("params validation failed CAIP-2",
					zap.String("str", key),
					zap.Error(err),
				)
				return false
			}
			key = resolvedNamespaceName
			chainID = &cID
		}

		if !isValidNamespaceName(key) {
			logutils.ZapLogger().Warn("invalid namespace name", zap.String("namespace", key))
			return false
		}

		if !ns.Valid(key, chainID) {
			return false
		}
	}

	return true
}

// ValidateProposal validates params part of the Proposal Namespace
// https://specs.walletconnect.com/2.0/specs/clients/sign/namespaces#controller-side-validation-of-incoming-proposal-namespaces-wallet
func (p *SessionProposal) ValidateProposal() bool {
	return p.Params.ValidateForProposal()
}

// AddSession adds a new active session to the database
func AddSession(db *sql.DB, networks []params.Network, session_json string) error {
	var session Session
	err := json.Unmarshal([]byte(session_json), &session)
	if err != nil {
		return fmt.Errorf("unmarshal session: %v", err)
	}

	chains := supportedChainsInSession(session)
	testChains, err := areTestChains(networks, chains)
	if err != nil {
		return fmt.Errorf("areTestChains: %v", err)
	}

	rowEntry := DBSession{
		Topic:            session.Topic,
		Disconnected:     false,
		SessionJSON:      session_json,
		Expiry:           session.Expiry,
		CreatedTimestamp: time.Now().Unix(),
		PairingTopic:     session.PairingTopic,
		TestChains:       testChains,
		DBDApp: DBDApp{
			URL:  session.Peer.Metadata.URL,
			Name: session.Peer.Metadata.Name,
		},
	}
	if len(session.Peer.Metadata.Icons) > 0 {
		rowEntry.IconURL = session.Peer.Metadata.Icons[0]
	}

	return UpsertSession(db, rowEntry)
}

// areTestChains assumes chains to tests are all testnets or all mainnets
func areTestChains(networks []params.Network, chainIDs []uint64) (isTest bool, err error) {
	for _, n := range networks {
		for _, chainID := range chainIDs {
			if n.ChainID == chainID {
				return n.IsTest, nil
			}
		}
	}

	return false, fmt.Errorf("no network found for chainIDs %v", chainIDs)
}

func supportedChainsInSession(session Session) []uint64 {
	caipChains := session.Namespaces[SupportedEip155Namespace].Chains
	chains := make([]uint64, 0, len(caipChains))
	for _, caip2Str := range caipChains {
		_, chainID, err := parseCaip2ChainID(caip2Str)
		if err != nil {
			logutils.ZapLogger().Warn("Failed parsing CAIP-2",
				zap.String("str", caip2Str),
				zap.Error(err),
			)
			continue
		}

		chains = append(chains, chainID)
	}
	return chains
}

func caip10Accounts(accounts []*accounts.Account, chains []uint64) []string {
	addresses := make([]string, 0, len(accounts)*len(chains))
	for _, acc := range accounts {
		for _, chainID := range chains {
			addresses = append(addresses, fmt.Sprintf("%s:%s:%s", SupportedEip155Namespace, strconv.FormatUint(chainID, 10), acc.Address.Hex()))
		}
	}
	return addresses
}

func SafeSignTypedDataForDApps(typedJson string, privateKey *ecdsa.PrivateKey, chainID uint64, legacy bool) (types.HexBytes, error) {
	// Parse the data for both legacy and non-legacy cases to validate the chain
	var typed typeddata.TypedData
	err := json.Unmarshal([]byte(typedJson), &typed)
	if err != nil {
		return types.HexBytes{}, err
	}

	chain := new(big.Int).SetUint64(chainID)

	var sig hexutil.Bytes
	if legacy {
		sig, err = typeddata.Sign(typed, privateKey, chain)
	} else {
		// Validate chainID if part of the typed data
		if _, exist := typed.Domain[typeddata.ChainIDKey]; exist {
			if err := typed.ValidateChainID(chain); err != nil {
				return types.HexBytes{}, err
			}
		}

		var typedV4 signercore.TypedData
		err = json.Unmarshal([]byte(typedJson), &typedV4)
		if err != nil {
			return types.HexBytes{}, err
		}

		sig, err = typeddata.SignTypedDataV4(typedV4, privateKey, chain)
	}
	if err != nil {
		return types.HexBytes{}, err
	}

	return types.HexBytes(sig), err
}
