//go:build !ios
// +build !ios

package watchdog

import (
	"fmt"
	"time"

	"github.com/elastic/gosigar"
)

var (
	sysmemFn = (*gosigar.Mem).Get
)

// SystemDriven starts a singleton system-driven watchdog.
//
// The system-driven watchdog keeps a threshold, above which GC will be forced.
// The watchdog polls the system utilization at the specified frequency. When
// the actual utilization exceeds the threshold, a GC is forced.
//
// This threshold is calculated by querying the policy every time that GC runs,
// either triggered by the runtime, or forced by us.
func SystemDriven(limit uint64, frequency time.Duration, policyCtor PolicyCtor) (err error, stopFn func()) {
	if limit == 0 {
		var sysmem gosigar.Mem
		if err := sysmemFn(&sysmem); err != nil {
			return fmt.Errorf("failed to get system memory stats: %w", err), nil
		}
		limit = sysmem.Total
	}

	policy, err := policyCtor(limit)
	if err != nil {
		return fmt.Errorf("failed to construct policy with limit %d: %w", limit, err), nil
	}

	if err := start(UtilizationSystem); err != nil {
		return err, nil
	}

	_watchdog.wg.Add(1)
	var sysmem gosigar.Mem
	go pollingWatchdog(policy, frequency, limit, func() (uint64, error) {
		if err := sysmemFn(&sysmem); err != nil {
			return 0, err
		}
		return sysmem.ActualUsed, nil
	})

	return nil, stop
}
