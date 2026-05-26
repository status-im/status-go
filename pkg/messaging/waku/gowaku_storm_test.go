package wakuv2

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/connection"
)

// TestShouldFireConnectionChanged validates the idempotency guard that
// suppresses no-op ConnectionChanged calls from checkForConnectionChanges.
//
// Without this guard, every libp2p peer event triggers a full filter
// resubscription cascade in lightClient mode (gowaku.go:1280 -> ConnectionChanged
// -> filterManager.OnConnectionStatusChange -> resubscribeAllSubscriptions ->
// per-peer libp2p dials). The dial completions emit fresh peer events that
// loop back into the trigger — a self-amplifying loop that accumulates
// thousands of orphan goroutines and pins statusgo at >2 cores of CPU.
//
// The guard fires on the very first observation (so downstream consumers like
// the LightClient FilterManager get initialized even when peers connect before
// the first poll) and thereafter only on a real change in Offline or Type.
// Expensive is intentionally excluded because checkForConnectionChanges has
// no OS visibility to learn about metered-connection changes — those flow in
// through the explicit mobile.ConnectionChange path.
func TestShouldFireConnectionChanged(t *testing.T) {
	wifi := connection.NewType("wifi")
	cellular := connection.NewType("cellular")
	unknown := connection.NewType("")

	cases := []struct {
		name            string
		prev            connection.State
		next            connection.State
		prevInitialized bool
		expect          bool
	}{
		{
			name:            "first observation, peers already online — fires (FilterManager init)",
			prev:            connection.State{}, // zero-value
			next:            connection.State{Offline: false, Type: unknown},
			prevInitialized: false,
			expect:          true,
		},
		{
			name:            "first observation, no peers yet — fires",
			prev:            connection.State{},
			next:            connection.State{Offline: true, Type: unknown},
			prevInitialized: false,
			expect:          true,
		},
		{
			name:            "no change — online wifi stays online wifi",
			prev:            connection.State{Offline: false, Type: wifi},
			next:            connection.State{Offline: false, Type: wifi},
			prevInitialized: true,
			expect:          false,
		},
		{
			name:            "offline → online",
			prev:            connection.State{Offline: true, Type: unknown},
			next:            connection.State{Offline: false, Type: unknown},
			prevInitialized: true,
			expect:          true,
		},
		{
			name:            "online → offline",
			prev:            connection.State{Offline: false, Type: wifi},
			next:            connection.State{Offline: true, Type: wifi},
			prevInitialized: true,
			expect:          true,
		},
		{
			name:            "type wifi → cellular (network switch)",
			prev:            connection.State{Offline: false, Type: wifi},
			next:            connection.State{Offline: false, Type: cellular},
			prevInitialized: true,
			expect:          true,
		},
		{
			name:            "type cellular → wifi (network switch)",
			prev:            connection.State{Offline: false, Type: cellular},
			next:            connection.State{Offline: false, Type: wifi},
			prevInitialized: true,
			expect:          true,
		},
		{
			name:            "no change — both offline same type",
			prev:            connection.State{Offline: true, Type: unknown},
			next:            connection.State{Offline: true, Type: unknown},
			prevInitialized: true,
			expect:          false,
		},
		{
			name:            "expensive flag toggled in isolation — ignored",
			prev:            connection.State{Offline: false, Type: wifi, Expensive: false},
			next:            connection.State{Offline: false, Type: wifi, Expensive: true},
			prevInitialized: true,
			expect:          false,
		},
		{
			name:            "expensive AND online state change — fires",
			prev:            connection.State{Offline: true, Type: wifi, Expensive: true},
			next:            connection.State{Offline: false, Type: wifi, Expensive: false},
			prevInitialized: true,
			expect:          true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, err := New(nil, nil, nil, nil, nil, nil)
			require.NoError(t, err)
			if tc.prevInitialized {
				w.stateMu.Lock()
				w.state = tc.prev
				w.stateInitialized = true
				w.stateMu.Unlock()
			}
			require.Equal(t, tc.expect, w.shouldFireConnectionChanged(tc.next))
		})
	}
}

// TestIsNetworkSwitchEvent guards the predicate that gates the
// DisconnectAllPeers branch in handleNetworkChangeFromApp. Pre-fix, the
// switch branch fired on the very first ConnectionChange after Waku
// construction because w.state.Type starts at zero-value connectionUnknown,
// and the first observation (e.g. wifi) compared as "type changed while
// online" — so the bootstrap peers that just connected were torn down,
// stranding the node on the 6 DNS-bootstrap peers until the next real
// network event (airplane toggle) gave the discv5 DHT enough time to fill.
//
// The fix is to distinguish "state not yet set" from "state observed with
// Offline=false": the first call has no prior state to compare against, so
// a Type difference vs the zero value is NOT a switch event.
func TestIsNetworkSwitchEvent(t *testing.T) {
	wifi := connection.NewType("wifi")
	cellular := connection.NewType("cellular")
	unknown := connection.NewType("")

	cases := []struct {
		name            string
		prev            connection.State
		next            connection.State
		prevInitialized bool
		expect          bool
	}{
		{
			name:            "cold start: not initialized + first online state — NOT a switch",
			prev:            connection.State{}, // zero-value: Offline=false, Type=unknown
			next:            connection.State{Offline: false, Type: wifi},
			prevInitialized: false,
			expect:          false,
		},
		{
			name:            "cold start: not initialized + first offline state — NOT a switch",
			prev:            connection.State{},
			next:            connection.State{Offline: true, Type: unknown},
			prevInitialized: false,
			expect:          false,
		},
		{
			name:            "real switch: wifi to cellular while online — IS a switch",
			prev:            connection.State{Offline: false, Type: wifi},
			next:            connection.State{Offline: false, Type: cellular},
			prevInitialized: true,
			expect:          true,
		},
		{
			name:            "real switch: cellular to wifi while online — IS a switch",
			prev:            connection.State{Offline: false, Type: cellular},
			next:            connection.State{Offline: false, Type: wifi},
			prevInitialized: true,
			expect:          true,
		},
		{
			name:            "no change: same type online — NOT a switch",
			prev:            connection.State{Offline: false, Type: wifi},
			next:            connection.State{Offline: false, Type: wifi},
			prevInitialized: true,
			expect:          false,
		},
		{
			name:            "going offline (prev online, next offline) — NOT a switch (handled by offline branch)",
			prev:            connection.State{Offline: false, Type: wifi},
			next:            connection.State{Offline: true, Type: wifi},
			prevInitialized: true,
			expect:          false,
		},
		{
			name:            "coming online (prev offline, next online) — NOT a switch (handled by online transition)",
			prev:            connection.State{Offline: true, Type: wifi},
			next:            connection.State{Offline: false, Type: wifi},
			prevInitialized: true,
			expect:          false,
		},
		{
			name:            "coming online with type change — NOT a switch (going-online edge owns this)",
			prev:            connection.State{Offline: true, Type: wifi},
			next:            connection.State{Offline: false, Type: cellular},
			prevInitialized: true,
			expect:          false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expect, isNetworkSwitchEvent(tc.prev, tc.next, tc.prevInitialized))
		})
	}
}

// TestDnsDiscoveryBackoff guards the exponential backoff schedule used to
// retry enrtree:// DNS bootstrap discovery when it fails at cold start
// (commonly: Android's DNS resolver isn't wired up yet, so the lookup goes
// to ::1:53 and returns "connection refused"). Pre-fix, the failure was
// logged once and the bootstrap was abandoned — the node stayed on the 6
// DNS-cached store peers until a real network event triggered a retry.
//
// The schedule: 0 (no initial wait), then 1s, 2s, 4s, 8s, 16s, 32s,
// capped at 60s thereafter. Capped at attempt 30 to avoid shift overflow.
func TestDnsDiscoveryBackoff(t *testing.T) {
	cases := []struct {
		attempt int
		expect  time.Duration
	}{
		{attempt: 0, expect: 0}, // sentinel: no wait
		{attempt: 1, expect: 1 * time.Second},
		{attempt: 2, expect: 2 * time.Second},
		{attempt: 3, expect: 4 * time.Second},
		{attempt: 4, expect: 8 * time.Second},
		{attempt: 5, expect: 16 * time.Second},
		{attempt: 6, expect: 32 * time.Second},
		{attempt: 7, expect: 60 * time.Second},    // cap kicks in
		{attempt: 50, expect: 60 * time.Second},   // stays at cap
		{attempt: 1000, expect: 60 * time.Second}, // overflow-safe
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("attempt=%d", tc.attempt), func(t *testing.T) {
			require.Equal(t, tc.expect, dnsDiscoveryBackoff(tc.attempt))
		})
	}
}
