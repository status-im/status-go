//go:build tkl

package token

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/builder"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/parsers"
	"github.com/status-im/go-wallet-sdk/pkg/tokens/types"
	"github.com/status-im/nim-token-lists/go/tkl"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
)

type shadowReport struct {
	Status                string
	SDKTokens             int
	NimTokens             int
	Differences           int
	ExpectedCustomMarkers int
	ParseErrors           int
	Examples              []string
}

func (r *shadowReport) example(value string) {
	if len(r.Examples) < 20 {
		r.Examples = append(r.Examples, value)
	}
}

// compareShadow uses only captured bytes and the SDK's parser/builder. It never
// starts a legacy manager, reads a database, performs HTTP, or publishes tokens.
func compareShadow(ctx context.Context, s tklmanager.ShadowSnapshot) shadowReport {
	result := shadowReport{Status: "match", NimTokens: len(s.Tokens)}
	if s.Skipped != "" {
		result.Status = s.Skipped
		return result
	}
	if ctx.Err() != nil {
		result.Status = "cancelled"
		return result
	}
	b := builder.New(s.Config.Chains, s.Config.Policy.SkippedKeys)
	if err := b.AddNativeTokenList(); err != nil {
		result.Status = "builder_error"
		return result
	}
	initial := make(map[string]tkl.ListContent, len(s.Config.InitialLists))
	stored := make(map[string]tkl.ListContent, len(s.Contents))
	for _, row := range s.Config.InitialLists {
		initial[row.ID] = row
	}
	for _, row := range s.Contents {
		stored[row.ID] = row
	}
	// Match manager priority without starting its scheduler or storage adapters.
	ids := make([]string, 0, len(initial)+len(stored))
	if s.Config.MainListID != "" {
		ids = append(ids, s.Config.MainListID)
	}
	keys := make([]string, 0, len(initial))
	for id := range initial {
		if id != "" && id != s.Config.MainListID {
			keys = append(keys, id)
		}
	}
	sort.Strings(keys)
	ids = append(ids, keys...)
	keys = keys[:0]
	for id := range stored {
		if _, ok := initial[id]; !ok && id != s.Config.MainListID && id != s.Config.RegistryID {
			keys = append(keys, id)
		}
	}
	sort.Strings(keys)
	ids = append(ids, keys...)
	parsedTokens := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			result.Status = "cancelled"
			return result
		}
		row, ok := stored[id]
		if !ok || row.Body == "" {
			row = initial[id]
		}
		var parser parsers.TokenListParser = &parsers.StandardTokenListParser{}
		if id == s.Config.MainListID {
			parser = &parsers.StatusTokenListParser{}
		}
		list, err := parser.Parse([]byte(row.Body), s.Config.Chains)
		if err != nil {
			result.ParseErrors++
			hash := sha256.Sum256([]byte(id))
			result.example(fmt.Sprintf("parse:%x", hash[:6]))
			continue
		}
		parsedTokens += len(list.Tokens)
		if parsedTokens > 100000 {
			result.Status = "parsed_token_limit"
			return result
		}
		b.AddTokenList(id, list)
	}
	customs := &types.TokenList{ID: "custom", Name: "Custom tokens"}
	for _, row := range s.Customs {
		if ctx.Err() != nil {
			result.Status = "cancelled"
			return result
		}
		token := &types.Token{ChainID: row.ChainID, Address: common.HexToAddress(row.Address), Symbol: row.Symbol, Name: row.Name, Decimals: uint(row.Decimals), LogoURI: row.LogoURI, CrossChainID: row.CrossChainID}
		if token.Validate(s.Config.Chains) == nil {
			customs.Tokens = append(customs.Tokens, token)
		}
	}
	b.AddTokenList(customs.ID, customs)
	expected := b.GetTokens()
	expectedCustom := make(map[string]bool, len(customs.Tokens))
	for _, row := range customs.Tokens {
		if expected[row.Key()] == row {
			expectedCustom[row.Key()] = true
		}
	}
	actual := make(map[string]types.Token, len(s.Tokens))
	for _, row := range s.Tokens {
		actual[row.Key()] = row
	}
	keys = keys[:0]
	for key := range expected {
		keys = append(keys, key)
	}
	for key := range actual {
		if _, ok := expected[key]; !ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result.SDKTokens = len(expected)
	for _, key := range keys {
		if ctx.Err() != nil {
			result.Status = "cancelled"
			return result
		}
		want, wantOK := expected[key]
		got, gotOK := actual[key]
		if !wantOK || !gotOK {
			result.Differences++
			if !wantOK {
				result.example(key + ":missing_sdk")
			} else {
				result.example(key + ":missing_nim")
			}
			continue
		}
		normalized := *want
		if expectedCustom[key] && got.CustomToken && !want.CustomToken {
			normalized.CustomToken = true
			result.ExpectedCustomMarkers++
		}
		if normalized != got {
			result.Differences++
			result.example(key + ":" + strings.Join(shadowTokenFields(normalized, got), ","))
		}
	}
	if result.Differences > 0 {
		result.Status = "mismatch"
	}
	// A failed parser never counts as parity, even if the remaining tokens match.
	if result.ParseErrors > 0 {
		result.Status = "source_error"
	}
	return result
}

func shadowTokenFields(a, b types.Token) []string {
	fields := make([]string, 0, 6)
	if a.Name != b.Name {
		fields = append(fields, "name")
	}
	if a.Symbol != b.Symbol {
		fields = append(fields, "symbol")
	}
	if a.Decimals != b.Decimals {
		fields = append(fields, "decimals")
	}
	if a.LogoURI != b.LogoURI {
		fields = append(fields, "logoUri")
	}
	if a.CrossChainID != b.CrossChainID {
		fields = append(fields, "crossChainId")
	}
	if a.CustomToken != b.CustomToken {
		fields = append(fields, "custom")
	}
	return fields
}

func logShadowComparison(ctx context.Context, snapshot tklmanager.ShadowSnapshot) {
	started := time.Now()
	report := compareShadow(ctx, snapshot)
	if report.Status == "cancelled" {
		return
	}
	logutils.ZapLogger().Info("Token catalogue shadow comparison",
		zap.String("status", report.Status), zap.Uint64("revision", snapshot.Revision), zap.Int("sequence", snapshot.Sequence), zap.Int("coalesced", snapshot.Coalesced),
		zap.Int("nimTokens", report.NimTokens), zap.Int("sdkTokens", report.SDKTokens), zap.Int("differences", report.Differences), zap.Int("expectedCustomMarkers", report.ExpectedCustomMarkers), zap.Int("parseErrors", report.ParseErrors),
		zap.Strings("examples", report.Examples), zap.Duration("mirrorBuild", snapshot.MirrorBuild), zap.Duration("comparison", time.Since(started)))
}
