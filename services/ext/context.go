package ext

import (
	"context"
	"time"

	"github.com/status-im/status-go/db"
)

// ContextKey is a type used for keys in ext Context.
type ContextKey struct {
	Name string
}

// NewContextKey returns new ContextKey instance.
func NewContextKey(name string) ContextKey {
	return ContextKey{Name: name}
}

var (
	timeKey = NewContextKey("time")
)

// NewContext creates Context with all required fields.
func NewContext(ctx context.Context, source TimeSource, storage db.Storage) Context {
	ctx = context.WithValue(ctx, timeKey, source)
	return Context{ctx}
}

// TimeSource is a type used for current time.
type TimeSource func() time.Time

// Context provides access to request-scoped values.
type Context struct {
	context.Context
}

// Time returns current time using time function associated with this request.
func (c Context) Time() time.Time {
	return c.Value(timeKey).(TimeSource)()
}
