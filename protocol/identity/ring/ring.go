package ring

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"

	"github.com/fogleman/gg"

	"github.com/status-im/status-go/multiaccounts"
)

type Theme int

const (
	LightTheme Theme = 1
	DarkTheme  Theme = 2
)

var (
	lightThemeIdenticonRingColors = []string{"#000000", "#00FF00", "#FFFF00", "#FF0000", "#FF00FF", "#0000FF", "#00FFFF", "#726F6F", "#009800", "#A8AC00", "#9A0000", "#900090", "#000086", "#008694", "#C4C4C4", "#B8FFBB", "#FFFFB0", "#FF9D9D", "#FFB0FF", "#9B81FF", "#C2FFFF", "#E7E7E7", "#FFC413", "#FF5733", "#FF0099", "#9E00FF", "#3FAEF9", "#00F0B6", "#FFFFFF", "#9F5947", "#C80078", "#9A6600"}
	darkThemeIdenticonRingColors  = []string{"#000000", "#00FF00", "#FFFF00", "#FF0000", "#FF00FF", "#0000FF", "#00FFFF", "#726F6F", "#009800", "#A8AC00", "#9A0000", "#900090", "#000086", "#008694", "#C4C4C4", "#B8FFBB", "#FFFFB0", "#FF9D9D", "#FFB0FF", "#9B81FF", "#C2FFFF", "#E7E7E7", "#FFC413", "#FF5733", "#FF0099", "#9E00FF", "#3FAEF9", "#00F0B6", "#FFFFFF", "#9F5947", "#C80078", "#9A6600"}
)

type DrawRingParam struct {
	Theme      Theme                   `json:"theme"`
	ColorHash  multiaccounts.ColorHash `json:"colorHash"`
	ImageBytes []byte                  `json:"imageBytes"`
	Height     int                     `json:"height"`
	Width      int                     `json:"width"`
}

func DrawRing(param *DrawRingParam) ([]byte, error) {
	var colors []string
	switch param.Theme {
	case LightTheme:
		colors = lightThemeIdenticonRingColors
	case DarkTheme:
		colors = darkThemeIdenticonRingColors
	default:
		return nil, fmt.Errorf("unknown theme")
	}

	dc := gg.NewContext(param.Width, param.Height)
	img, _, err := image.Decode(bytes.NewReader(param.ImageBytes))
	if err != nil {
		return nil, err
	}
	dc.DrawImage(img, 0, 0)

	ringPxSize := math.Max(2.0, float64(param.Width/16.0))
	radius := (float64(param.Height) - ringPxSize) / 2
	arcPos := 0.0

	totalRingUnits := 0
	for i := 0; i < len(param.ColorHash); i++ {
		totalRingUnits += param.ColorHash[i][0]
	}
	unitRadLen := 2 * math.Pi / float64(totalRingUnits)

	for i := 0; i < len(param.ColorHash); i++ {
		dc.SetHexColor(colors[param.ColorHash[i][1]])
		dc.DrawArc(float64(param.Width/2), float64(param.Height/2), radius, arcPos, arcPos+unitRadLen*float64(param.ColorHash[i][0]))
		dc.SetLineWidth(ringPxSize)
		dc.Stroke()
		arcPos += unitRadLen * float64(param.ColorHash[i][0])
	}

	buf := new(bytes.Buffer)
	err = png.Encode(buf, dc.Image())
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
