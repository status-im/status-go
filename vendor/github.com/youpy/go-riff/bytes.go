package riff

import (
	"io"
)

type byteReader struct {
	offset uint32
	io.ReaderAt
}

func newByteReader(r io.ReaderAt) (bytes *byteReader) {
	bytes = &byteReader{0, r}

	return
}

func (bytes *byteReader) readLEUint32() uint32 {
	offset := bytes.offset
	data := make([]byte, 4)

	n, err := bytes.ReadAt(data, int64(offset))

	if err != nil || n < 4 {
		panic("Can't read bytes")
	}

	defer func() {
		bytes.offset += 4
	}()

	return uint32(data[3])<<24 +
		uint32(data[2])<<16 +
		uint32(data[1])<<8 +
		uint32(data[0])
}

func (bytes *byteReader) readLEUint16() uint16 {
	offset := bytes.offset
	data := make([]byte, 2)

	n, err := bytes.ReadAt(data, int64(offset))

	if err != nil || n < 2 {
		panic("Can't read bytes")
	}

	defer func() {
		bytes.offset += 2
	}()

	return uint16(data[1])<<8 + uint16(data[0])
}

func (bytes *byteReader) readLEInt16() int16 {
	offset := bytes.offset
	data := make([]byte, 2)

	n, err := bytes.ReadAt(data, int64(offset))

	if err != nil || n < 2 {
		panic("Can't read bytes")
	}

	defer func() {
		bytes.offset += 2
	}()

	return int16(data[offset+1])<<8 + int16(data[offset])
}

func (bytes *byteReader) readBytes(size uint32) []byte {
	offset := bytes.offset
	data := make([]byte, size)

	n, err := bytes.ReadAt(data, int64(offset))

	if err != nil || n < int(size) {
		panic("Can't read bytes")
	}

	defer func() {
		bytes.offset += size
	}()

	return data
}
