// Package bgtrace provides temporary, greppable instrumentation for diagnosing
// background battery/CPU drain. It records whether the app is currently backgrounded
// and lets every periodic loop emit a marker each time it does work, so that an
// overnight background capture can be filtered down to exactly which loops kept running
// (and in which process) while the UI was hidden.
//
// All log lines use stable, greppable messages:
//   - "BG_LIFECYCLE"  - a background<->foreground transition (paused/resumed)
//   - "BG_ACTIVITY"   - a periodic loop fired (WARN while backgrounded = smoking gun)
//   - "BG_WATCHDOG"   - periodic goroutine-count + stack summary while backgrounded
//
// This package is diagnostic scaffolding: it is intended to be removed (or downgraded)
// once the pausability work lands. It imports only logutils to avoid import cycles.
package bgtrace

import (
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/logutils"
)

var background atomic.Bool

func logger() *zap.Logger { return logutils.ZapLogger().Named("bgtrace") }

// SetBackground records whether the app is currently backgrounded. It is called from the
// lifecycle bridge (PauseServices/ResumeServices) and read by Tick and the watchdog.
func SetBackground(b bool) { background.Store(b) }

// IsBackground reports the last known background state.
func IsBackground() bool { return background.Load() }

// Tick is called at the top of every periodic loop body (each ticker fire). While the
// app is backgrounded it logs at WARN so the line stands out; while foregrounded it logs
// at DEBUG to avoid flooding normal runs. `loop` is a stable identifier for the loop,
// `paused` is the owning service's view of its own pause state (so we can spot loops
// that fire despite believing they are paused), and `interval` is the tick period.
func Tick(loop string, paused bool, interval time.Duration) {
	fields := []zap.Field{
		zap.String("loop", loop),
		zap.Bool("background", background.Load()),
		zap.Bool("paused", paused),
		zap.Duration("interval", interval),
	}
	if background.Load() {
		logger().Warn("BG_ACTIVITY", fields...)
		return
	}
	logger().Debug("BG_ACTIVITY", fields...)
}

// LogLifecycle brackets a background/foreground transition. Called from the lifecycle
// bridge with the list of services being paused/resumed so the log unambiguously marks
// when each background episode starts and ends, and confirms resume actually happened.
func LogLifecycle(paused bool, services []string) {
	transition := "resumed"
	if paused {
		transition = "paused"
	}
	logger().Warn("BG_LIFECYCLE",
		zap.String("transition", transition),
		zap.Strings("services", services),
		zap.Int("serviceCount", len(services)),
	)
}

var watchdogOnce sync.Once

// StartWatchdog launches (once) a goroutine that, while the app is backgrounded, logs the
// live goroutine count and a compact, deduplicated stack summary every 10s. This is the
// low-touch safety net: it surfaces any periodic loop still running in the background even
// if we forgot to instrument it with Tick. It is a no-op while foregrounded. The goroutine
// lives for the remainder of the process; it is cheap when idle.
func StartWatchdog() {
	watchdogOnce.Do(func() {
		go watchdogLoop()
	})
}

func watchdogLoop() {
	defer common.LogOnPanic()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if !background.Load() {
			continue
		}
		buf := make([]byte, 1<<20)
		buf = buf[:runtime.Stack(buf, true)]
		logger().Warn("BG_WATCHDOG",
			zap.Int("numGoroutines", runtime.NumGoroutine()),
			zap.String("byState", summarizeStates(buf)),
			zap.String("byFrame", summarizeFrames(buf)),
		)
	}
}

// summarizeStates counts goroutines grouped by their scheduler state (the "[...]" in the
// "goroutine N [state]:" header line), e.g. "select:42, IO wait:3".
func summarizeStates(buf []byte) string {
	counts := map[string]int{}
	for _, block := range strings.Split(string(buf), "\n\n") {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) == 0 {
			continue
		}
		header := lines[0]
		open := strings.Index(header, "[")
		closeIdx := strings.Index(header, "]")
		if open >= 0 && closeIdx > open {
			counts[header[open+1:closeIdx]]++
		}
	}
	return topCounts(counts, 15)
}

// summarizeFrames counts goroutines grouped by the first non-runtime function frame in
// their stack (the application-level location where the goroutine is parked), e.g.
// "ens.(*Verifier).verifyLoop:1, communities.(*Manager).reeval:1". This is what reveals
// which named loops are alive while backgrounded.
func summarizeFrames(buf []byte) string {
	counts := map[string]int{}
	for _, block := range strings.Split(string(buf), "\n\n") {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		// Frame layout after the header: func line, file line, func line, file line, ...
		// Function lines are at odd indices (1, 3, 5, ...).
		fn := "unknown"
		for i := 1; i < len(lines); i += 2 {
			candidate := strings.TrimSpace(lines[i])
			if isRuntimeFrame(candidate) {
				continue
			}
			fn = trimFuncArgs(candidate)
			break
		}
		counts[shortFn(fn)]++
	}
	return topCounts(counts, 25)
}

func isRuntimeFrame(fn string) bool {
	return strings.HasPrefix(fn, "runtime.") ||
		strings.HasPrefix(fn, "internal/poll.") ||
		strings.HasPrefix(fn, "sync.") ||
		strings.HasPrefix(fn, "os.")
}

func trimFuncArgs(fn string) string {
	if i := strings.LastIndex(fn, "("); i >= 0 {
		return strings.TrimSpace(fn[:i])
	}
	return fn
}

// shortFn drops the long module path prefix, keeping the package-qualified name.
func shortFn(fn string) string {
	if i := strings.LastIndex(fn, "/"); i >= 0 {
		return fn[i+1:]
	}
	return fn
}

func topCounts(counts map[string]int, limit int) string {
	type kv struct {
		k string
		n int
	}
	arr := make([]kv, 0, len(counts))
	for k, n := range counts {
		arr = append(arr, kv{k, n})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].n != arr[j].n {
			return arr[i].n > arr[j].n
		}
		return arr[i].k < arr[j].k
	})
	var sb strings.Builder
	for i, e := range arr {
		if i >= limit {
			sb.WriteString(", ...")
			break
		}
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(e.k)
		sb.WriteString(":")
		sb.WriteString(strconv.Itoa(e.n))
	}
	return sb.String()
}
