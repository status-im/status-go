package appmetrics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestNavigateToCofxSchema(t *testing.T) {
	NavigateToCofxLoader := gojsonschema.NewGoLoader(NavigateToCofxSchema)
	schema, _ := gojsonschema.NewSchema(NavigateToCofxLoader)

	// test correct json
	validSampleVal := `{"view_id": "less-than-32", "params": {"screen": ""}}`
	doc := gojsonschema.NewStringLoader(validSampleVal)
	result, err := schema.Validate(doc)
	require.NoError(t, err)
	require.True(t, result.Valid())

	// test in-correct json
	invalidSampleVal := `{"view_id": "more-than-32-chars-3232323232323232323232", "params": {"screen": "not-login"}}`
	doc = gojsonschema.NewStringLoader(invalidSampleVal)
	result, err = schema.Validate(doc)
	require.NoError(t, err)
	require.False(t, result.Valid())

	// test extra params
	extraParamsVal := `{"view_id": "valid-view", "params": {"screen": "login"}, "fishy-key": "fishy-val"}`
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
