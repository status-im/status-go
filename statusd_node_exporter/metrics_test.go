package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransformMetrics(t *testing.T) {
	rawdata := `{
		"discv5": {
			"InboundTraffic": {
				"AvgRate01Min": 2.0795560694463914e-25,
				"AvgRate05Min": 0.008658276307800729,
				"AvgRate15Min": 55.54020873026976,
				"MeanRate": 58.25590232351501,
				"Overall": 233315
			},
			"OutboundTraffic": {
				"AvgRate01Min": 3.5387721621595252e-25,
				"AvgRate05Min": 0.00811869219645128,
				"AvgRate15Min": 46.94673320076724,
				"MeanRate": 85.21793726030589,
				"Overall": 341298
			}
		},
		"chain": {
			"inserts": {
				"AvgRate01Min": 0,
				"AvgRate05Min": 0,
				"AvgRate15Min": 0,
				"MeanRate": 0,
				"Overall": 0,
				"Percentiles": {
					"5": 0,
					"20": 0,
					"50": 0,
					"80": 0,
					"95": 0
				}
			}
    }
	}`

	expected := flatMetrics{
		"discv5_inboundTraffic_avgRate01Min":  "2.0795560694463914e-25",
		"discv5_inboundTraffic_avgRate05Min":  "0.008658276307800729",
		"discv5_inboundTraffic_avgRate15Min":  "55.54020873026976",
		"discv5_inboundTraffic_meanRate":      "58.25590232351501",
		"discv5_inboundTraffic_overall":       "233315",
		"discv5_outboundTraffic_avgRate01Min": "3.5387721621595252e-25",
		"discv5_outboundTraffic_avgRate05Min": "0.00811869219645128",
		"discv5_outboundTraffic_avgRate15Min": "46.94673320076724",
		"discv5_outboundTraffic_meanRate":     "85.21793726030589",
		"discv5_outboundTraffic_overall":      "341298",
		"chain_inserts_avgRate01Min":          "0",
		"chain_inserts_avgRate05Min":          "0",
		"chain_inserts_avgRate15Min":          "0",
		"chain_inserts_meanRate":              "0",
		"chain_inserts_overall":               "0",
		"chain_inserts_percentiles_5":         "0",
		"chain_inserts_percentiles_20":        "0",
		"chain_inserts_percentiles_50":        "0",
		"chain_inserts_percentiles_80":        "0",
		"chain_inserts_percentiles_95":        "0",
	}

	var data map[string]interface{}
	err := json.Unmarshal([]byte(rawdata), &data)
	require.Nil(t, err)

	m := transformMetrics(data)
	require.Equal(t, expected, m)
}

func TestNormalizeKey(t *testing.T) {
	scenarios := [][2]string{
		{"", ""},
		{"foo", "foo"},
		{"Foo", "foo"},
		{"FooBar", "fooBar"},
	}

	for i, s := range scenarios {
		t.Run(fmt.Sprintf("Scenario %d", i+1), func(t *testing.T) {
			require.Equal(t, s[1], normalizeKey(s[0]))
		})
	}
}
