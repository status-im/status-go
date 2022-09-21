package riff

import (
	"encoding/binary"
	"io"
)

type Writer struct {
	io.Writer
}

type writeCallback func(w io.Writer)

func NewWriter(w io.Writer, fileType []byte, fileSize uint32) *Writer {
	w.Write([]byte("RIFF"))
	binary.Write(w, binary.LittleEndian, fileSize)
	w.Write(fileType)

	return &Writer{w}
}

func (w *Writer) WriteChunk(chunkID []byte, chunkSize uint32, cb writeCallback) (err error) {
	_, err = w.Write(chunkID)

	if err != nil {
		return
	}

	err = binary.Write(w, binary.LittleEndian, chunkSize)

	if err != nil {
		return
	}

	cb(w)

	return
}
