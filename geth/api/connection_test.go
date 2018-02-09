package api

import "testing"

func TestConnectionType(t *testing.T) {
	c := NewConnectionType("wifi")
	if c != ConnectionWifi {
		t.Fatalf("Wrong connection type: %s", c)
	}
	c = NewConnectionType("cellular")
	if c != ConnectionCellular {
		t.Fatalf("Wrong connection type: %s", c)
	}
	c = NewConnectionType("bluetooth")
	if c != ConnectionUnknown {
		t.Fatalf("Wrong connection type: %s", c)
	}
}

func TestConnectionState(t *testing.T) {
	tests := []struct {
		name     string
		state    ConnectionState
		expected string
	}{
		{
			"zero value",
			ConnectionState{},
			"cellular",
		},
		{
			"offline",
			ConnectionState{Offline: true},
			"offline",
		},
		{
			"wifi",
			ConnectionState{Type: ConnectionWifi},
			"wifi",
		},
		{
			"wifi tethered",
			ConnectionState{Type: ConnectionWifi, Expensive: true},
			"wifi (expensive)",
		},
		{
			"unknown",
			ConnectionState{Type: ConnectionUnknown},
			"unknown",
		},
	}

	for _, test := range tests {
		str := test.state.String()
		if str != test.expected {
			t.Fatalf("Expected String() to return '%s', got '%s'", test.expected, str)
		}
	}
}
