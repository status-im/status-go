package sentry

import (
	"slices"

	"github.com/getsentry/sentry-go"
)

var stacktraceFilters = []struct {
	Module    string
	Functions []string
}{
	{
		Module:    "github.com/status-im/status-go/internal/sentry",
		Functions: []string{"Recover", "RecoverError"},
	},
	{
		Module:    "github.com/status-im/status-go/common",
		Functions: []string{"LogOnPanic"},
	},
	{
		Module:    "github.com/status-im/status-go/mobile/callog",
		Functions: []string{"Recover"},
	},
}

func trimStacktrace(stacktrace *sentry.Stacktrace) {
	if stacktrace == nil {
		return
	}

	if len(stacktrace.Frames) <= 1 {
		return
	}

	trim := 0

	// Trim max 2 frames from the end
	for i := len(stacktrace.Frames) - 1; i >= 0; i-- {
		if !matchFilter(stacktrace.Frames[i]) {
			// break as soon as we find a frame that doesn't match
			break
		}

		trim++
		if trim == 2 {
			break
		}
	}

	stacktrace.Frames = stacktrace.Frames[:len(stacktrace.Frames)-trim]
}

func matchFilter(frame sentry.Frame) bool {
	for _, filter := range stacktraceFilters {
		if frame.Module != filter.Module {
			continue
		}
		if !slices.Contains(filter.Functions, frame.Function) {
			continue
		}
		return true
	}
	return false
}
