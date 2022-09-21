// Package aacenc implements cgo bindings for [VisualOn AAC encoder library](https://github.com/mstorsjo/vo-aacenc) library.
package aacenc

//#include "voAAC.h"
import "C"

import (
	"errors"
	"unsafe"
)

// Constants.
const (
	// AAC Param ID
	VoPidAacMdoule   = 0x42211000
	VoPidAacEncparam = VoPidAacMdoule | 0x0040

	// AAC decoder error ID
	VoErrAacMdoule        = 0x82210000
	VoErrAacUnsfileformat = (VoErrAacMdoule | 0xF001)
	VoErrAacUnsprofile    = (VoErrAacMdoule | 0xF002)

	// The base param ID for AUDIO codec
	VoPidAudioBase = 0x42000000
	// The format data of audio in track
	VoPidAudioFormat = (VoPidAudioBase | 0x0001)
	// The sample rate of audio
	VoPidAudioSampleRate = (VoPidAudioBase | 0x0002)
	// The channel of audio
	VoPidAudioChannels = (VoPidAudioBase | 0x0003)
	// The bit rate of audio
	VoPidAudioBitrate = (VoPidAudioBase | 0x0004)
	// The channel mode of audio
	VoPidAudioChannelmode = (VoPidAudioBase | 0x0005)

	// The base of common param ID
	VoPidCommonBase = 0x40000000
	// Query the memory needed; Reserved
	VoPidCommonQueryMem = (VoPidCommonBase | 0)
	// Set or get the input buffer type
	VoPidCommonInputType = (VoPidCommonBase | 0)
	// Query it has resource to be used
	VoPidCommonHasResource = (VoPidCommonBase | 0)
	// Decoder track header data
	VoPidCommonHeadData = (VoPidCommonBase | 0)
	// VoPidCommonFlush as defined in include/voIndex.h:182
	VoPidCommonFlush = (VoPidCommonBase | 0)
)

// Error codes.
const (
	VoErrNone              = 0x00000000
	VoErrFinish            = 0x00000001
	VoErrFailed            = 0x80000001
	VoErrOutofMemory       = 0x80000002
	VoErrNotImplement      = 0x80000003
	VoErrInvalidArg        = 0x80000004
	VoErrInputBufferSmall  = 0x80000005
	VoErrOutputBufferSmall = 0x80000006
	VoErrWrongStatus       = 0x80000007
	VoErrWrongParamId      = 0x80000008
	VoErrLicenseError      = 0x80000009

	VoErrAudioBase          = 0x82000000
	VoErrAudioUnsChannel    = VoErrAudioBase | 0x0001
	VoErrAudioUnsSampleRate = VoErrAudioBase | 0x0002
	VoErrAudioUnsFeature    = VoErrAudioBase | 0x0003
)

// Enumeration used to define the possible audio coding formats.
const (
	// Placeholder value when coding is N/A
	VoAudioCodingUnused int32 = iota
	// Any variant of PCM coding
	VoAudioCodingPcm
	// Any variant of ADPCM encoded data
	VoAudioCodingAdpcm
	// Any variant of AMR encoded data
	VoAudioCodingAmrnb
	// Any variant of AMR encoded data
	VoAudioCodingAmrwb
	// Any variant of AMR encoded data
	VoAudioCodingAmrwbp
	// Any variant of QCELP 13kbps encoded data
	VoAudioCodingQcelp13
	// Any variant of EVRC encoded data
	VoAudioCodingEvrc
	// Any variant of AAC encoded data, 0xA106 - ISO/MPEG-4 AAC, 0xFF - AAC
	VoAudioCodingAac
	// Any variant of AC3 encoded data
	VoAudioCodingAc3
	// Any variant of FLAC encoded data
	VoAudioCodingFlac
	// Any variant of MP1 encoded data
	VoAudioCodingMp1
	// Any variant of MP3 encoded data
	VoAudioCodingMp3
	// Any variant of OGG encoded data
	VoAudioCodingOgg
	// Any variant of WMA encoded data
	VoAudioCodingWma
	// Any variant of RA encoded data
	VoAudioCodingRa
	// Any variant of MIDI encoded data
	VoAudioCodingMidi
	// Any variant of dra encoded data
	VoAudioCodingDra
	// Any variant of dra encoded data
	VoAudioCodingG729
)

// The frame type that the decoder supports.
const (
	// Contains only raw aac data in a frame
	VoAacRawdata int32 = iota
	// Contains ADTS header + raw AAC data in a frame
	VoAacAdts
)

// The channel type value.
const (
	// Center channel
	VoChannelCenter int32 = 1
	// Front left channel
	VoChannelFrontLeft = 1 << 1
	// Front right channel
	VoChannelFrontRight = 1 << 2
	// Side left channel
	VoChannelSideLeft = 1 << 3
	// Side right channel
	VoChannelSideRight = 1 << 4
	// Back left channel
	VoChannelBackLeft = 1 << 5
	// Back right channel
	VoChannelBackRight = 1 << 6
	// Back center channel
	VoChannelBackCenter = 1 << 7
	// Low-frequency effects bass channel
	VoChannelLfeBass = 1 << 8
	// Include all channels (default)
	VoChannelAll = 0xffff
)

// Input stream format, Frame or Stream.
const (
	// Input contains completely frame(s) data
	VoInputFrame int32 = iota + 1
	// Input is stream data.
	VoInputStream
)

// VoAudioFormat - general audio format info.
type VoAudioFormat struct {
	// Sample rate
	SampleRate int
	// Channel count
	Channels int
	// Bits per sample
	SampleBits int
}

// cptr return C pointer.
func (v *VoAudioFormat) cptr() *C.VO_AUDIO_FORMAT {
	return (*C.VO_AUDIO_FORMAT)(unsafe.Pointer(v))
}

// VoAudioOutputinfo - general audio output info.
type VoAudioOutputinfo struct {
	// Sample rate
	Format VoAudioFormat
	// Channel count
	InputUsed uint
	// Reserved
	Reserve uint
}

// cptr return C pointer.
func (v *VoAudioOutputinfo) cptr() *C.VO_AUDIO_OUTPUTINFO {
	return (*C.VO_AUDIO_OUTPUTINFO)(unsafe.Pointer(v))
}

// VoCodecBuffer - general data buffer, used as input or output.
type VoCodecBuffer struct {
	// Buffer pointer
	Buffer unsafe.Pointer
	// Buffer size in byte
	Length uint64
	// The time of the buffer
	Time int64
}

// cptr return C pointer.
func (v *VoCodecBuffer) cptr() *C.VO_CODECBUFFER {
	return (*C.VO_CODECBUFFER)(unsafe.Pointer(v))
}

// AacencParam - the structure for AAC encoder input parameter.
type AacencParam struct {
	// Audio file sample rate
	SampleRate int32
	// Encoder bit rate in bits/sec
	BitRate int32
	// Number of channels on input (1,2)
	NChannels int16
	// Whether write adts header
	AdtsUsed int16
}

var handle C.VO_HANDLE

// Errors.
var (
	ErrFinish            = errors.New("aac: error finish")
	ErrFailed            = errors.New("aac: process data failed")
	ErrOutOfMemory       = errors.New("aac: out of memory")
	ErrNotImplement      = errors.New("aac: feature not implemented")
	ErrInvalidArg        = errors.New("aac: invalid argument")
	ErrInputBufferSmall  = errors.New("aac: input buffer data too small")
	ErrOutputBufferSmall = errors.New("aac: output buffer size too small")
	ErrWrongStatus       = errors.New("aac: wrong encoder run-time status")
	ErrWrongParamId      = errors.New("aac: wrong parameter id")
	ErrLicenseError      = errors.New("aac: license error")

	ErrAudioBase          = errors.New("aac: error audio base")
	ErrAudioUnsChannel    = errors.New("aac: unsupported number of channel")
	ErrAudioUnsSampleRate = errors.New("aac: unsupported sample rate")
	ErrAudioUnsFeature    = errors.New("aac: unsupported feature")
)

// ErrorFromResult returns error for result code
func ErrorFromResult(r uint) error {
	switch r {
	case VoErrNone:
		return nil
	case VoErrFinish:
		return ErrFinish
	case VoErrFailed:
		return ErrFailed
	case VoErrOutofMemory:
		return ErrOutOfMemory
	case VoErrNotImplement:
		return ErrNotImplement
	case VoErrInvalidArg:
		return ErrInvalidArg
	case VoErrInputBufferSmall:
		return ErrInputBufferSmall
	case VoErrOutputBufferSmall:
		return ErrOutputBufferSmall
	case VoErrWrongStatus:
		return ErrWrongStatus
	case VoErrWrongParamId:
		return ErrWrongParamId
	case VoErrLicenseError:
		return ErrLicenseError
	case VoErrAudioBase:
		return ErrAudioBase
	case VoErrAudioUnsChannel:
		return ErrAudioUnsChannel
	case VoErrAudioUnsSampleRate:
		return ErrAudioUnsSampleRate
	case VoErrAudioUnsFeature:
		return ErrAudioUnsFeature
	default:
		return nil
	}
}

// Init - init the audio codec module and return codec handle.
func Init(vtype int32) uint {
	cvtype := (C.VO_AUDIO_CODINGTYPE)(vtype)
	ret := C.voAACEncInit(&handle, cvtype, nil)
	v := (uint)(ret)
	return v
}

// SetInputData - set input audio data.
func SetInputData(pinput *VoCodecBuffer) uint {
	cpinput := pinput.cptr()
	ret := C.voAACEncSetInputData(handle, cpinput)
	v := (uint)(ret)
	return v
}

// GetOutputData - get the outut audio data.
func GetOutputData(poutbuffer *VoCodecBuffer, poutinfo *VoAudioOutputinfo) uint {
	cpoutbuffer := poutbuffer.cptr()
	cpoutinfo := poutinfo.cptr()
	ret := C.voAACEncGetOutputData(handle, cpoutbuffer, cpoutinfo)
	v := (uint)(ret)
	return v
}

// SetParam - set the parameter for the specified param ID.
func SetParam(uparamid int, pdata unsafe.Pointer) uint {
	cuparamid := (C.VO_S32)(uparamid)
	cpdata := (C.VO_PTR)(pdata)
	ret := C.voAACEncSetParam(handle, cuparamid, cpdata)
	v := (uint)(ret)
	return v
}

// GetParam - get the parameter for the specified param ID.
func GetParam(uparamid int, pdata unsafe.Pointer) uint {
	cuparamid := (C.VO_S32)(uparamid)
	cpdata := (C.VO_PTR)(pdata)
	ret := C.voAACEncGetParam(handle, cuparamid, cpdata)
	v := (uint)(ret)
	return v
}

// Uninit - uninit the Codec.
func Uninit() uint {
	ret := C.voAACEncUninit(handle)
	v := (uint)(ret)
	return v
}
