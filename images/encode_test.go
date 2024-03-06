package images

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	cs := []struct {
		FileName   string
		RenderSize int
	}{
		{
			"elephant.jpg",
			1447,
		},
		{
			"rose.webp",
			11119,
		},
		{
			"spin.gif",
			2263,
		},
		{
			"status.png",
			5834,
		},
	}
	options := EncodeConfig{
		Quality: 70,
	}

	for _, c := range cs {
		img, err := Decode(path + c.FileName)
		require.NoError(t, err)

		bb := bytes.NewBuffer([]byte{})
		err = Encode(bb, img, options)
		require.NoError(t, err)

		require.Exactly(t, c.RenderSize, bb.Len())
	}
}

func TestEncodeToBestSize(t *testing.T) {
	cs := []struct {
		FileName   string
		RenderSize int
		Error      error
	}{
		{
			"elephant.jpg",
			1467,
			nil,
		},
		{
			"rose.webp",
			8513,
			errors.New("image size after processing exceeds max, expected < '5632', received < '8513'"),
		},
		{
			"spin.gif",
			2407,
			nil,
		},
		{
			"status.png",
			4725,
			nil,
		},
	}

	for _, c := range cs {
		img, err := Decode(path + c.FileName)
		require.NoError(t, err)

		bb := bytes.NewBuffer([]byte{})
		err = EncodeToBestSize(bb, img, ResizeDimensions[0])

		require.Exactly(t, c.RenderSize, bb.Len())

		if c.Error != nil {
			require.EqualError(t, err, c.Error.Error())
		} else {
			require.NoError(t, err)
		}
	}
}

func TestCompressToFileLimits(t *testing.T) {
	img, err := Decode(path + "IMG_1205.HEIC.jpg")
	require.NoError(t, err)

	bb := bytes.NewBuffer([]byte{})
	err = CompressToFileLimits(bb, img, FileSizeLimits{50000, 350000})
	require.NoError(t, err)
	require.Equal(t, 291645, bb.Len())
}

func TestGetPayloadFromURI(t *testing.T) {
	payload, err := GetPayloadFromURI("data:image/jpeg;base64,/9j/2wCEAFA3PEY8MlA=")
	require.NoError(t, err)
	require.Equal(
		t,
		[]byte{0xff, 0xd8, 0xff, 0xdb, 0x0, 0x84, 0x0, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x50},
		payload,
	)
}

func TestIsSvg(t *testing.T) {
	GoodSVG := []byte(`<svg width="300" height="130" xmlns="http://www.w3.org/2000/svg">
	  <rect width="200" height="100" x="10" y="10" rx="20" ry="20" fill="blue" />
	  Sorry, your browser does not support inline SVG.  
	</svg>`)

	BadSVG := []byte(`<head>
	<link rel="stylesheet" href="styles.css">
  </head>`)

	require.Equal(t, IsSVG(BadSVG), false)
	require.Equal(t, IsSVG(GoodSVG), true)
}

func TestIsIco(t *testing.T) {
	GoodICO := `AAABAAEAEBAAAAEAIABoBAAAFgAAACgAAAAQAAAAIAAAAAEAIAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAQEtAAAAugAAAL0DAwMwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABDPz8/0UJCQv9AQED/QUFB1AAAAEcAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAATExNKPz8/4ltbW/9paWn/a2tr/19fX/9EREXkGhoaTQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAgI0VFRU22dnZ/9WVlb/cHBw/3BwcP9VVVX/Y2Nj/1JSUt0CAgI2AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQLi4rtlNTUf9wcHD/bW1t/1tbW/9cXFz/bm5u/29vb/9VVVL/MjIwuAAAAREAAAAAAAAAAAAAAAAAAAAAAwUqZTg9hftQUnH/Fxcb/xAQEP9DQ0P/RERE/w4ODv8TExv/Sk6C/zY8kvsDBSpmAAAAAAAAAAAAAAAAAAAAEQkPZ8QQHNX/CA+E/wAABf8CAgL/FRUV/xQUFP8CAgH/AAAH/wkRkf8QHNj/CQ9mxAAAABAAAAAAAAAAAAIDE0MNFpz0FyHG/xYZUP8ODgz/WVlZ/7q6uv+5ubn/V1dX/w4ODP8WGVH/FiHI/w0Wm/QCAxJCAAAAAAAAAAAFCDt9DRi8/1FWov+xsbL/sLCx/62trP/W1tX/1tbW/62trP+wsLH/sbGy/01SpP8NGLz/BQg6ewAAAAAAAAADCAxYqBEcz/8oL6f/O0Gf/zpAoP9jZ5v/y8vM/8zMzP9iZZr/OkCg/zxCoP8nL6n/ERzP/wgMV6cAAAADAAAADAkPasESHtj/GCTb/xwo2/8UINv/FiHI/3R2nP9zdZ3/FiHI/xQg2/8cKNv/GCTb/xIe1/8JD2nAAAAADAAAABIKEHLKER3Z/0lT4/+kqfH/Lzrf/xEd3P8VHq3/FR6u/xEd3P8vOt//pKnx/0lS4/8RHdn/ChByygAAABIAAAAPChBvxxIe2f8+SOL/gYfs/ys23/8TH9v/UVrl/09Y5f8TH9v/Kzbf/4GH7P8+R+L/Eh7Z/woQcMgAAAAQAAAABwgMWLQSHc3/Eh/e/xYi3f8RHdv/Gibc/5GX7v+Nk+7/GSXc/xEe2/8WIt3/Eh/e/xIdzP8IDVm3AAAACAAAAAADBB5JCQ5iwQ4Wn/URG8T/Eh7V/xMf2/8lMd//JTDf/xMf2v8SHtP/EBu//w0Vl/AIDVy5AgQcSAAAAAAAAAAAAAAAAAAAABMCAxlTBglDnwkOZNYLEXrzCxKE/goSgv0KEXbuCA1dzQUIO5ICAhJHAAAADQAAAAAAAAAA/D8AAPgfAADwDwAA4AcAAMADAADAAwAAgAEAAIABAACAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAQAAwAMAAA==`
	GoodPNG := `iVBORw0KGgoAAAANSUhEUgAAABEAAAAOCAMAAAD+MweGAAADAFBMVEUAAAAAAFUAAKoAAP8AJAAAJFUAJKoAJP8ASQAASVUASaoASf8AbQAAbVUAbaoAbf8AkgAAklUAkqoAkv8AtgAAtlUAtqoAtv8A2wAA21UA26oA2/8A/wAA/1UA/6oA//8kAAAkAFUkAKokAP8kJAAkJFUkJKokJP8kSQAkSVUkSaokSf8kbQAkbVUkbaokbf8kkgAkklUkkqokkv8ktgAktlUktqoktv8k2wAk21Uk26ok2/8k/wAk/1Uk/6ok//9JAABJAFVJAKpJAP9JJABJJFVJJKpJJP9JSQBJSVVJSapJSf9JbQBJbVVJbapJbf9JkgBJklVJkqpJkv9JtgBJtlVJtqpJtv9J2wBJ21VJ26pJ2/9J/wBJ/1VJ/6pJ//9tAABtAFVtAKptAP9tJABtJFVtJKptJP9tSQBtSVVtSaptSf9tbQBtbVVtbaptbf9tkgBtklVtkqptkv9ttgBttlVttqpttv9t2wBt21Vt26pt2/9t/wBt/1Vt/6pt//+SAACSAFWSAKqSAP+SJACSJFWSJKqSJP+SSQCSSVWSSaqSSf+SbQCSbVWSbaqSbf+SkgCSklWSkqqSkv+StgCStlWStqqStv+S2wCS21WS26qS2/+S/wCS/1WS/6qS//+2AAC2AFW2AKq2AP+2JAC2JFW2JKq2JP+2SQC2SVW2Saq2Sf+2bQC2bVW2baq2bf+2kgC2klW2kqq2kv+2tgC2tlW2tqq2tv+22wC221W226q22/+2/wC2/1W2/6q2///bAADbAFXbAKrbAP/bJADbJFXbJKrbJP/bSQDbSVXbSarbSf/bbQDbbVXbbarbbf/bkgDbklXbkqrbkv/btgDbtlXbtqrbtv/b2wDb21Xb26rb2//b/wDb/1Xb/6rb////AAD/AFX/AKr/AP//JAD/JFX/JKr/JP//SQD/SVX/Sar/Sf//bQD/bVX/bar/bf//kgD/klX/kqr/kv//tgD/tlX/tqr/tv//2wD/21X/26r/2////wD//1X//6r////qm24uAAAA1ElEQVR42h1PMW4CQQwc73mlFJGCQChFIp0Rh0RBGV5AFUXKC/KPfCFdqryEgoJ8IX0KEF64q0PPnow3jT2WxzNj+gAgAGfvvDdCQIHoSnGYcGDE2nH92DoRqTYJ2bTcsKgqhIi47VdgAWNmwFSFA1UAAT2sSFcnq8a3x/zkkJrhaHT3N+hD3aH7ZuabGHX7bsSMhxwTJLr3evf1e0nBVcwmqcTZuatKoJaB7dSHjTZdM0G1HBTWefly//q2EB7/BEvk5vmzeQaJ7/xKPImpzv8/s4grhAxHl0DsqGUAAAAASUVORK5CYII=`

	GoodIcoBytes, err := base64.StdEncoding.DecodeString(GoodICO)

	require.NoError(t, err)

	GoodPNGBytes, err := base64.StdEncoding.DecodeString(GoodPNG)
	require.NoError(t, err)

	require.Equal(t, IsIco(GoodIcoBytes), true)
	require.Equal(t, IsPng(GoodPNGBytes), true)
	require.Equal(t, IsIco(GoodPNGBytes), false)
}
