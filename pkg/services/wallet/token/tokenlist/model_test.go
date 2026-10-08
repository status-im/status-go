package tokenlist

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestTokenWireFormat(t *testing.T) {
	token := Token{CrossChainID: "ethereum", ChainID: 1, Decimals: 18, Name: "Ether", Symbol: "ETH"}
	body, err := json.Marshal(token)
	require.NoError(t, err)
	require.JSONEq(t, `{"crossChainId":"ethereum","chainId":1,"address":"0x0000000000000000000000000000000000000000","decimals":18,"name":"Ether","symbol":"ETH","logoUri":"","custom":false}`, string(body))
	var decoded Token
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, token, decoded)
	require.True(t, token.IsNative())
	token.Address = common.HexToAddress("0xaBcD")
	require.False(t, token.IsNative())
	require.Equal(t, "1-0x000000000000000000000000000000000000abcd", token.Key())
}

func TestTokenKeyCompatibility(t *testing.T) {
	for _, key := range []string{"", "-1-0x01", "1-0x01-extra", "18446744073709551616-0x01"} {
		_, _, ok := ChainAndAddressFromTokenKey(key)
		require.False(t, ok, key)
	}
	// Preserve the existing permissive address parsing at the DTO boundary.
	for key, address := range map[string]common.Address{
		"1-0xaBcD": common.HexToAddress("0xabcd"), "1-nothex": {},
	} {
		chain, got, ok := ChainAndAddressFromTokenKey(key)
		require.True(t, ok)
		require.Equal(t, uint64(1), chain)
		require.Equal(t, address, got)
	}
}

func TestListWireFormat(t *testing.T) {
	list := TokenList{ID: "list", Version: Version{Major: 1, Minor: 2, Patch: 3}}
	body, err := json.Marshal(list)
	require.NoError(t, err)
	require.JSONEq(t, `{"id":"list","name":"","timestamp":"","fetchedTimestamp":"","source":"","version":{"major":1,"minor":2,"patch":3},"tags":null,"logoUri":"","keywords":null,"tokens":null}`, string(body))
	require.Equal(t, "1.2.3", list.Version.String())
	list.Tokens, list.Keywords, list.Tags = []*Token{}, []string{}, map[string]interface{}{}
	body, err = json.Marshal(list)
	require.NoError(t, err)
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &decoded))
	require.Equal(t, []interface{}{}, decoded["tokens"])
	require.Equal(t, []interface{}{}, decoded["keywords"])
	require.Equal(t, map[string]interface{}{}, decoded["tags"])
}
