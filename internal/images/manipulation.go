package images

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"go.uber.org/zap"
	xdraw "golang.org/x/image/draw"

	"github.com/status-im/status-go/internal/logutils"
)

type Circle struct {
	X, Y, R int
}

func (c *Circle) ColorModel() color.Model {
	return color.AlphaModel
}
func (c *Circle) Bounds() image.Rectangle {
	return image.Rect(c.X-c.R, c.Y-c.R, c.X+c.R, c.Y+c.R)
}
func (c *Circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.X)+0.5, float64(y-c.Y)+0.5, float64(c.R)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{A: 255}
	}
	return color.Alpha{A: 0}
}

// Calculates scaling factors using old and new image dimensions.
func calcFactors(width, height int, oldWidth, oldHeight float64) (scaleX, scaleY float64) {
	if width == 0 {
		if height == 0 {
			scaleX = 1.0
			scaleY = 1.0
		} else {
			scaleY = oldHeight / float64(height)
			scaleX = scaleY
		}
	} else {
		scaleX = oldWidth / float64(width)
		if height == 0 {
			scaleY = scaleX
		} else {
			scaleY = oldHeight / float64(height)
		}
	}
	return
}

func resizeImage(img image.Image, width int, height int) image.Image {
	scaleX, scaleY := calcFactors(width, height, float64(img.Bounds().Dx()), float64(img.Bounds().Dy()))
	if width == 0 {
		width = int(0.7 + float64(img.Bounds().Dx())/scaleX)
	}
	if height == 0 {
		height = int(0.7 + float64(img.Bounds().Dy())/scaleY)
	}

	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.BiLinear.Scale(out, out.Bounds(), img, img.Bounds(), draw.Over, nil)
	return out
}

func Resize(size ResizeDimension, img image.Image) image.Image {
	var width, height int

	switch {
	case img.Bounds().Max.X == img.Bounds().Max.Y:
		width, height = int(size), int(size)
	case img.Bounds().Max.X > img.Bounds().Max.Y:
		width, height = 0, int(size)
	default:
		width, height = int(size), 0
	}

	logutils.ZapLogger().Info("resizing",
		zap.Uint("size", uint(size)),
		zap.Int("width", width),
		zap.Int("height", height))

	return resizeImage(img, width, height)
}

func Scale(percent int, img image.Image) image.Image {
	width := img.Bounds().Max.X * percent / 100
	height := img.Bounds().Max.Y * percent / 100

	return resizeImage(img, width, height)
}

func ShrinkOnly(size ResizeDimension, img image.Image) image.Image {
	finalSize := min(int(size), img.Bounds().Dx(), img.Bounds().Dy())
	return Resize(ResizeDimension(finalSize), img)
}

func Crop(img image.Image, rect image.Rectangle) (image.Image, error) {

	if img.Bounds().Max.X < rect.Max.X || img.Bounds().Max.Y < rect.Max.Y {
		return nil, fmt.Errorf(
			"crop dimensions out of bounds of image, image width '%dpx' & height '%dpx'; crop bottom right coordinate at X '%dpx' Y '%dpx'",
			img.Bounds().Max.X, img.Bounds().Max.Y,
			rect.Max.X, rect.Max.Y,
		)
	}

	// Use standard library cropping via SubImage on an RGBA buffer
	// Ensure the requested rectangle is within the image bounds (checked above)
	// Create a new RGBA of the crop size and draw from the source offset by rect.Min
	cropW, cropH := rect.Dx(), rect.Dy()
	out := image.NewRGBA(image.Rect(0, 0, cropW, cropH))
	// Source point corresponds to rect.Min in the original image
	draw.Draw(out, out.Bounds(), img, rect.Min, draw.Src)
	return out, nil
}

// CropCenter takes an image, usually downloaded from a URL
// If the image is square, the full image is returned
// If the image is rectangular, the largest central square is returned
func CropCenter(img image.Image) (image.Image, error) {
	var cropRect image.Rectangle
	maxBounds := img.Bounds().Max

	if maxBounds.X == maxBounds.Y {
		return img, nil
	}

	if maxBounds.X > maxBounds.Y {
		// the final output should be YxY
		cropRect = image.Rectangle{
			Min: image.Point{X: maxBounds.X/2 - maxBounds.Y/2, Y: 0},
			Max: image.Point{X: maxBounds.X/2 + maxBounds.Y/2, Y: maxBounds.Y},
		}
	} else {
		// the final output should be XxX
		cropRect = image.Rectangle{
			Min: image.Point{X: 0, Y: maxBounds.Y/2 - maxBounds.X/2},
			Max: image.Point{X: maxBounds.X, Y: maxBounds.Y/2 + maxBounds.X/2},
		}
	}
	return Crop(img, cropRect)
}

func ImageToBytesAndImage(imagePath string) ([]byte, image.Image, error) {
	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, nil, err
	}

	// Create a new buffer to hold the image data
	var imgBuffer bytes.Buffer

	// Encode the image to the desired format and save it in the buffer
	err = png.Encode(&imgBuffer, img)
	if err != nil {
		return nil, nil, err
	}

	// Return the image data as a byte slice
	return imgBuffer.Bytes(), img, nil
}

func CreateCircleWithPadding(img image.Image, padding int) *image.RGBA {
	bounds := img.Bounds()
	width := bounds.Dx()
	// only relying on width as a metric here because we know that we
	// store profile images in a perfect circle
	radius := width / 2

	paddedWidth := width + 2*padding
	paddedRadius := paddedWidth / 2

	// Create a new circular image with padding
	newBounds := image.Rect(0, 0, paddedWidth, paddedWidth)
	circle := image.NewRGBA(newBounds)

	// Create a larger circular mask for the padding
	paddingMask := &Circle{
		X: paddedRadius,
		Y: paddedRadius,
		R: paddedRadius,
	}

	// Draw the white color onto the circle with padding mask
	draw.DrawMask(circle, circle.Bounds(), image.NewUniform(color.White), image.ZP, paddingMask, image.ZP, draw.Src)

	// Create a new circle mask with the original size
	circleMask := &Circle{
		X: radius,
		Y: radius,
		R: radius,
	}

	// Draw the original image onto the white circular image at the center (with padding offset)
	draw.DrawMask(circle, bounds.Add(image.Pt(padding, padding)), img, image.ZP, circleMask, image.ZP, draw.Over)

	return circle
}

func RoundCrop(inputImage []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(inputImage))
	if err != nil {
		return nil, err
	}
	result := CreateCircleWithPadding(img, 0)

	var outputImage bytes.Buffer
	err = png.Encode(&outputImage, result)
	if err != nil {
		return nil, err
	}
	return outputImage.Bytes(), nil
}
