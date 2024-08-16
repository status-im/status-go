package package_3

import (
	"testing"
	"github.com/status-im/status-go/flaky-packages/utils"
)

func TestSleep(t *testing.T) {
	utils.Sleep()
	utils.Foo()
}
