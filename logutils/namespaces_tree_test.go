package logutils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestBuildNamespacesTree(t *testing.T) {
	tests := []struct {
		name       string
		namespaces string
		minLevel   zapcore.Level
		err        error
	}{
		{
			name:       "valid namespaces",
			namespaces: "namespace1:debug,namespace2.namespace3:error",
			minLevel:   zapcore.DebugLevel,
			err:        nil,
		},
		{
			name:       "invalid format",
			namespaces: "namespace1:debug,namespace2.namespace3",
			minLevel:   zapcore.DebugLevel,
			err:        errInvalidNamespacesFormat,
		},
		{
			name:       "empty namespaces",
			namespaces: "",
			minLevel:   zapcore.InvalidLevel,
			err:        nil,
		},
		{
			name:       "invalid level",
			namespaces: "namespace1:invalid",
			minLevel:   zapcore.InvalidLevel,
			err:        errInvalidNamespacesFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree, err := buildNamespacesTree(tt.namespaces)
			require.ErrorIs(t, err, tt.err)
			if tt.err == nil {
				require.NotNil(t, tree)
			}
		})
	}
}

func TestLevelFor(t *testing.T) {
	tree, err := buildNamespacesTree("namespace1:error,namespace1.namespace2:debug,namespace1.namespace2.namespace3:info,namespace3.namespace4:warn")
	require.NoError(t, err)
	require.NotNil(t, tree)

	tests := []struct {
		name     string
		input    string
		expected zapcore.Level
	}{
		{
			name:     "exact match 1",
			input:    "namespace1",
			expected: zapcore.ErrorLevel,
		},
		{
			name:     "exact match 2",
			input:    "namespace1.namespace2",
			expected: zapcore.DebugLevel,
		},
		{
			name:     "exact match 3",
			input:    "namespace3.namespace4",
			expected: zapcore.WarnLevel,
		},
		{
			name:     "exact match 3",
			input:    "namespace1.namespace2.namespace3",
			expected: zapcore.InfoLevel,
		},
		{
			name:     "partial match 1",
			input:    "namespace1.unregistered",
			expected: zapcore.ErrorLevel,
		},
		{
			name:     "partial match 2",
			input:    "namespace1.namespace2.unregistered",
			expected: zapcore.DebugLevel,
		},
		{
			name:     "no match 1",
			input:    "namespace2",
			expected: zapcore.InvalidLevel,
		},
		{
			name:     "no match 2",
			input:    "namespace3",
			expected: zapcore.InvalidLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := tree.LevelFor(tt.input)
			require.Equal(t, tt.expected, level)
		})
	}
}
