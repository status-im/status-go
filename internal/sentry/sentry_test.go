package sentry

import (
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
)

func TestBeforeSend(t *testing.T) {
	// Initialize a sample event with a stacktrace
	event := &sentry.Event{
		Modules:    map[string]string{"example": "1.0.0"},
		ServerName: "test-server",
		Exception: []sentry.Exception{
			{
				Stacktrace: &sentry.Stacktrace{
					Frames: []sentry.Frame{
						{Module: "github.com/status-im/status-go/other", Function: "OtherFunction"},
						{Module: "github.com/status-im/status-go/internal/sentry", Function: "Recover"},
						{Module: "github.com/status-im/status-go/internal/sentry", Function: "RecoverError"},
					},
				},
			},
		},
	}

	// Call the beforeSend function
	result := beforeSend(event, nil)

	// Verify that the stacktrace frames are correctly trimmed
	assert.NotNil(t, result)
	assert.Len(t, result.Exception[0].Stacktrace.Frames, 1)
	assert.Equal(t, "OtherFunction", result.Exception[0].Stacktrace.Frames[0].Function)

	// Verify that Modules and ServerName are empty
	assert.Empty(t, result.Modules)
	assert.Empty(t, result.ServerName)
}
