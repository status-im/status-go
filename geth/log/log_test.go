// log_test
package log

import (
	"testing"
)

func TestLogger(t *testing.T) {

	t.Log("Testing log package..")

	Trace("Trace Message")
	Debug("Debug Message")
	Info("Info Message")
	Warn("Warn Message")
	Error("Error Message")
	Crit("Crit Message")
}
