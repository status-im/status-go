// Package tokenlist defines the wallet's catalogue data and read contract.
// Catalogue parsing, validation and merge policy belong to the native core.
package tokenlist

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const LocalSourceURL = "local"

// Token is the shared catalogue DTO, before wallet/community enrichment.
type Token struct {
	CrossChainID string         `json:"crossChainId"`
	ChainID      uint64         `json:"chainId"`
	Address      common.Address `json:"address"`
	Decimals     uint           `json:"decimals"`
	Name         string         `json:"name"`
	Symbol       string         `json:"symbol"`
	LogoURI      string         `json:"logoUri"`
	CustomToken  bool           `json:"custom"`
}

func TokenKey(chainID uint64, address common.Address) string {
	// Plain lowercase hex: the same text as strings.ToLower(address.Hex()) without the keccak checksum.
	buf := make([]byte, 0, 20+1+2+2*common.AddressLength)
	buf = strconv.AppendUint(buf, chainID, 10)
	buf = append(buf, "-0x"...)
	buf = hex.AppendEncode(buf, address[:])
	return string(buf)
}

// ChainAndAddressFromTokenKey preserves the wallet's existing key decoding.
func ChainAndAddressFromTokenKey(key string) (uint64, common.Address, bool) {
	parts := strings.Split(key, "-")
	if len(parts) != 2 {
		return 0, common.Address{}, false
	}
	chainID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, common.Address{}, false
	}
	return chainID, common.HexToAddress(parts[1]), true
}

// ChainAddress identifies one token in a batch lookup.
type ChainAddress struct {
	ChainID uint64
	Address common.Address
}

// ChainToken is a token narrowed to what balance fetching reads.
type ChainToken struct {
	ChainID  uint64
	Address  common.Address
	Decimals uint
}

func (t ChainToken) IsNative() bool { return t.Address == (common.Address{}) }

func (t *Token) Key() string    { return TokenKey(t.ChainID, t.Address) }
func (t *Token) IsNative() bool { return t.Address == (common.Address{}) }

func (t *Token) ChainToken() ChainToken {
	return ChainToken{ChainID: t.ChainID, Address: t.Address, Decimals: t.Decimals}
}

type Version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}

func (v *Version) String() string { return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch) }

type TokenList struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Timestamp        string                 `json:"timestamp"`
	FetchedTimestamp string                 `json:"fetchedTimestamp"`
	Source           string                 `json:"source"`
	Version          Version                `json:"version"`
	Tags             map[string]interface{} `json:"tags"`
	LogoURI          string                 `json:"logoUri"`
	Keywords         []string               `json:"keywords"`
	Tokens           []*Token               `json:"tokens"`
}

// Catalogue is implemented by the native facade.
type Catalogue interface {
	Start(context.Context, bool, chan struct{}) error
	Stop() error
	EnableAutoRefresh(context.Context) error
	DisableAutoRefresh(context.Context) error
	TriggerRefresh(context.Context) error
	SetChains([]uint64) error
	UniqueTokens() []*Token
	GetTokenByChainAddress(uint64, common.Address) (*Token, bool)
	GetTokensByChain(uint64) []*Token
	GetTokensByKeys([]string) ([]*Token, error)
	TokenLists() []*TokenList
	TokenList(string) (*TokenList, bool)
}
