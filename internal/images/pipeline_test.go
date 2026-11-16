package images

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/image/webp"
)

// linearEncodeToLimits is the previous EncodeToLimits (walk the whole quality ladder), kept as
// the reference the binary search must match.
func linearEncodeToLimits(bb *bytes.Buffer, img image.Image, bounds FileSizeLimits) error {
	for q := MaxJpegQuality; q > MinJpegQuality-1; q -= 2 {
		if err := Encode(bb, img, EncodeConfig{Quality: q}); err != nil {
			return err
		}
		if bounds.Ideal > bb.Len() {
			return nil
		}
		if q == MinJpegQuality {
			if bounds.Max > bb.Len() {
				return nil
			}
			return &FileSizeError{expected: bounds.Max, received: bb.Len()}
		}
		bb.Reset()
	}
	return nil
}

func noiseImage(w, h int, seed int64, amplitude int) *image.RGBA {
	r := rand.New(rand.NewSource(seed)) //nolint:gosec // test data
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			base := uint8((x + y) * 255 / (w + h))
			n := func() uint8 { return uint8(int(base) + r.Intn(amplitude+1) - amplitude/2) }
			img.SetRGBA(x, y, color.RGBA{n(), n(), n(), 255})
		}
	}
	return img
}

func TestEncodeToLimitsMatchesLinearWalk(t *testing.T) {
	limits := []FileSizeLimits{DefaultBounds, DimensionSizeLimit[SmallDim], DimensionSizeLimit[LargeDim], {Ideal: 30000, Max: 60000}}
	imgs := map[string]image.Image{
		"smooth":    noiseImage(400, 300, 1, 0),
		"mild":      noiseImage(400, 300, 2, 40),
		"noisy":     noiseImage(600, 400, 3, 255),
		"verynoisy": noiseImage(900, 700, 4, 255),
	}
	for name, img := range imgs {
		for _, l := range limits {
			want := bytes.NewBuffer(nil)
			wantErr := linearEncodeToLimits(want, img, l)
			got := bytes.NewBuffer(nil)
			gotErr := EncodeToLimits(got, img, l)

			require.Equal(t, wantErr, gotErr, "%s %+v", name, l)
			require.True(t, bytes.Equal(want.Bytes(), got.Bytes()), "%s %+v: output differs (%d vs %d bytes)", name, l, want.Len(), got.Len())
		}
	}
}

func TestResizeImageReturnsRGBA(t *testing.T) {
	out := resizeImage(noiseImage(40, 20, 5, 10), 20, 0)
	_, ok := out.(*image.RGBA)
	require.True(t, ok, "jpeg.Encode only has a fast path for *image.RGBA / *image.YCbCr")
	require.Equal(t, image.Rect(0, 0, 20, 10), out.Bounds())
}

// exifSegment builds an APP1 EXIF segment with the given orientation and a GPS-looking marker.
func exifSegment(orientation uint16, bigEndian bool) []byte {
	var bo interface {
		binary.ByteOrder
		binary.AppendByteOrder
	} = binary.LittleEndian
	tiff := []byte("II*\x00")
	if bigEndian {
		bo = binary.BigEndian
		tiff = []byte("MM\x00*")
	}
	tiff = bo.AppendUint32(tiff, 8)                  // IFD0 offset
	tiff = bo.AppendUint16(tiff, 1)                  // 1 entry
	tiff = bo.AppendUint16(tiff, exifOrientationTag) // tag
	tiff = bo.AppendUint16(tiff, 3)                  // SHORT
	tiff = bo.AppendUint32(tiff, 1)                  // count
	tiff = bo.AppendUint16(tiff, orientation)
	tiff = append(tiff, 0, 0)       // value padding
	tiff = bo.AppendUint32(tiff, 0) // next IFD
	tiff = append(tiff, []byte("GPS 46.7712N 23.6236E secret")...)

	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1}
	seg = binary.BigEndian.AppendUint16(seg, uint16(len(payload)+2)) //nolint:gosec // small test segment
	return append(seg, payload...)
}

// withSegment inserts a JPEG segment right after SOI.
func withSegment(jpg, seg []byte) []byte {
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

// halvesImage is red on the left half and blue on the right half.
func halvesImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{255, 0, 0, 255}
			if x >= w/2 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func encodeJpeg(t *testing.T, img image.Image, q int) []byte {
	var b bytes.Buffer
	require.NoError(t, jpeg.Encode(&b, img, &jpeg.Options{Quality: q}))
	return b.Bytes()
}

func isRed(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r > 0xC000 && g < 0x4000 && b < 0x4000
}

func isBlue(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return b > 0xC000 && r < 0x4000 && g < 0x4000
}

func TestJpegOrientation(t *testing.T) {
	jpg := encodeJpeg(t, halvesImage(8, 4), 90)
	require.Equal(t, 1, jpegOrientation(jpg))
	for o := uint16(1); o <= 8; o++ {
		require.Equal(t, int(o), jpegOrientation(withSegment(jpg, exifSegment(o, false))))
		require.Equal(t, int(o), jpegOrientation(withSegment(jpg, exifSegment(o, true))))
	}
	require.Equal(t, 1, jpegOrientation(withSegment(jpg, exifSegment(9, false))), "invalid value")
	require.Equal(t, 1, jpegOrientation([]byte{0xFF, 0xD8, 0xFF}), "truncated")
	require.Equal(t, 1, jpegOrientation([]byte("not a jpeg")))
}

func TestDecodeAppliesJpegOrientation(t *testing.T) {
	// 40x20, red left half / blue right half
	jpg := encodeJpeg(t, halvesImage(40, 20), 95)
	cs := []struct {
		orientation uint16
		size        image.Point
		red, blue   image.Point // a point that must be red / blue once upright
	}{
		{1, image.Pt(40, 20), image.Pt(5, 10), image.Pt(35, 10)},
		{2, image.Pt(40, 20), image.Pt(35, 10), image.Pt(5, 10)}, // mirrored
		{3, image.Pt(40, 20), image.Pt(35, 10), image.Pt(5, 10)}, // 180
		{4, image.Pt(40, 20), image.Pt(5, 10), image.Pt(35, 10)}, // flipped vertically
		{5, image.Pt(20, 40), image.Pt(10, 5), image.Pt(10, 35)}, // transpose: left goes to top
		{6, image.Pt(20, 40), image.Pt(10, 5), image.Pt(10, 35)}, // 90 CW: left goes to top
		{7, image.Pt(20, 40), image.Pt(10, 35), image.Pt(10, 5)}, // transverse: left goes to bottom
		{8, image.Pt(20, 40), image.Pt(10, 35), image.Pt(10, 5)}, // 90 CCW: left goes to bottom
	}
	for _, c := range cs {
		data := withSegment(jpg, exifSegment(c.orientation, false))
		img, err := DecodeImageData(data, bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, c.size, img.Bounds().Size(), "orientation %d", c.orientation)
		require.True(t, isRed(img.At(c.red.X, c.red.Y)), "orientation %d: %v should be red", c.orientation, c.red)
		require.True(t, isBlue(img.At(c.blue.X, c.blue.Y)), "orientation %d: %v should be blue", c.orientation, c.blue)
	}
}

// transparentImage is fully transparent except for an opaque square in the middle.
func transparentImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := h / 4; y < 3*h/4; y++ {
		for x := w / 4; x < 3*w/4; x++ {
			img.SetNRGBA(x, y, color.NRGBA{0, 128, 255, 255})
		}
	}
	return img
}

func TestCompressToFileLimitsKeepsTransparency(t *testing.T) {
	bb := bytes.NewBuffer(nil)
	require.NoError(t, CompressToFileLimits(bb, transparentImage(300, 200), DefaultBounds))
	require.True(t, IsPng(bb.Bytes()))

	out, err := png.Decode(bytes.NewReader(bb.Bytes()))
	require.NoError(t, err)
	_, _, _, a := out.At(2, 2).RGBA()
	require.Zero(t, a, "corner must stay transparent")
	_, _, _, a = out.At(150, 100).RGBA()
	require.Equal(t, uint32(0xFFFF), a, "centre must stay opaque")
}

func TestCompressToFileLimitsFlattensLargeTransparentImageOnWhite(t *testing.T) {
	// Opaque noise (doesn't fit as PNG) with a transparent corner.
	src := noiseImage(1200, 1200, 6, 255)
	img := image.NewNRGBA(src.Bounds())
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1200; x++ {
			c := src.RGBAAt(x, y)
			if x < 100 && y < 100 {
				img.SetNRGBA(x, y, color.NRGBA{})
				continue
			}
			img.SetNRGBA(x, y, color.NRGBA{c.R, c.G, c.B, 255})
		}
	}

	bb := bytes.NewBuffer(nil)
	require.NoError(t, CompressToFileLimits(bb, img, DefaultBounds))
	require.True(t, IsJpeg(bb.Bytes()))

	out, err := jpeg.Decode(bytes.NewReader(bb.Bytes()))
	require.NoError(t, err)
	r, g, b, _ := out.At(20, 20).RGBA()
	require.True(t, r > 0xF000 && g > 0xF000 && b > 0xF000, "transparent area must be white, got %v", out.At(20, 20))
}

func writeTemp(t *testing.T, name string, data []byte) string {
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, data, 0600))
	return p
}

func TestOpenAndAdjustImagePassesSmallJpegThroughWithoutMetadata(t *testing.T) {
	jpg := encodeJpeg(t, noiseImage(300, 200, 7, 60), 85)
	src := withSegment(jpg, exifSegment(1, false))
	src = withSegment(src, append([]byte{0xFF, 0xFE, 0x00, 0x0A}, []byte("comment!")...))

	out, err := OpenAndAdjustImage(CroppedImage{ImagePath: writeTemp(t, "a.jpg", src)}, false)
	require.NoError(t, err)
	require.Equal(t, jpg, out, "only the metadata segments may be removed")
	require.False(t, bytes.Contains(out, []byte("Exif")))
	require.False(t, bytes.Contains(out, []byte("secret")))
}

func TestOpenAndAdjustImageRotatesInsteadOfPassingThroughOrientedJpeg(t *testing.T) {
	src := withSegment(encodeJpeg(t, halvesImage(40, 20), 95), exifSegment(6, false))

	out, err := OpenAndAdjustImage(CroppedImage{ImagePath: writeTemp(t, "a.jpg", src)}, false)
	require.NoError(t, err)
	require.False(t, bytes.Contains(out, []byte("secret")))
	img, err := jpeg.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	require.Equal(t, image.Pt(20, 40), img.Bounds().Size(), "pixels must be rotated upright")
}

func TestOpenAndAdjustImageKeepsAnimatedGif(t *testing.T) {
	src, err := os.ReadFile(path + "spin.gif")
	require.NoError(t, err)
	// add a comment extension, which must be stripped
	withComment := append([]byte{}, src[:len(src)-1]...)
	withComment = append(withComment, 0x21, 0xFE, 6, 's', 'e', 'c', 'r', 'e', 't', 0, 0x3B)

	out, err := OpenAndAdjustImage(CroppedImage{ImagePath: writeTemp(t, "a.gif", withComment)}, false)
	require.NoError(t, err)
	require.Equal(t, src, out)

	g, err := gif.DecodeAll(bytes.NewReader(out))
	require.NoError(t, err)
	require.Len(t, g.Image, 12)
}

func TestOpenAndAdjustImageKeepsAnimatedWebp(t *testing.T) {
	src, err := os.ReadFile(path + "animated.webp")
	require.NoError(t, err)

	out, err := OpenAndAdjustImage(CroppedImage{ImagePath: writeTemp(t, "a.webp", src)}, false)
	require.NoError(t, err)
	require.Equal(t, src, out)
	require.True(t, bytes.Contains(out, []byte("ANIM")))
}

func TestOpenAndAdjustImageKeepsSmallTransparentPng(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, transparentImage(64, 64)))
	src := b.Bytes()
	// insert a tEXt chunk after IHDR (8 byte signature + 25 byte IHDR chunk)
	text := []byte("Comment\x00secret")
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(text))) //nolint:gosec // small test chunk
	chunk = append(chunk, "tEXt"...)
	chunk = append(chunk, text...)
	chunk = append(chunk, 0, 0, 0, 0) // CRC isn't checked by the stripper
	withText := append(append(append([]byte{}, src[:33]...), chunk...), src[33:]...)

	out, err := OpenAndAdjustImage(CroppedImage{ImagePath: writeTemp(t, "a.png", withText)}, false)
	require.NoError(t, err)
	require.Equal(t, src, out)
}

func TestOpenAndAdjustImageReencodesImagesOverTheLimits(t *testing.T) {
	// larger than longSideMax: goes through the pipeline and is resized
	src := encodeJpeg(t, noiseImage(2400, 100, 8, 10), 80)
	out, err := OpenAndAdjustImage(CroppedImage{ImagePath: writeTemp(t, "a.jpg", src)}, false)
	require.NoError(t, err)
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	require.Equal(t, longSideMax, cfg.Width)
}

func TestStripWebpMetadata(t *testing.T) {
	src, err := os.ReadFile(path + "rose.webp") // VP8X with alpha
	require.NoError(t, err)

	// append an EXIF chunk (odd size, so padded) and set the VP8X EXIF flag
	withExif := append([]byte{}, src...)
	withExif[20] |= 0x08
	withExif = append(withExif, "EXIF"...)
	withExif = binary.LittleEndian.AppendUint32(withExif, 7)
	withExif = append(withExif, "secret!"...)
	withExif = append(withExif, 0)
	binary.LittleEndian.PutUint32(withExif[4:8], uint32(len(withExif)-8)) //nolint:gosec // test data

	out, err := stripWebpMetadata(withExif)
	require.NoError(t, err)
	require.Equal(t, src, out)
	_, err = webp.Decode(bytes.NewReader(out))
	require.NoError(t, err)
}

func TestStripMetadataRejectsMalformedData(t *testing.T) {
	jpg := encodeJpeg(t, halvesImage(8, 8), 80)
	gifData, err := os.ReadFile(path + "spin.gif")
	require.NoError(t, err)
	var pngBuf bytes.Buffer
	require.NoError(t, png.Encode(&pngBuf, halvesImage(8, 8)))
	webpData, err := os.ReadFile(path + "rose.webp")
	require.NoError(t, err)

	for _, c := range []struct {
		name string
		t    ImageType
		data []byte
	}{
		{"jpeg", JPEG, jpg[:len(jpg)/2]},
		{"png", PNG, pngBuf.Bytes()[:20]},
		{"gif", GIF, gifData[:len(gifData)/2]},
		{"webp", WEBP, webpData[:30]},
		{"unknown", UNKNOWN, []byte("x")},
	} {
		_, err := stripMetadata(c.data, c.t)
		require.Error(t, err, c.name)
	}
	require.True(t, errors.Is(func() error { _, e := stripJpegMetadata([]byte("x")); return e }(), errMalformedImage))
}
