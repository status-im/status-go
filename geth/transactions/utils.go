package transactions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime/debug"

	"github.com/pborman/uuid"
	"github.com/status-im/status-go/geth/signal"
)

const (
	// MessageIDKey is a key for message ID
	// This ID is required to track from which chat a given send transaction request is coming.
	MessageIDKey = contextKey("message_id")
)

type contextKey string // in order to make sure that our context key does not collide with keys from other packages

//ErrTxQueueRunFailure - error running transaction queue
var ErrTxQueueRunFailure = errors.New("error running transaction queue")

// haltOnPanic recovers from panic, logs issue, sends upward notification, and exits
func haltOnPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("%v: %v", ErrTxQueueRunFailure, r)

		// send signal up to native app
		signal.Send(signal.Envelope{
			Type: signal.EventNodeCrashed,
			Event: signal.NodeCrashEvent{
				Error: err,
			},
		})

		fatalf(err) // os.exit(1) is called internally
	}
}

// messageIDFromContext returns message id from context (if exists)
func messageIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if messageID, ok := ctx.Value(MessageIDKey).(string); ok {
		return messageID
	}

	return ""
}

// fatalf is used to halt the execution.
// When called the function prints stack end exits.
// Failure is logged into both StdErr and StdOut.
func fatalf(reason interface{}, args ...interface{}) {
	// decide on output stream
	w := io.MultiWriter(os.Stdout, os.Stderr)
	outf, _ := os.Stdout.Stat() // nolint: gas
	errf, _ := os.Stderr.Stat() // nolint: gas
	if outf != nil && errf != nil && os.SameFile(outf, errf) {
		w = os.Stderr
	}

	// find out whether error or string has been passed as a reason
	r := reflect.ValueOf(reason)
	if r.Kind() == reflect.String {
		fmt.Fprintf(w, "Fatal Failure: %v\n%v\n", reason.(string), args)
	} else {
		fmt.Fprintf(w, "Fatal Failure: %v\n", reason.(error))
	}

	debug.PrintStack()

	os.Exit(1)
}

// Create returns a transaction object.
func Create(ctx context.Context, args SendTxArgs) *QueuedTx {
	return &QueuedTx{
		ID:      uuid.New(),
		Context: ctx,
		Args:    args,
		Result:  make(chan Result, 1),
	}
}
