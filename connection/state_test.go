package connection

import "testing"

func TestConnectionType(t *testing.T) {
	c := NewConnectionType("wifi")
	if c != connectionWifi {
		t.Fatalf("Wrong connection type: %v", c)
	}
	c = NewConnectionType("cellular")
	if c != connectionCellular {
		t.Fatalf("Wrong connection type: %v", c)
	}
	c = NewConnectionType("bluetooth")
	if c != connectionUnknown {
		t.Fatalf("Wrong connection type: %v", c)
	}
}

func TestState(t *testing.T) {
	tests := []struct {
		name     string
		state    State
		expected string
	}{
		{
			"zero value",
			State{},
			"unknown",
		},
		{
			"offline",
			State{Offline: true},
			"offline",
		},
		{
			"wifi",
			State{Type: connectionWifi},
			"wifi",
		},
		{
			"wifi tethered",
			State{Type: connectionWifi, Expensive: true},
			"wifi (expensive)",
		},
		{
			"unknown",
			State{Type: connectionUnknown},
			"unknown",
		},
		{
			"cellular",
			State{Type: connectionCellular},
			"cellular",
		},
	}

	for _, test := range tests {
		str := test.state.String()
		if str != test.expected {
			t.Fatalf("Expected String() to return '%s', got '%s'", test.expected, str)
		}
	}
}
