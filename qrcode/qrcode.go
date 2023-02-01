package qrcode

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/yeqown/go-qrcode/v2"
	xdraw "golang.org/x/image/draw"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/url"
	"os"
)

const (
	defaultPadding  = 20
	correctionLevel = qrcode.ErrorCorrectionMedium
)

func GetImageDimensions(imgBytes []byte) (int, int, error) {
	// Decode image bytes
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return 0, 0, err
	}
	// Get the image dimensions
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	return width, height, nil
}
func ToLogoImageFromBytes(imageBytes []byte, padding int) []byte {
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		panic(err)
	}
	bounds := img.Bounds()
	newBounds := image.Rect(bounds.Min.X-padding, bounds.Min.Y-padding, bounds.Max.X+padding, bounds.Max.Y+padding)
	white := image.NewRGBA(newBounds)
	draw.Draw(white, newBounds, &image.Uniform{C: color.White}, image.ZP, draw.Src)
	// Create a circular mask
	circle := image.NewRGBA(bounds)
	draw.DrawMask(circle, bounds, img, image.ZP, &Circle{
		X: bounds.Dx() / 2,
		Y: bounds.Dy() / 2,
		R: bounds.Dx() / 2,
	}, image.ZP, draw.Over)
	// Calculate the center point of the new image
	centerX := (newBounds.Min.X + newBounds.Max.X) / 2
	centerY := (newBounds.Min.Y + newBounds.Max.Y) / 2
	// Draw the circular image in the center of the new image
	draw.Draw(white, bounds.Add(image.Pt(centerX-bounds.Dx()/2, centerY-bounds.Dy()/2)), circle, image.ZP, draw.Over)
	// Encode image to png format and save in a bytes
	var resultImg bytes.Buffer
	png.Encode(&resultImg, white)
	resultBytes := resultImg.Bytes()
	return resultBytes
}
func SuperimposeImage(imageBytes []byte, qrFilepath []byte) []byte {
	// Read the two images from bytes
	img1, _, _ := image.Decode(bytes.NewReader(imageBytes))
	img2, _, _ := image.Decode(bytes.NewReader(qrFilepath))
	// Create a new image with the dimensions of the first image
	result := image.NewRGBA(img1.Bounds())
	// Draw the first image on the new image
	draw.Draw(result, img1.Bounds(), img1, image.ZP, draw.Src)
	// Get the dimensions of the second image
	img2Bounds := img2.Bounds()
	// Calculate the x and y coordinates to center the second image
	x := (img1.Bounds().Dx() - img2Bounds.Dx()) / 2
	y := (img1.Bounds().Dy() - img2Bounds.Dy()) / 2
	// Draw the second image on top of the first image at the calculated coordinates
	draw.Draw(result, img2Bounds.Add(image.Pt(x, y)), img2, image.ZP, draw.Over)
	// Encode the final image to a desired format
	var b bytes.Buffer
	if err := png.Encode(&b, result); err != nil {
		fmt.Println(err)
	}
	return b.Bytes()
}
func ResizeImage(imgBytes []byte, width, height int) ([]byte, error) {
	// Decode image bytes
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}
	// Create a new image with the desired dimensions
	newImg := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.BiLinear.Scale(newImg, newImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	// Encode the new image to bytes
	var newImgBytes bytes.Buffer
	if err = png.Encode(&newImgBytes, newImg); err != nil {
		return nil, err
	}
	return newImgBytes.Bytes(), nil
}

func ImageToBytes(imagePath string) ([]byte, error) {
	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	// Create a new buffer to hold the image data
	var imgBuffer bytes.Buffer

	// Encode the image to the desired format and save it in the buffer
	err = png.Encode(&imgBuffer, img)
	if err != nil {
		return nil, err
	}

	// Return the image data as a byte slice
	return imgBuffer.Bytes(), nil
}

func GetLogoImage(multiaccountsDB *multiaccounts.Database, params url.Values) ([]byte, error) {
	var imageFolderBasePath = "../_assets/tests/"
	var logoFileStaticPath = imageFolderBasePath + "status.png"

	keyUids, ok := params["keyUid"]
	if !ok || len(keyUids) == 0 {
		return nil, errors.New("no keyUid")
	}
	imageNames, ok := params["imageName"]
	if !ok || len(imageNames) == 0 {
		return nil, errors.New("no imageName")
	}
	identityImageObjectFromDB, err := multiaccountsDB.GetIdentityImage(keyUids[0], imageNames[0])
	if err != nil {
		return nil, err
	}

	staticLogoFileBytes, _ := ImageToBytes(logoFileStaticPath)

	if identityImageObjectFromDB == nil {
		return ToLogoImageFromBytes(staticLogoFileBytes, GetPadding(staticLogoFileBytes)), nil
	} else {
		return ToLogoImageFromBytes(identityImageObjectFromDB.Payload, GetPadding(identityImageObjectFromDB.Payload)), nil
	}

}

func GetPadding(imgBytes []byte) int {
	size, _, err := GetImageDimensions(imgBytes)
	if err != nil {
		return defaultPadding
	}
	return size / 5
}

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
		return color.Alpha{255}
	}
	return color.Alpha{0}
}
