package token

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/db/appdatabase"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/db/walletdb"
	"github.com/status-im/status-go/internal/testutils"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/pkg/services/wallet/token/tklmanager"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

var benchChains = []uint64{1, 10, 42161, 8453, 56, 59144}

// benchLists serves the registry and the embedded lists from memory. A list
// answers 304 to its current etag; bumping a list's version changes its etag.
type benchLists struct {
	versions map[string]*atomic.Int64
	bodies   map[string][]byte
	registry []byte
}

func newBenchLists(tb testing.TB) *benchLists {
	ids := initialListIDsFromEmbedded()
	sort.Strings(ids)
	lists := &benchLists{versions: map[string]*atomic.Int64{}, bodies: map[string][]byte{}}
	entries := make([]string, 0, len(ids))
	for _, id := range ids {
		data, err := initialListProviderFromEmbedded(id)
		require.NoError(tb, err)
		lists.bodies[id] = data
		lists.versions[id] = &atomic.Int64{}
		entries = append(entries, fmt.Sprintf(`{"id":%q,"sourceUrl":"https://bench.invalid/%s"}`, id, id))
	}
	lists.registry = []byte(`{"timestamp":"2026-01-01T00:00:00Z","version":{"major":1,"minor":0,"patch":0},"tokenLists":[` + strings.Join(entries, ",") + `]}`)
	lists.versions[remoteListOfTokenListsID] = &atomic.Int64{}
	lists.bodies[remoteListOfTokenListsID] = lists.registry
	return lists
}

func (l *benchLists) RoundTrip(req *http.Request) (*http.Response, error) {
	id := strings.TrimPrefix(req.URL.Path, "/")
	if req.URL.String() == remoteListOfTokenLists {
		id = remoteListOfTokenListsID
	}
	version, ok := l.versions[id]
	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}, Request: req}, nil
	}
	etag := fmt.Sprintf(`"%s-%d"`, id, version.Load())
	header := http.Header{"Etag": []string{etag}}
	if req.Header.Get("If-None-Match") == etag {
		return &http.Response{StatusCode: http.StatusNotModified, Body: io.NopCloser(bytes.NewReader(nil)), Header: header, Request: req}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(l.bodies[id])), Header: header, ContentLength: int64(len(l.bodies[id])), Request: req}, nil
}

type benchEnv struct {
	manager *Manager
	lists   *benchLists
	close   func()
}

func newBenchEnv(tb testing.TB, stored bool) *benchEnv {
	appDB, err := testutils.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(tb, err)
	walletDB, err := testutils.SetupTestMemorySQLDB(walletdb.DbInitializer{})
	require.NoError(tb, err)
	settingsDB, err := settings.MakeNewDB(appDB)
	require.NoError(tb, err)
	networks := json.RawMessage(`{}`)
	require.NoError(tb, settingsDB.CreateSettings(settings.Settings{Networks: &networks}, params.NodeConfig{}))
	env := &benchEnv{manager: &Manager{walletDB: walletDB, settings: settingsDB, tokenBalancesStorage: balanceStorage{walletDB: walletDB}}, lists: newBenchLists(tb)}
	if stored {
		insertBenchStoredLists(tb, walletDB, env.lists)
	}
	env.close = func() {
		if env.manager.tokensManager != nil {
			require.NoError(tb, env.manager.tokensManager.Stop())
		}
		require.NoError(tb, appDB.Close())
		require.NoError(tb, walletDB.Close())
	}
	return env
}

func insertBenchStoredLists(tb testing.TB, db *sql.DB, lists *benchLists) {
	for id, body := range lists.bodies {
		source := "https://bench.invalid/" + id
		if id == remoteListOfTokenListsID {
			source = remoteListOfTokenLists
		}
		_, err := db.Exec(`INSERT INTO token_lists (id,source,etag,tokens_json,fetched) VALUES (?,?,?,?,?)`, id, source, fmt.Sprintf(`"%s-0"`, id), body, time.Unix(1_790_000_000, 0).UTC())
		require.NoError(tb, err)
	}
}

func (e *benchEnv) start(tb testing.TB) *tklmanager.Manager {
	client := &http.Client{Transport: e.lists}
	facade, err := newTKLRefreshManager(e.manager, benchChains, time.Time{}, client, time.Hour, time.Hour)
	require.NoError(tb, err)
	require.NoError(tb, facade.Start(context.Background(), false, nil))
	e.manager.tokensManager = facade
	return facade
}

func (e *benchEnv) stop(tb testing.TB) {
	require.NoError(tb, e.manager.tokensManager.Stop())
	e.manager.tokensManager = nil
}

// nativeHeapInUse reports malloc'd bytes (Nim and SQLite) when a platform probe is linked in.
var nativeHeapInUse func() int64

// reportRetained reports the live heap a value keeps after a full GC.
func reportRetained(b *testing.B, build func() func()) {
	runtime.GC()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	var native int64
	if nativeHeapInUse != nil {
		native = nativeHeapInUse()
	}
	release := build()
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&after)
	b.ReportMetric(float64(int64(after.HeapAlloc)-int64(before.HeapAlloc)), "retained-B")
	if nativeHeapInUse != nil {
		b.ReportMetric(float64(nativeHeapInUse()-native), "native-B")
	}
	release()
}

func BenchmarkTKLLoginLoad(b *testing.B) {
	for _, stored := range []bool{false, true} {
		b.Run(fmt.Sprintf("stored=%v", stored), func(b *testing.B) {
			env := newBenchEnv(b, stored)
			defer env.close()
			b.ReportAllocs()
			for b.Loop() {
				env.start(b)
				b.StopTimer()
				env.stop(b)
				b.StartTimer()
			}
			b.StopTimer()
			reportRetained(b, func() func() {
				env.start(b)
				return func() { env.stop(b) }
			})
		})
	}
}

func BenchmarkTKLRefresh(b *testing.B) {
	b.Run("first", func(b *testing.B) {
		env := newBenchEnv(b, false)
		defer env.close()
		b.ReportAllocs()
		for b.Loop() {
			b.StopTimer()
			_, err := env.manager.walletDB.Exec(`DELETE FROM token_lists`)
			require.NoError(b, err)
			facade := env.start(b)
			b.StartTimer()
			require.NoError(b, facade.TriggerRefresh(context.Background()))
			b.StopTimer()
			env.stop(b)
			b.StartTimer()
		}
		b.StopTimer()
		reportRetained(b, func() func() {
			_, err := env.manager.walletDB.Exec(`DELETE FROM token_lists`)
			require.NoError(b, err)
			facade := env.start(b)
			require.NoError(b, facade.TriggerRefresh(context.Background()))
			return func() { env.stop(b) }
		})
	})
	b.Run("oneList", func(b *testing.B) {
		env := newBenchEnv(b, true)
		defer env.close()
		facade := env.start(b)
		require.NoError(b, facade.TriggerRefresh(context.Background()))
		uniswap := env.lists.versions["uniswap"]
		require.NotNil(b, uniswap)
		b.ReportAllocs()
		for b.Loop() {
			uniswap.Add(1)
			require.NoError(b, facade.TriggerRefresh(context.Background()))
		}
	})
}

func BenchmarkTKLLookups(b *testing.B) {
	env := newBenchEnv(b, false)
	defer env.close()
	env.start(b)
	tm := env.manager
	all, err := tm.GetAllTokens()
	require.NoError(b, err)
	require.Greater(b, len(all), 1000)
	keys := make([]string, 0, 50)
	balances := map[common.Address][]tokentypes.StorageToken{}
	owner := common.HexToAddress("0x1234")
	for i := 0; i < len(all) && len(keys) < 50; i += len(all) / 50 {
		keys = append(keys, all[i].Key())
		balances[owner] = append(balances[owner], tokentypes.StorageToken{TokenAddress: all[i].Address, TokenChainID: all[i].ChainID, RawBalance: "1", Balance: big.NewFloat(1)})
	}
	require.NoError(b, tm.CacheBalances(balances))
	usdc := "1-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	usdt := "1-0xdac17f958d2ee523a2206206994597c13d831ec7"

	b.Run("balanceByChains", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			tokens, err := tm.GetTokensByChains(benchChains)
			if err != nil || len(tokens) == 0 {
				b.Fatal(err)
			}
		}
	})
	b.Run("balanceChainTokens", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			tokens, err := tm.GetChainTokens(benchChains, nil)
			if err != nil || len(tokens) == 0 {
				b.Fatal(err)
			}
		}
	})
	// The balance fetcher asks for each chain's contract addresses.
	b.Run("contractAddresses", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for _, chain := range benchChains {
				tokens, err := tm.GetChainTokens([]uint64{chain}, nil)
				if err != nil || len(tokens) == 0 {
					b.Fatal(err)
				}
				addresses := make([]common.Address, len(tokens))
				for i, token := range tokens {
					addresses[i] = token.Address
				}
			}
		}
	})
	b.Run("balanceByKeys50", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			tokens, err := tm.GetTokensByKeys(keys)
			if err != nil || len(tokens) != len(keys) {
				b.Fatal(err, len(tokens))
			}
		}
	})
	b.Run("previouslyOwned50", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			tokens, err := tm.GetPreviouslyOwnedTokens()
			if err != nil || len(tokens[owner]) != len(keys) {
				b.Fatal(err)
			}
		}
	})
	b.Run("router", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			from, err := tm.GetTokenByKey(usdc)
			if err != nil || from == nil {
				b.Fatal(err)
			}
			to, err := tm.GetTokenByKey(usdt)
			if err != nil || to == nil {
				b.Fatal(err)
			}
			native, err := tm.GetNativeTokenForChain(1)
			if err != nil || native == nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("singleByChainAddress", func(b *testing.B) {
		address := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
		b.ReportAllocs()
		for b.Loop() {
			token, err := tm.GetTokenByChainAddress(1, address)
			if err != nil || token == nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("getAllTokens", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			tokens, err := tm.GetAllTokens()
			if err != nil || len(tokens) == 0 {
				b.Fatal(err)
			}
		}
	})
}
