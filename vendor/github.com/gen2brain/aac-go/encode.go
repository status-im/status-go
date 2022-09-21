// Package aac provides AAC codec encoder based on [VisualOn AAC encoder](https://github.com/mstorsjo/vo-aacenc) library.
package aac

import "C"

import (
	"io"
	"unsafe"

	"github.com/gen2brain/aac-go/aacenc"
)

// Options represent encoding options.
type Options struct {
	// Audio file sample rate
	SampleRate int
	// Encoder bit rate in bits/sec
	BitRate int
	// Number of channels on input (1,2)
	NumChannels int
}

// Encoder type.
type Encoder struct {
	w io.Writer

	insize int
	inbuf  []byte
	outbuf []byte
}

// NewEncoder returns new AAC encoder.
func NewEncoder(w io.Writer, opts *Options) (e *Encoder, err error) {
	e = &Encoder{}
	e.w = w

	if opts.BitRate == 0 {
		opts.BitRate = 64000
	}

	ret := aacenc.Init(aacenc.VoAudioCodingAac)
	err = aacenc.ErrorFromResult(ret)
	if err != nil {
		return
	}

	var params aacenc.AacencParam
	params.SampleRate = int32(opts.SampleRate)
	params.BitRate = int32(opts.BitRate)
	params.NChannels = int16(opts.NumChannels)
	params.AdtsUsed = 1

	ret = aacenc.SetParam(aacenc.VoPidAacEncparam, unsafe.Pointer(&params))
	err = aacenc.ErrorFromResult(ret)
	if err != nil {
		return
	}

	e.insize = int(opts.NumChannels) * 2 * 1024

	e.inbuf = make([]byte, e.insize)
	e.outbuf = make([]byte, 20480)

	return
}

// Encode encodes data from reader.
func (e *Encoder) Encode(r io.Reader) (err error) {
	for {
		n, err := r.Read(e.inbuf)
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		if n < e.insize {
			break
		}

		var outinfo aacenc.VoAudioOutputinfo
		var input, output aacenc.VoCodecBuffer

		input.Buffer = C.CBytes(e.inbuf)
		input.Length = uint64(n)

		ret := aacenc.SetInputData(&input)
		err = aacenc.ErrorFromResult(ret)
		if err != nil {
			return err
		}

		output.Buffer = C.CBytes(e.outbuf)
		output.Length = uint64(len(e.outbuf))

		ret = aacenc.GetOutputData(&output, &outinfo)
		err = aacenc.ErrorFromResult(ret)
		if err != nil {
			return err
		}

		_, err = e.w.Write(C.GoBytes(output.Buffer, C.int(output.Length)))
		if err != nil {
			return err
		}
	}

	return nil
}

// Close closes encoder.
func (e *Encoder) Close() error {
	ret := aacenc.Uninit()
	return aacenc.ErrorFromResult(ret)
}
