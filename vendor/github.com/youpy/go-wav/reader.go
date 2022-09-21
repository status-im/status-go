package wav

import (
	"bufio"
	"encoding/binary"
	"errors"
	"math"
	"time"

	"github.com/youpy/go-riff"
	"github.com/zaf/g711"
)

type Reader struct {
	r         *riff.Reader
	riffChunk *riff.RIFFChunk
	format    *WavFormat
	*WavData
}

func NewReader(r riff.RIFFReader) *Reader {
	riffReader := riff.NewReader(r)
	return &Reader{r: riffReader}
}

func (r *Reader) Format() (format *WavFormat, err error) {
	if r.format == nil {
		format, err = r.readFormat()
		if err != nil {
			return
		}
		r.format = format
	} else {
		format = r.format
	}

	return
}

func (r *Reader) Duration() (time.Duration, error) {
	format, err := r.Format()
	if err != nil {
		return 0.0, err
	}

	err = r.loadWavData()
	if err != nil {
		return 0.0, err
	}

	sec := float64(r.WavData.Size) / float64(format.BlockAlign) / float64(format.SampleRate)

	return time.Duration(sec*1000000000) * time.Nanosecond, nil
}

func (r *Reader) Read(p []byte) (n int, err error) {
	err = r.loadWavData()
	if err != nil {
		return n, err
	}

	return r.WavData.Read(p)
}

func (r *Reader) ReadSamples(params ...uint32) (samples []Sample, err error) {
	var bytes []byte
	var numSamples, b, n int

	if len(params) > 0 {
		numSamples = int(params[0])
	} else {
		numSamples = 2048
	}

	format, err := r.Format()
	if err != nil {
		return
	}

	numChannels := int(format.NumChannels)
	blockAlign := int(format.BlockAlign)
	bitsPerSample := int(format.BitsPerSample)

	bytes = make([]byte, numSamples*blockAlign)
	n, err = r.Read(bytes)

	if err != nil {
		return
	}

	numSamples = n / blockAlign
	r.WavData.pos += uint32(numSamples * blockAlign)
	samples = make([]Sample, numSamples)
	offset := 0

	for i := 0; i < numSamples; i++ {
		for j := 0; j < numChannels; j++ {
			soffset := offset + (j * bitsPerSample / 8)

			switch format.AudioFormat {
			case AudioFormatIEEEFloat:
				bits :=
					uint32((int(bytes[soffset+3]) << 24) +
						(int(bytes[soffset+2]) << 16) +
						(int(bytes[soffset+1]) << 8) +
						int(bytes[soffset]))
				samples[i].Values[j] = int(math.MaxInt32 * math.Float32frombits(bits))

			case AudioFormatALaw:
				var val uint
				pcm := g711.DecodeAlaw(bytes[soffset : soffset+(bitsPerSample/8)])
				for b = 0; b < len(pcm); b++ {
					val += uint(pcm[b]) << uint(b*8)
				}

				samples[i].Values[j] = toInt(val, bitsPerSample*2)

			case AudioFormatMULaw:
				var val uint
				pcm := g711.DecodeUlaw(bytes[soffset : soffset+(bitsPerSample/8)])
				for b = 0; b < len(pcm); b++ {
					val += uint(pcm[b]) << uint(b*8)
				}

				samples[i].Values[j] = toInt(val, bitsPerSample*2)

			default:
				var val uint
				for b = 0; b*8 < bitsPerSample; b++ {
					val += uint(bytes[soffset+b]) << uint(b*8)
				}

				samples[i].Values[j] = toInt(val, bitsPerSample)
			}
		}

		offset += blockAlign
	}

	return
}

func (r *Reader) IntValue(sample Sample, channel uint) int {
	return sample.Values[channel]
}

func (r *Reader) FloatValue(sample Sample, channel uint) float64 {
	return float64(r.IntValue(sample, channel)) / math.Pow(2, float64(r.format.BitsPerSample)-1)
}

func (r *Reader) readFormat() (fmt *WavFormat, err error) {
	var riffChunk *riff.RIFFChunk

	fmt = new(WavFormat)

	if r.riffChunk == nil {
		riffChunk, err = r.r.Read()
		if err != nil {
			return
		}

		r.riffChunk = riffChunk
	} else {
		riffChunk = r.riffChunk
	}

	fmtChunk := findChunk(riffChunk, "fmt ")

	if fmtChunk == nil {
		err = errors.New("Format chunk is not found")
		return
	}

	err = binary.Read(fmtChunk, binary.LittleEndian, fmt)
	if err != nil {
		return
	}

	return
}

func (r *Reader) loadWavData() error {
	if r.WavData == nil {
		data, err := r.readData()
		if err != nil {
			return err
		}
		r.WavData = data
	}

	return nil
}

func (r *Reader) readData() (data *WavData, err error) {
	var riffChunk *riff.RIFFChunk

	if r.riffChunk == nil {
		riffChunk, err = r.r.Read()
		if err != nil {
			return
		}

		r.riffChunk = riffChunk
	} else {
		riffChunk = r.riffChunk
	}

	dataChunk := findChunk(riffChunk, "data")
	if dataChunk == nil {
		err = errors.New("Data chunk is not found")
		return
	}

	data = &WavData{bufio.NewReader(dataChunk), dataChunk.ChunkSize, 0}

	return
}

func findChunk(riffChunk *riff.RIFFChunk, id string) (chunk *riff.Chunk) {
	for _, ch := range riffChunk.Chunks {
		if string(ch.ChunkID[:]) == id {
			chunk = ch
			break
		}
	}

	return
}

func toInt(value uint, bits int) int {
	var result int

	switch bits {
	case 32:
		result = int(int32(value))
	case 16:
		result = int(int16(value))
	case 8:
		result = int(value)
	default:
		msb := uint(1 << (uint(bits) - 1))

		if value >= msb {
			result = -int((1 << uint(bits)) - value)
		} else {
			result = int(value)
		}
	}

	return result
}
