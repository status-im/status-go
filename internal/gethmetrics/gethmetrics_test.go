package gethmetrics

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/metrics"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type echoService struct{}

func (echoService) Echo(s string) string { return s }
func (echoService) Fail() error          { return errors.New("fail") }
func (echoService) Ping() string         { return "pong" }

type GethMetricsSuite struct {
	suite.Suite
}

func TestGethMetricsSuite(t *testing.T) {
	suite.Run(t, new(GethMetricsSuite))
}

func (s *GethMetricsSuite) TestRPCCallsDoNotRetainServeTimeHistograms() {
	defer func(rate int) { runtime.MemProfileRate = rate }(runtime.MemProfileRate)
	runtime.MemProfileRate = 1

	server := gethrpc.NewServer()
	s.Require().NoError(server.RegisterName("test", echoService{}))
	client := gethrpc.DialInProc(server)
	defer client.Close()
	defer server.Stop()

	var echoed, pong string
	s.Require().NoError(client.Call(&echoed, "test_echo", "hi"))
	s.Require().Equal("hi", echoed)
	s.Require().NoError(client.Call(&pong, "test_ping"))
	s.Require().Equal("pong", pong)
	s.Require().EqualError(client.Call(nil, "test_fail"), "fail")

	s.Require().Zero(liveBytesAllocatedBy("rpc.updateServeTimeHistogram"))
}

func (s *GethMetricsSuite) TestOtherMetricsAreStillRegistered() {
	counter := metrics.GetOrRegisterCounter("gethmetrics/test/counter", nil)
	s.Require().Same(counter, metrics.GetOrRegisterCounter("gethmetrics/test/counter", nil))
	s.Require().Same(counter, metrics.DefaultRegistry.Get("gethmetrics/test/counter"))
}

func liveBytesAllocatedBy(function string) int64 {
	runtime.GC()
	runtime.GC()

	n, _ := runtime.MemProfile(nil, false)
	records := make([]runtime.MemProfileRecord, n+64)
	n, _ = runtime.MemProfile(records, false)

	var live int64
	for _, r := range records[:n] {
		frames := runtime.CallersFrames(r.Stack())
		for {
			f, more := frames.Next()
			if strings.HasSuffix(f.Function, function) {
				live += r.InUseBytes()
				break
			}
			if !more {
				break
			}
		}
	}
	return live
}
