package sentry

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
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
	require.NotNil(t, result)
	require.Len(t, result.Exception[0].Stacktrace.Frames, 1)
	require.Equal(t, "OtherFunction", result.Exception[0].Stacktrace.Frames[0].Function)

	// Verify that Modules and ServerName are empty
	require.Empty(t, result.Modules)
	require.Empty(t, result.ServerName)
}

func TestSentry(t *testing.T) {
	dsn := fmt.Sprintf("https://%s@sentry.example.com/%d",
		gofakeit.LetterN(32),
		gofakeit.Number(0, 1000),
	)
	context := gofakeit.LetterN(5)
	version := gofakeit.LetterN(5)

	var producedEvent *sentry.Event
	interceptor := func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		producedEvent = event
		return nil
	}

	err := Init(
		WithDSN(dsn),
		WithContext(context, version),
		func(o *sentry.ClientOptions) {
			o.BeforeSend = interceptor
		},
	)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		message := gofakeit.LetterN(5)
		err := errors.New(message)
		defer func() {
			recoveredError := recover().(error)
			require.NotNil(t, recoveredError)
			require.ErrorIs(t, err, recoveredError)
			wg.Done()
		}()
		defer Recover()
		panic(err)
	}()

	wg.Wait()
	require.NotNil(t, producedEvent)

	err = Close()
	require.NoError(t, err)
}
