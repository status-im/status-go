package topics

import (
	"testing"

	"github.com/status-im/status-go/geth/params"
	"github.com/stretchr/testify/assert"
)

func TestTopicFlags(t *testing.T) {
	type testCase struct {
		shortcut string
		flags    []string
		expected TopicFlag
	}

	for _, tc := range []testCase{
		{
			shortcut: "single",
			flags:    []string{"whisper"},
			expected: TopicFlag{"whisper"},
		},
		{
			shortcut: "multiple",
			flags:    []string{"whisper", "les"},
			expected: TopicFlag{"whisper", "les"},
		},
		{
			shortcut: "corrupted",
			flags:    []string{" whisper ", "les "},
			expected: TopicFlag{"whisper", "les"},
		},
	} {
		t.Run(tc.shortcut, func(t *testing.T) {
			result := TopicFlag{}
			for _, flag := range tc.flags {
				assert.NoError(t, result.Set(flag))
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestTopicLimitsFlag(t *testing.T) {
	type testCase struct {
		shortcut  string
		flags     []string
		expected  TopicLimitsFlag
		expectErr bool
	}
	for _, tc := range []testCase{
		{
			shortcut: "single",
			flags:    []string{"whisper=1,1"},
			expected: TopicLimitsFlag{"whisper": params.NewLimits(1, 1)},
		},
		{
			shortcut: "multiple",
			flags:    []string{"whisper=1,1", "les=2,3"},
			expected: TopicLimitsFlag{"whisper": params.NewLimits(1, 1), "les": params.NewLimits(2, 3)},
		},
		{
			shortcut: "corrupted",
			flags:    []string{" whisper=1,1 ", " les=2,3"},
			expected: TopicLimitsFlag{"whisper": params.NewLimits(1, 1), "les": params.NewLimits(2, 3)},
		},
		{
			shortcut:  "badseparator",
			flags:     []string{"whisper==1,1"},
			expected:  TopicLimitsFlag{},
			expectErr: true,
		},
		{
			shortcut:  "singlelimit",
			flags:     []string{"whisper=1"},
			expected:  TopicLimitsFlag{},
			expectErr: true,
		},
		{
			shortcut:  "minnotanumber",
			flags:     []string{"whisper=a,1"},
			expected:  TopicLimitsFlag{},
			expectErr: true,
		},
		{
			shortcut:  "maxnotanumber",
			flags:     []string{"whisper=1,a"},
			expected:  TopicLimitsFlag{},
			expectErr: true,
		},
	} {
		t.Run(tc.shortcut, func(t *testing.T) {
			result := TopicLimitsFlag{}
			for _, flag := range tc.flags {
				err := result.Set(flag)
				if tc.expectErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
			assert.Equal(t, tc.expected, result)
		})
	}
}
