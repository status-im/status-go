package common

import (
	"fmt"
)

type CustomizationColor string

const (
	CustomizationColorPrimary   CustomizationColor = "primary"
	CustomizationColorPurple    CustomizationColor = "purple"
	CustomizationColorIndigo    CustomizationColor = "indigo"
	CustomizationColorTurquoise CustomizationColor = "turquoise"
	CustomizationColorBlue      CustomizationColor = "blue"
	CustomizationColorGreen     CustomizationColor = "green"
	CustomizationColorYellow    CustomizationColor = "yellow"
	CustomizationColorOrange    CustomizationColor = "orange"
	CustomizationColorRed       CustomizationColor = "red"
	CustomizationColorFlamingo  CustomizationColor = "flamingo"
	CustomizationColorBrown     CustomizationColor = "brown"
	CustomizationColorSky       CustomizationColor = "sky"
	CustomizationColorArmy      CustomizationColor = "army"
	CustomizationColorMagenta   CustomizationColor = "magenta"
	CustomizationColorCopper    CustomizationColor = "copper"
	CustomizationColorCamel     CustomizationColor = "camel"
	CustomizationColorYinYang   CustomizationColor = "yinyang"
	CustomizationColorBeige     CustomizationColor = "beige"
)

var colorToIDMap = map[CustomizationColor]uint32{
	CustomizationColorPrimary:   0,
	CustomizationColorPurple:    1,
	CustomizationColorIndigo:    2,
	CustomizationColorTurquoise: 3,
	CustomizationColorBlue:      4,
	CustomizationColorGreen:     5,
	CustomizationColorYellow:    6,
	CustomizationColorOrange:    7,
	CustomizationColorRed:       8,
	CustomizationColorFlamingo:  9,
	CustomizationColorBrown:     10,
	CustomizationColorSky:       11,
	CustomizationColorArmy:      12,
	CustomizationColorMagenta:   13,
	CustomizationColorCopper:    14,
	CustomizationColorCamel:     15,
	CustomizationColorYinYang:   16,
	CustomizationColorBeige:     17,
}

func ColorToID(color CustomizationColor) (uint32, error) {
	id, ok := colorToIDMap[color]
	if !ok {
		return 0, fmt.Errorf("Invalid color: %s", color)
	}
	return id, nil
}

func IDToColor(id uint32) (CustomizationColor, error) {
	for color, colorID := range colorToIDMap {
		if colorID == id {
			return color, nil
		}
	}
	return "", fmt.Errorf("Invalid color id: %d", id)
}

func ColorToIDFallbackToBlue(color CustomizationColor) uint32 {
	id, err := ColorToID(color)
	if err != nil {
		return 4
	}
	return id
}

func IDToColorFallbackToBlue(id uint32) CustomizationColor {
	color, err := IDToColor(id)
	if err != nil {
		return CustomizationColorBlue
	}

	return color
}
