package sentry

import (
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
)

func TestTrimStacktrace(t *testing.T) {
	tests := []struct {
		name       string
		stacktrace *sentry.Stacktrace
		expected   []sentry.Frame
	}{
		{
			name:       "nil stacktrace",
			stacktrace: nil,
			expected:   nil,
		},
		{
			name: "single frame",
			stacktrace: &sentry.Stacktrace{
				Frames: []sentry.Frame{
					{Module: "github.com/status-im/status-go/internal/sentry", Function: "Recover"},
				},
			},
			expected: []sentry.Frame{
				{Module: "github.com/status-im/status-go/internal/sentry", Function: "Recover"},
			},
		},
		{
			name: "multiple frames with matching filters",
			stacktrace: &sentry.Stacktrace{
				Frames: []sentry.Frame{
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc"},
					{Module: "github.com/status-im/status-go/mobile/callog", Function: "Recover"},
					{Module: "github.com/status-im/status-go/internal/sentry", Function: "RecoverError"},
				},
			},
			expected: []sentry.Frame{
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc"},
			},
		},
		{
			name: "matching module but not function",
			stacktrace: &sentry.Stacktrace{
				Frames: []sentry.Frame{
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc"},
					{Module: "github.com/status-im/status-go/internal/sentry", Function: "Init"},
				},
			},
			expected: []sentry.Frame{
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc"},
				{Module: "github.com/status-im/status-go/internal/sentry", Function: "Init"},
			},
		},
		{
			name: "multiple frames without matching filters",
			stacktrace: &sentry.Stacktrace{
				Frames: []sentry.Frame{
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc"},
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc2"},
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc3"},
				},
			},
			expected: []sentry.Frame{
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc"},
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc2"},
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc3"},
			},
		},
		{
			name: "matching filters only at the end",
			stacktrace: &sentry.Stacktrace{
				Frames: []sentry.Frame{
					{Module: "github.com/status-im/status-go/internal/sentry", Function: "Recover"},
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc"},
					{Module: "github.com/status-im/status-go/common", Function: "LogOnPanic"},
				},
			},
			expected: []sentry.Frame{
				{Module: "github.com/status-im/status-go/internal/sentry", Function: "Recover"},
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc"},
			},
		},
		{
			name: "remove at most 2 frames",
			stacktrace: &sentry.Stacktrace{
				Frames: []sentry.Frame{
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc1"},
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc2"},
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc3"},
					{Module: "github.com/status-im/status-go/internal/sentry", Function: "Recover"},
					{Module: "github.com/status-im/status-go/mobile/callog", Function: "Recover"},
					{Module: "github.com/status-im/status-go/common", Function: "LogOnPanic"},
				},
			},
			expected: []sentry.Frame{
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc1"},
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc2"},
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc3"},
				{Module: "github.com/status-im/status-go/internal/sentry", Function: "Recover"},
			},
		},
		{
			name: "break if non-matching frame found",
			stacktrace: &sentry.Stacktrace{
				Frames: []sentry.Frame{
					{Module: "github.com/status-im/status-go/mobile/callog", Function: "Recover"},
					{Module: "github.com/status-im/status-go/internal/sentry", Function: "RecoverError"},
					{Module: "github.com/status-im/status-go/other", Function: "OtherFunc1"},
				},
			},
			expected: []sentry.Frame{
				{Module: "github.com/status-im/status-go/mobile/callog", Function: "Recover"},
				{Module: "github.com/status-im/status-go/internal/sentry", Function: "RecoverError"},
				{Module: "github.com/status-im/status-go/other", Function: "OtherFunc1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimStacktrace(tt.stacktrace)
			if tt.stacktrace != nil {
				require.Equal(t, tt.expected, tt.stacktrace.Frames)
			}
		})
	}
}
