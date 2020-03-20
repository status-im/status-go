package node

import (
	"testing"

	"github.com/stretchr/testify/require"

	prom "github.com/prometheus/client_golang/prometheus"
)

func TestParsingLabelsFromNodeName(t *testing.T) {
	var labels prom.Labels
	var err error

	// mobile name
	labels, err = labelsFromNodeName("StatusIM/v0.30.1-beta.2/android-arm/go1.12")
	require.NoError(t, err)
	require.Equal(t, labels,
		prom.Labels{
			"platform": "android-arm",
			"type":     "StatusIM",
			"version":  "v0.30.1-beta.2",
		})
	// desktop name
	labels, err = labelsFromNodeName("Statusd/v0.29.0-beta.2/linux-amd64/go1.11")
	require.NoError(t, err)
	require.Equal(t, labels,
		prom.Labels{
			"platform": "linux-amd64",
			"type":     "Statusd",
			"version":  "v0.29.0-beta.2",
		})
	// missing version
	labels, err = labelsFromNodeName("StatusIM/android-arm64/go1.11")
	require.NoError(t, err)
	require.Equal(t, labels,
		prom.Labels{
			"platform": "android-arm64",
			"type":     "StatusIM",
			"version":  "unknown",
		})
}
