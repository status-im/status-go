package appmetrics

var StringSchema = map[string]interface{}{
	"type": "string",
}

var NavigationNavigateToCofxSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"view_id": map[string]interface{}{
			"type": "string",
			"maxLength": 16,
		},
		"params": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"screen": map[string]interface{}{
					"enum": []string{"allowed-screen-name"},
				},
			},
		},
	},
}
