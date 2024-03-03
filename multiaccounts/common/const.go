package common

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

// ColorToID converts a CustomizationColor to its index.
func ColorToID(color CustomizationColor) uint32 {
	switch color {
	case CustomizationColorPrimary:
		return 0
	case CustomizationColorPurple:
		return 1
	case CustomizationColorIndigo:
		return 2
	case CustomizationColorTurquoise:
		return 3
	case CustomizationColorBlue:
		return 4
	case CustomizationColorGreen:
		return 5
	case CustomizationColorYellow:
		return 6
	case CustomizationColorOrange:
		return 7
	case CustomizationColorRed:
		return 8
	case CustomizationColorFlamingo:
		return 9
	case CustomizationColorBrown:
		return 10
	case CustomizationColorSky:
		return 11
	case CustomizationColorArmy:
		return 12
	case CustomizationColorMagenta:
		return 13
	case CustomizationColorCopper:
		return 14
	case CustomizationColorCamel:
		return 15
	case CustomizationColorYinYang:
		return 16
	case CustomizationColorBeige:
		return 17
	default:
		return 0
	}
}

// IDToColor converts an index to its corresponding CustomizationColor.
func IDToColor(index uint32) CustomizationColor {
	switch index {
	case 0:
		return CustomizationColorPrimary
	case 1:
		return CustomizationColorPurple
	case 2:
		return CustomizationColorIndigo
	case 3:
		return CustomizationColorTurquoise
	case 4:
		return CustomizationColorBlue
	case 5:
		return CustomizationColorGreen
	case 6:
		return CustomizationColorYellow
	case 7:
		return CustomizationColorOrange
	case 8:
		return CustomizationColorRed
	case 9:
		return CustomizationColorFlamingo
	case 10:
		return CustomizationColorBrown
	case 11:
		return CustomizationColorSky
	case 12:
		return CustomizationColorArmy
	case 13:
		return CustomizationColorMagenta
	case 14:
		return CustomizationColorCopper
	case 15:
		return CustomizationColorCamel
	case 16:
		return CustomizationColorYinYang
	case 17:
		return CustomizationColorBeige
	default:
		return CustomizationColorPrimary
	}
}
