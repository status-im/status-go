package preflight

import (
	"testing"
)

func TestCheckOutbound(t *testing.T) {
	err := CheckOutbound()
	if err != nil {
		t.Fatal(err)
	}
}
