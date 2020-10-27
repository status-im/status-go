package images

const (
	UNKNOWN ImageType = 1 + iota

	// Raster image types
	JPEG
	PNG
	GIF
	WEBP
)

const (
	MaxJpegQuality = 80
	MinJpegQuality = 50
)

var (
	ResizeDimensions = []ResizeDimension{80, 240}

	// DimensionSizeLimit the size limits imposed on each resize dimension
	// Figures are based on the following sample data https://github.com/status-im/status-react/issues/11047#issuecomment-694970473
	DimensionSizeLimit = map[ResizeDimension]DimensionLimits{
		80: {
			Ideal: 2560, // Base on the largest sample image at quality 60% (2,554 bytes ∴ 1024 * 2.5)
			Max:   5632, // Base on the largest sample image at quality 80% + 50% margin (3,683 bytes * 1.5 ≈ 5500 ∴ 1024 * 5.5)
		},
		240: {
			Ideal: 16384, // Base on the largest sample image at quality 60% (16,143 bytes ∴ 1024 * 16)
			Max:   38400, // Base on the largest sample image at quality 80% + 50% margin (24,290 bytes * 1.5 ≈ 37500 ∴ 1024 * 37.5)
		},
	}
)

type DimensionLimits struct {
	Ideal int
	Max   int
}

type ImageType uint
type ResizeDimension uint
