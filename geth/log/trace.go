package log

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// vars
var (
	stackSize = 1 << 6
	question  = "???"
)

// Trace defines a structure which contains the stack, start and endtime
// on a given from a trace call to trace a given call with stack details
// and execution time.
type Trace struct {
	File       string    `json:"file"`
	Package    string    `json:"Package"`
	Function   string    `json:"function"`
	Comments   string    `json:"comments"`
	LineNumber int       `json:"line_number"`
	BeginStack []byte    `json:"begin_stack"`
	EndStack   []byte    `json:"end_stack"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
}

// NewTrace returns a Trace object which is used to track the execution and
// stack details of a given trace call.
func NewTrace(comments string) *Trace {
	trace := make([]byte, stackSize)
	trace = trace[:runtime.Stack(trace, false)]

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
	}

	var pkg, pkgFile string
	pkgFileBase := file

	if file != "???" {
		pkgPieces := strings.SplitAfter(pkgFileBase, "/src/")
		if len(pkgPieces) > 1 {
			pkgFileBase = pkgPieces[1]
		}

		pkg = filepath.Dir(pkgFileBase)
		pkgFile = filepath.Base(pkgFileBase)
	}

	return &Trace{
		Package:    pkg,
		LineNumber: line,
		BeginStack: trace,
		Comments:   comments,
		StartTime:  time.Now(),
		File:       pkgFile,
		Function:   getFunctionName(3),
	}

}

// NewTraceWithCallDepth returns a Trace object which is used to track the execution and
// stack details of a given trace call.
func NewTraceWithCallDepth(depth int, comments string) *Trace {
	trace := make([]byte, stackSize)
	trace = trace[:runtime.Stack(trace, false)]

	_, file, line, ok := runtime.Caller(depth)
	if !ok {
		file = question
	}

	var pkg, pkgFile string
	pkgFileBase := file

	if file != question {
		pkgPieces := strings.SplitAfter(pkgFileBase, "/src/")
		if len(pkgPieces) > 1 {
			pkgFileBase = pkgPieces[1]
		}

		pkg = filepath.Dir(pkgFileBase)
		pkgFile = filepath.Base(pkgFileBase)
	}

	return &Trace{
		Package:    pkg,
		LineNumber: line,
		BeginStack: trace,
		File:       pkgFile,
		Comments:   comments,
		StartTime:  time.Now(),
		Function:   getFunctionName(3),
	}
}

// String returns the giving trace timestamp for the execution time.
func (t *Trace) String() string {
	return fmt.Sprintf("[Total=%+q, Start=%+q, End=%+q]", t.EndTime.Sub(t.StartTime), t.StartTime, t.EndTime)
}

// End stops the trace, captures the current stack trace and returns the
// entry related to the trace.
func (t *Trace) End() *Trace {
	trace := make([]byte, stackSize)
	trace = trace[:runtime.Stack(trace, false)]
	t.EndStack = trace
	t.EndTime = time.Now()
	return t
}

// getFunctionName returns the caller of the function that called it :)
func getFunctionName(depth int) string {

	// we get the callers as uintptrs - but we just need 1
	fpcs := make([]uintptr, 1)

	// skip 3 levels to get to the caller of whoever called Caller()
	n := runtime.Callers(depth, fpcs)
	if n == 0 {
		return "Unknown()" // proper error her would be better
	}

	// get the info of the actual function that's in the pointer
	fun := runtime.FuncForPC(fpcs[0] - 1)
	if fun == nil {
		return "Unknown()" // proper error her would be better
	}

	// return its name
	return fun.Name()
}
