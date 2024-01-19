package log

import (
	"runtime"
	"strings"
	"sync"
)

func getSingleCallerPc(skip int) uintptr {
	var pc [1]uintptr
	runtime.Callers(skip+2, pc[:])
	return pc[0]
}

type Loc struct {
	Package  string
	Function string
	File     string
	Line     int
}

func locFromPc(pc uintptr) Loc {
	f, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	lastSlash := strings.LastIndexByte(f.Function, '/')
	firstDot := strings.IndexByte(f.Function[lastSlash+1:], '.')
	return Loc{
		Package:  f.Function[:lastSlash+1+firstDot],
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}

var pcToLoc sync.Map

func getMsgLogLoc(msg Msg) Loc {
	var pc [1]uintptr
	msg.Callers(1, pc[:])
	locIf, ok := pcToLoc.Load(pc[0])
	if ok {
		return locIf.(Loc)
	}
	loc := locFromPc(pc[0])
	pcToLoc.Store(pc[0], loc)
	return loc
}
