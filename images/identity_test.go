package images

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIdentityImage_GetDataURI(t *testing.T) {
	cs := []struct {
		II    IdentityImage
		URI   string
		Error error
	}{
		{
			IdentityImage{Payload: testJpegBytes},
			"data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=",
			nil,
		},
		{
			IdentityImage{Payload: testPngBytes},
			"data:image/png;base64,iVBORw0KGgoAAAANSUg=",
			nil,
		},
		{
			IdentityImage{Payload: testGifBytes},
			"data:image/gif;base64,R0lGODlhAAEAAYQfAP8=",
			nil,
		},
		{
			IdentityImage{Payload: testWebpBytes},
			"data:image/webp;base64,UklGRpBJAABXRUJQVlA=",
			nil,
		},
		{
			IdentityImage{Payload: testAacBytes},
			"",
			errors.New("image format not supported"),
		},
	}

	for _, c := range cs {
		u, err := c.II.GetDataURI()

		if c.Error == nil {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, c.Error.Error())
		}

		require.Exactly(t, c.URI, u)
	}
}

func TestIdentityImage_MarshalJSON(t *testing.T) {
	ii := IdentityImage{
		Name:         "thumbnail",
		Payload:      testJpegBytes,
		Width:        80,
		Height:       80,
		FileSize:     256,
		ResizeTarget: 80,
	}
	expected := `{"keyUid":"","type":"thumbnail","uri":"data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=","width":80,"height":80,"fileSize":256,"resizeTarget":80}`

	js, err := json.Marshal(ii)
	require.NoError(t, err)
	require.Exactly(t, expected, string(js))
}
