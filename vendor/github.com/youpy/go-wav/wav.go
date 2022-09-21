package wav

import (
	"io"
)

const (
	AudioFormatPCM       = 1
	AudioFormatIEEEFloat = 3
	AudioFormatALaw      = 6
	AudioFormatMULaw     = 7
)

type WavFormat struct {
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

type WavData struct {
	io.Reader
	Size uint32
	pos  uint32
}

type Sample struct {
	Values [2]int
}
