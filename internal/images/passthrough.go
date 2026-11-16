package images

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
)

var errMalformedImage = errors.New("malformed image data")

// passThrough returns data, with its metadata stripped, when it can be sent as it is: a supported
// format (JPEG, PNG, GIF, WebP) within the chat image limits (resizeTargetImageSize bytes,
// longSideMax pixels) that displays upright without its EXIF orientation. This keeps transparency
// and animation (which re-encoding to JPEG would lose) and skips needless re-encoding.
// ok is false when the image has to go through the regular pipeline instead.
func passThrough(data []byte) (payload []byte, ok bool) {
	t := GetType(data)
	switch t {
	case JPEG, PNG, GIF, WEBP:
	default:
		return nil, false
	}
	if len(data) > resizeTargetImageSize {
		return nil, false
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > longSideMax || cfg.Height > longSideMax {
		return nil, false
	}
	// Stripping EXIF drops the orientation tag: such photos must be rotated by the pipeline.
	if t == JPEG && jpegOrientation(data) != 1 {
		return nil, false
	}
	stripped, err := stripMetadata(data, t)
	if err != nil {
		return nil, false
	}
	return stripped, true
}

// stripMetadata removes metadata that may identify the sender or their device (EXIF incl. GPS,
// XMP, IPTC, text chunks, comments) without decoding or re-encoding the image data.
func stripMetadata(data []byte, t ImageType) ([]byte, error) {
	switch t {
	case JPEG:
		return stripJpegMetadata(data)
	case PNG:
		return stripPngMetadata(data)
	case GIF:
		return stripGifMetadata(data)
	case WEBP:
		return stripWebpMetadata(data)
	default:
		return nil, errors.New("unsupported image type")
	}
}

// stripJpegMetadata drops APPn segments other than JFIF (APP0), ICC profiles (APP2) and
// Adobe colour info (APP14), and COM segments. Everything from the first SOS on is kept as is.
func stripJpegMetadata(data []byte) ([]byte, error) {
	if !IsJpeg(data) {
		return nil, errMalformedImage
	}
	out := make([]byte, 0, len(data))
	out = append(out, 0xFF, 0xD8)
	i := 2
	for {
		if i+2 > len(data) || data[i] != 0xFF {
			return nil, errMalformedImage
		}
		marker := data[i+1]
		switch {
		case marker == 0xFF: // fill byte
			i++
			continue
		case marker == 0xD9: // EOI
			return append(out, data[i:]...), nil
		case marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7): // no length
			out = append(out, data[i:i+2]...)
			i += 2
			continue
		}
		if i+4 > len(data) {
			return nil, errMalformedImage
		}
		segEnd := i + 2 + int(binary.BigEndian.Uint16(data[i+2:i+4]))
		if segEnd < i+4 || segEnd > len(data) {
			return nil, errMalformedImage
		}
		if marker == 0xDA { // SOS: the entropy-coded data and the rest of the file follow
			return append(out, data[i:]...), nil
		}
		seg := data[i+4 : segEnd]
		keep := true
		switch {
		case marker == 0xFE: // COM
			keep = false
		case marker == 0xE0: // JFIF / JFXX
		case marker == 0xE2:
			keep = bytes.HasPrefix(seg, []byte("ICC_PROFILE\x00"))
		case marker == 0xEE:
			keep = bytes.HasPrefix(seg, []byte("Adobe"))
		case marker >= 0xE1 && marker <= 0xEF: // EXIF, XMP, IPTC, vendor data
			keep = false
		}
		if keep {
			out = append(out, data[i:segEnd]...)
		}
		i = segEnd
	}
}

var pngMetadataChunks = map[string]bool{"tEXt": true, "zTXt": true, "iTXt": true, "eXIf": true, "tIME": true}

// stripPngMetadata drops text, EXIF and timestamp chunks; image and animation (APNG) chunks are kept.
func stripPngMetadata(data []byte) ([]byte, error) {
	if !IsPng(data) || len(data) < 8 {
		return nil, errMalformedImage
	}
	out := make([]byte, 0, len(data))
	out = append(out, data[:8]...)
	i := 8
	for i < len(data) {
		if i+12 > len(data) {
			return nil, errMalformedImage
		}
		n := int(binary.BigEndian.Uint32(data[i : i+4]))
		end := i + 12 + n
		if n < 0 || end > len(data) || end < i {
			return nil, errMalformedImage
		}
		typ := string(data[i+4 : i+8])
		if !pngMetadataChunks[typ] {
			out = append(out, data[i:end]...)
		}
		i = end
		if typ == "IEND" {
			break
		}
	}
	return out, nil
}

// stripGifMetadata drops comment extensions and application extensions other than the
// animation loop ones (NETSCAPE2.0, ANIMEXTS1.0), e.g. XMP.
func stripGifMetadata(data []byte) ([]byte, error) {
	if !IsGif(data) || len(data) < 13 {
		return nil, errMalformedImage
	}
	i := 13 // header + logical screen descriptor
	if flags := data[10]; flags&0x80 != 0 {
		i += 3 << (int(flags&0x07) + 1) // global colour table
	}
	if i > len(data) {
		return nil, errMalformedImage
	}
	out := make([]byte, 0, len(data))
	out = append(out, data[:i]...)

	// skipSubBlocks returns the index after a run of data sub-blocks starting at j.
	skipSubBlocks := func(j int) (int, error) {
		for {
			if j >= len(data) {
				return 0, errMalformedImage
			}
			n := int(data[j])
			j++
			if n == 0 {
				return j, nil
			}
			j += n
		}
	}

	for i < len(data) {
		switch data[i] {
		case 0x3B: // trailer
			return append(out, data[i]), nil
		case 0x21: // extension
			if i+2 > len(data) {
				return nil, errMalformedImage
			}
			label := data[i+1]
			end, err := skipSubBlocks(i + 2)
			if err != nil {
				return nil, err
			}
			keep := label != 0xFE // comment
			if label == 0xFF {
				app := data[i+2:]
				keep = len(app) >= 12 && app[0] == 11 &&
					(string(app[1:12]) == "NETSCAPE2.0" || string(app[1:12]) == "ANIMEXTS1.0")
			}
			if keep {
				out = append(out, data[i:end]...)
			}
			i = end
		case 0x2C: // image descriptor
			if i+10 > len(data) {
				return nil, errMalformedImage
			}
			j := i + 10
			if flags := data[i+9]; flags&0x80 != 0 {
				j += 3 << (int(flags&0x07) + 1) // local colour table
			}
			j++ // LZW minimum code size
			end, err := skipSubBlocks(j)
			if err != nil {
				return nil, err
			}
			out = append(out, data[i:end]...)
			i = end
		default:
			return nil, errMalformedImage
		}
	}
	return nil, errMalformedImage // no trailer
}

// stripWebpMetadata drops the EXIF and XMP chunks and clears their flags in the VP8X header.
func stripWebpMetadata(data []byte) ([]byte, error) {
	if !IsWebp(data) || len(data) < 12 || string(data[:4]) != "RIFF" {
		return nil, errMalformedImage
	}
	end := 8 + int(binary.LittleEndian.Uint32(data[4:8]))
	if end > len(data) || end < 12 {
		return nil, errMalformedImage
	}
	out := make([]byte, 0, len(data))
	out = append(out, data[:12]...)
	i := 12
	for i < end {
		if i+8 > end {
			return nil, errMalformedImage
		}
		n := int(binary.LittleEndian.Uint32(data[i+4 : i+8]))
		chunkEnd := i + 8 + n + n&1 // chunks are padded to an even size
		if n < 0 || chunkEnd > end || chunkEnd < i {
			return nil, errMalformedImage
		}
		switch string(data[i : i+4]) {
		case "EXIF", "XMP ":
		case "VP8X":
			start := len(out)
			out = append(out, data[i:chunkEnd]...)
			if n >= 1 {
				out[start+8] &^= 0x08 | 0x04 // EXIF and XMP metadata flags
			}
		default:
			out = append(out, data[i:chunkEnd]...)
		}
		i = chunkEnd
	}
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8)) //nolint:gosec // bounded by the input size
	return out, nil
}
