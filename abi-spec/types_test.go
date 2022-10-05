package abispec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToGoTypeValue(t *testing.T) {
	var raw json.RawMessage
	err := json.Unmarshal([]byte(`"dave"`), &raw)
	require.NoError(t, err)
	val, err := toGoTypeValue("bytes", raw)
	require.NoError(t, err)
	require.Equal(t, []byte("dave"), val.Elem().Bytes())

	err = json.Unmarshal([]byte(`true`), &raw)
	require.NoError(t, err)
	val, err = toGoTypeValue("bool", raw)
	require.NoError(t, err)
	require.True(t, val.Elem().Bool())
}

func TestToGoType(t *testing.T) {
	var raws []json.RawMessage
	err := json.Unmarshal([]byte("[8]"), &raws)
	require.NoError(t, err)
	value, err := toGoTypeValue("uint8", raws[0])
	require.NoError(t, err)
	require.Equal(t, uint8(8), *value.Interface().(*uint8))

	goType, err := toGoType("uint256[][3][]")
	require.NoError(t, err)
	require.Equal(t, "[][3][]*big.Int", goType.String())

	goType, err = toGoType("uint256[][][3]")
	require.NoError(t, err)
	require.Equal(t, "[3][][]*big.Int", goType.String())

	goType, err = toGoType("uint256[3][][]")
	require.NoError(t, err)
	require.Equal(t, "[][][3]*big.Int", goType.String())

	goType, err = toGoType("bytes3[2]")
	require.NoError(t, err)
	require.Equal(t, "[2][3]uint8", goType.String())

}

func TestArrayTypePattern(t *testing.T) {
	require.True(t, arrayTypePattern.MatchString(`uint8[]`))
	require.False(t, arrayTypePattern.MatchString(`uint8`))

	s := "uint8[][2][1][]"
	matches := arrayTypePattern.FindAllStringSubmatch(s, -1)
	require.Equal(t, 3, len(matches[0]))
	require.Equal(t, "", matches[0][2])
	require.Equal(t, "2", matches[1][2])

	index := arrayTypePattern.FindStringIndex(s)[0]
	require.Equal(t, 5, index)
	require.Equal(t, "uint8", s[0:index])
}
