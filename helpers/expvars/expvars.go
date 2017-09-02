package expvars

import (
	"expvar"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now().UTC()

// goroutines is an expvar.Func compliant
// wrapper for active goroutines counter
func goroutines() interface{} {
	return runtime.NumGoroutine()
}

// uptime is an expvar.Func compliant
// wrapper for uptime info
func uptime() interface{} {
	uptime := time.Since(startTime)
	return int64(uptime)
}

func init() {
	expvar.Publish("Goroutines", expvar.Func(goroutines))
	expvar.Publish("Uptime", expvar.Func(uptime))

	fmt.Println("Starting expvars")
	expvarsPort := ":10000" //os.Getenv("STATUS_EXPVARS")
	if expvarsPort != "" {
		go http.ListenAndServe(expvarsPort, nil)
	}

}
