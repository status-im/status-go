package images

const (
	UNKNOWN FileType = 1 + iota

	// Raster image types
	JPEG
	PNG
	GIF
	WEBP
)

type Details struct {
	SizePixel  uint
	SizeFile   int64
	Quality    int
	FileName   string
	Properties string
}

type FileType uint

