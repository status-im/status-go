// This work is subject to the CC0 1.0 Universal (CC0 1.0) Public Domain Dedication
// license. Its contents can be found at:
// http://creativecommons.org/publicdomain/zero/1.0/

package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

const (
	AppName         = "go-bindata"
	AppVersionMajor = 4
	AppVersionMinor = 0
	AppVersionRev   = 2
)

var vsn, longVsn string
var vsnOnce, longVsnOnce sync.Once

func Version() string {
	vsnOnce.Do(func() {
		vsn = fmt.Sprintf(`go-bindata version %d.%d.%d`, AppVersionMajor, AppVersionMinor, AppVersionRev)
	})
	return vsn
}

func LongVersion() string {
	longVsnOnce.Do(func() {
		longVsn = fmt.Sprintf(`%s %d.%d.%d (Go runtime %s).
Copyright (c) 2010-2015, Jim Teeuwen.
Copyright (c) 2017-%d, Kevin Burke.`, AppName,
			AppVersionMajor, AppVersionMinor, AppVersionRev,
			runtime.Version(), time.Now().Year())
	})
	return longVsn
}
