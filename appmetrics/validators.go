package appmetrics

var NavigationNavigateToCofxSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"view_id": map[string]interface{}{
			"type":      "string",
			"maxLength": 16,
		},
		"params": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"screen": map[string]interface{}{
					"enum": []string{"allowed-screen-name"},
				},
			},
			"additionalProperties": false,
			"required":             []string{"screen"},
		},
	},
	"additionalProperties": false,
	"required":             []string{"view_id", "params"},
}
