package images

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/draw"
)

const exifOrientationTag = 0x0112

// jpegOrientation returns the EXIF Orientation (1-8) stored in a JPEG's APP1 segment,
// or 1 when there is none or it can't be read.
func jpegOrientation(data []byte) int {
	if !IsJpeg(data) {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xFF { // fill byte
			i++
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // SOS / EOI: no metadata past this point
			return 1
		}
		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if segLen < 2 || i+2+segLen > len(data) {
			return 1
		}
		seg := data[i+4 : i+2+segLen]
		if marker == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return tiffOrientation(seg[6:])
		}
		i += 2 + segLen
	}
	return 1
}

// tiffOrientation reads the Orientation tag from IFD0 of a TIFF/EXIF block.
func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int(bo.Uint32(tiff[4:8]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	n := int(bo.Uint16(tiff[ifd : ifd+2]))
	for k := 0; k < n; k++ {
		e := ifd + 2 + k*12
		if e+12 > len(tiff) {
			return 1
		}
		if bo.Uint16(tiff[e:e+2]) != exifOrientationTag {
			continue
		}
		// type SHORT, count 1: the value sits in the first 2 bytes of the value field
		if bo.Uint16(tiff[e+2:e+4]) != 3 {
			return 1
		}
		o := int(bo.Uint16(tiff[e+8 : e+10]))
		if o < 1 || o > 8 {
			return 1
		}
		return o
	}
	return 1
}

// applyOrientation returns img transformed so that it displays upright for the given
// EXIF orientation (1-8). Orientation 1 (or an unknown value) returns img unchanged.
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}

	b := img.Bounds()
	src := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(src, src.Bounds(), img, b.Min, draw.Src)

	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if orientation >= 5 { // 5-8 swap width and height
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	for y := 0; y < h; y++ {
		row := src.Pix[y*src.Stride : y*src.Stride+w*4]
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
			case 2: // mirror horizontal
				dx, dy = w-1-x, y
			case 3: // rotate 180
				dx, dy = w-1-x, h-1-y
			case 4: // mirror vertical
				dx, dy = x, h-1-y
			case 5: // transpose
				dx, dy = y, x
			case 6: // rotate 90 clockwise
				dx, dy = h-1-y, x
			case 7: // transverse
				dx, dy = h-1-y, w-1-x
			case 8: // rotate 90 counter-clockwise
				dx, dy = y, w-1-x
			}
			o := dy*dst.Stride + dx*4
			copy(dst.Pix[o:o+4], row[x*4:x*4+4])
		}
	}
	return dst
}
