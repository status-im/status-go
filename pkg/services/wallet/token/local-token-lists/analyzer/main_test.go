package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	defaulttokenlists "github.com/status-im/status-go/pkg/services/wallet/token/local-token-lists/default-lists"
)

func TestRebuildTokensMapSkipsInvalidLists(t *testing.T) {
	const valid = `{"name":"Valid","version":{"major":1,"minor":0,"patch":0},"tokens":[{"chainId":1,"address":"0x0000000000000000000000000000000000001234","symbol":"VALID","decimals":18}]}`
	for _, invalid := range []defaulttokenlists.DownloadedTokenList{
		{ID: "invalid-json", JsonData: []byte(`{`)},
		{ID: "invalid-tags", JsonData: []byte(`{"name":"Invalid tags","version":{"major":1,"minor":0,"patch":0},"tags":{"bad":1e1000},"tokens":[]}`)},
		{ID: "", JsonData: []byte(valid)},
	} {
		t.Run(invalid.ID, func(t *testing.T) {
			lists, err := rebuildTokensMap([]defaulttokenlists.DownloadedTokenList{
				{ID: "before", JsonData: []byte(valid)},
				invalid,
				{ID: "after", JsonData: []byte(valid)},
			})
			require.NoError(t, err)
			require.Len(t, lists, 2)
			for _, id := range []string{"before", "after"} {
				require.Contains(t, lists, id)
				require.Len(t, lists[id].Tokens, 1)
				require.Equal(t, "VALID", lists[id].Tokens[0].Symbol)
			}
		})
	}
}
