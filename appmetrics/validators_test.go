package appmetrics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestStringSchema(t *testing.T) {
	StringLoader := gojsonschema.NewGoLoader(StringSchema)
	schema, _ := gojsonschema.NewSchema(StringLoader)

	// test loading valid strings
	doc := gojsonschema.NewStringLoader(`"valid-string"`)
	result, err := schema.Validate(doc)
	require.NoError(t, err)
	require.True(t, result.Valid())

	// invalid strings
	doc = gojsonschema.NewStringLoader("1")
	result, err = schema.Validate(doc)
	require.NoError(t, err)
	require.False(t, result.Valid())
}

func TestNavigationNavigateToCofxSchema(t *testing.T) {
	NavigationNavigateToCofxLoader := gojsonschema.NewGoLoader(NavigationNavigateToCofxSchema)
	schema, _ := gojsonschema.NewSchema(NavigationNavigateToCofxLoader)

	// test correct json
	validSampleVal := `{"view_id": "less-than-16", "params": {"screen": "allowed-screen-name"}}`
	doc := gojsonschema.NewStringLoader(validSampleVal)
	result, err := schema.Validate(doc)
	require.NoError(t, err)
	require.True(t, result.Valid())

	// test in-correct json
	invalidSampleVal := `{"view_id": "more-than-16-chars", "params": {"screen": "not-allowed-screen-name"}}`
	doc = gojsonschema.NewStringLoader(invalidSampleVal)
	result, err = schema.Validate(doc)
	require.NoError(t, err)
	require.False(t, result.Valid())

	// test extra params
	extraParamsVal := `{"view_id": "valid-view", "params": {"screen": "allowed-screen-name"}, "fishy-key": "fishy-val"}`
	doc = gojsonschema.NewStringLoader(extraParamsVal)
	result, err = schema.Validate(doc)
	require.NoError(t, err)
	require.False(t, result.Valid())

	// test less params
	lessParamsVal := `{"view_id": "valid-view"}`
	doc = gojsonschema.NewStringLoader(lessParamsVal)
	result, err = schema.Validate(doc)
	require.NoError(t, err)
	require.False(t, result.Valid())
}
