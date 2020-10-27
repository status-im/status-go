package images

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/status-im/status-go/appdatabase"

	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (Database, func()) {
	tmpfile, err := ioutil.TempFile("", "images-tests-")
	require.NoError(t, err)
	db, err := appdatabase.InitializeDB(tmpfile.Name(), "images-tests")
	require.NoError(t, err)
	return NewDatabase(db), func() {
		require.NoError(t, db.Close())
		require.NoError(t, os.Remove(tmpfile.Name()))
	}
}

func TestDatabase_GetIdentityImages(t *testing.T) {
	db, stop := setupTestDB(t)
	defer stop()

	iis := []IdentityImage{
		{
			Type:         "thumbnail",
			Payload:      testJpegBytes,
			Width:        80,
			Height:       80,
			FileSize:     256,
			ResizeTarget: 80,
		},
		{
			Type:         "large",
			Payload:      testPngBytes,
			Width:        240,
			Height:       300,
			FileSize:     1024,
			ResizeTarget: 240,
		},
	}
	expected := `[{"type":"large","uri":"data:image/png;base64,iVBORw0KGgoAAAANSUg=","width":240,"height":300,"file_size":1024,"resize_target":240},{"type":"thumbnail","uri":"data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=","width":80,"height":80,"file_size":256,"resize_target":80}]`

	err := db.StoreIdentityImages(iis)
	require.NoError(t, err)

	oiis, err := db.GetIdentityImages()
	require.NoError(t, err)

	joiis, err := json.Marshal(oiis)
	require.NoError(t, err)
	require.Exactly(t, expected, string(joiis))
}

func TestIdentityImage_GetDataURI(t *testing.T) {
	cs := []struct{
		II IdentityImage
		URI string
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
		Type:         "thumbnail",
		Payload:      testJpegBytes,
		Width:        80,
		Height:       80,
		FileSize:     256,
		ResizeTarget: 80,
	}
	expected := `{"type":"thumbnail","uri":"data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=","width":80,"height":80,"file_size":256,"resize_target":80}`

	js, err := json.Marshal(ii)
	require.NoError(t, err)
	require.Exactly(t, expected, string(js))
}
