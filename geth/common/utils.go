package common

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"

	"github.com/ethereum/go-ethereum/log"
	"github.com/pborman/uuid"
	"github.com/status-im/status-go/static"
)

const (
	// MessageIDKey is a key for message ID
	// This ID is required to track from which chat a given send transaction request is coming.
	MessageIDKey = contextKey("message_id")
)

type contextKey string // in order to make sure that our context key does not collide with keys from other packages

// All general log messages in this package should be routed through this logger.
var logger = log.New("package", "status-go/geth/common")

// ImportTestAccount imports keystore from static resources, see "static/keys" folder
func ImportTestAccount(keystoreDir, accountFile string) error {
	// make sure that keystore folder exists
	if _, err := os.Stat(keystoreDir); os.IsNotExist(err) {
		os.MkdirAll(keystoreDir, os.ModePerm) // nolint: errcheck, gas
	}

	dst := filepath.Join(keystoreDir, accountFile)
	err := ioutil.WriteFile(dst, static.MustAsset("keys/"+accountFile), 0644)
	if err != nil {
		logger.Warn("cannot copy test account PK", "error", err)
	}

	return err
}

// MessageIDFromContext returns message id from context (if exists)
func MessageIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if messageID, ok := ctx.Value(MessageIDKey).(string); ok {
		return messageID
	}

	return ""
}

// Fatalf is used to halt the execution.
// When called the function prints stack end exits.
// Failure is logged into both StdErr and StdOut.
func Fatalf(reason interface{}, args ...interface{}) {
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

// CreateTransaction returns a transaction object.
func CreateTransaction(ctx context.Context, args SendTxArgs) *QueuedTx {
	return &QueuedTx{
		ID:      QueuedTxID(uuid.New()),
		Context: ctx,
		Args:    args,
		Result:  make(chan TransactionResult, 1),
	}
}
