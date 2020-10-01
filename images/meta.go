package images

import (
	"fmt"
	"strings"
)

const (
	UNKNOWN FileType = 1 + iota

	// Raster image types
	JPEG
	PNG
	GIF
	WEBP

	// Vector image types
	SVG
	AI
)

type Details struct {
	SizePixel  uint
	SizeFile   int64
	Quality    int
	FileName   string
	Properties string
}

type FileType uint

func MakeDetails(imageName string, size uint, quality int, properties string) Details {
	return Details{
		SizePixel:  size,
		Quality:    quality,
		Properties: properties,
		FileName:   makeOutputName(imageName, size, quality, properties),
	}
}

func makeOutputName(imageName string, size uint, i int, properties string) string {
	if properties != "" {
		properties = "_" + strings.ReplaceAll(properties, " ", "-")
	}
	return fmt.Sprintf(ImageDir+"%s_s-%d_q-%d%s.jpg", imageName, size, i, properties)
}
