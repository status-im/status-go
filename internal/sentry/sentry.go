package sentry

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/status-im/status-go/internal/version"
)

func Init(opts ...Option) error {
	cfg := defaultConfig()
	applyOptions(cfg, opts...)
	return sentry.Init(*cfg)
}

func MustInit(options ...Option) {
	if err := Init(options...); err != nil {
		panic(err)
	}
}

func Recover() {
	err := recover()
	if err == nil {
		return
	}
	RecoverError(err)
	panic(err)
}

func RecoverError(err interface{}) {
	sentry.CurrentHub().Recover(err)
	sentry.Flush(time.Second * 5)
}

func defaultConfig() *sentry.ClientOptions {
	return &sentry.ClientOptions{
		EnableTracing:  false,
		Debug:          true, // FIXME: Set to false
		SendDefaultPII: false,
		Release:        version.Version(),
		Environment:    Environment(),
		Tags:           make(map[string]string),
		BeforeSend:     beforeSend,
	}
}

func beforeSend(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	event.Modules = nil   // Clear modules as we know all dependencies by commit hash
	event.ServerName = "" // Clear server name as it might be sensitive

	// Cleanup the stacktrace from last Recover/LogOnPanic frames
	for _, exception := range event.Exception {
		trimStacktrace(exception.Stacktrace)
	}

	// FIXME: Remove this temporary printing
	eventJSON, _ := json.Marshal(event)
	hintJSON, _ := json.Marshal(hint)
	fmt.Println("publish sentry event")
	fmt.Println(string(eventJSON))
	fmt.Println(string(hintJSON))
	return event
}

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
		Functions: []string{"Call.func1"},
	},
}

func trimStacktrace(stacktrace *sentry.Stacktrace) {
	if stacktrace == nil {
		return
	}

	if len(stacktrace.Frames) <= 1 {
		return
	}

	// Trim max 2 frames from the end
	for i := len(stacktrace.Frames) - 1; i >= len(stacktrace.Frames)-3; i-- {
		frame := stacktrace.Frames[i]
		for _, filter := range stacktraceFilters {
			if frame.Module != filter.Module {
				continue
			}
			if !slices.Contains(filter.Functions, frame.Function) {
				continue
			}
			stacktrace.Frames = stacktrace.Frames[:i]
		}
	}
}
