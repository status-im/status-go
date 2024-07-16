package statusgo

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSetSignalBlocklist(t *testing.T) {
	request := "{\"blocklist\":[\"wakuv2.peerstats\",\"history.request.started\"]}"
	require.Equal(t, "{\"error\":\"\"}", SetSignalBlocklist(request))

	request = "{\"blocklist\":[]}"
	require.Equal(t, "{\"error\":\"\"}", SetSignalBlocklist(request))
}
