package api

import "testing"

func TestConnectionType(t *testing.T) {
	c := newConnectionType("wifi")
	if c != connectionWifi {
		t.Fatalf("Wrong connection type: %v", c)
	}
	c = newConnectionType("cellular")
	if c != connectionCellular {
		t.Fatalf("Wrong connection type: %v", c)
	}
	c = newConnectionType("bluetooth")
	if c != connectionUnknown {
		t.Fatalf("Wrong connection type: %v", c)
	}
}

func TestConnectionState(t *testing.T) {
	tests := []struct {
		name     string
		state    connectionState
		expected string
	}{
		{
			"zero value",
			connectionState{},
			"unknown",
		},
		{
			"offline",
			connectionState{Offline: true},
			"offline",
		},
		{
			"wifi",
			connectionState{Type: connectionWifi},
			"wifi",
		},
		{
			"wifi tethered",
			connectionState{Type: connectionWifi, Expensive: true},
			"wifi (expensive)",
		},
		{
			"unknown",
			connectionState{Type: connectionUnknown},
			"unknown",
		},
		{
			"cellular",
			connectionState{Type: connectionCellular},
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
