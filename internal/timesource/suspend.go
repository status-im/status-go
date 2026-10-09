package timesource

import "time"

// suspendDetectionThreshold is how far the wall clock may run ahead of the
// monotonic clock between two readings before it is attributed to a system
// suspend rather than clock adjustments.
const suspendDetectionThreshold = time.Second

// SuspendedBetween reports whether the system was suspended between two
// time.Now() readings. Go's monotonic clock, which drives timers, tickers and
// durations between such readings, does not advance while the system is
// suspended (e.g. on Linux and macOS), whereas the wall clock does. Readings
// without a monotonic component (e.g. after Round(0)) never report a suspend.
func SuspendedBetween(prev, now time.Time) bool {
	return wallClockAhead(now.Round(0).Sub(prev.Round(0)), now.Sub(prev))
}

func wallClockAhead(wallElapsed, monotonicElapsed time.Duration) bool {
	return wallElapsed-monotonicElapsed > suspendDetectionThreshold
}
