/*
	Copyright (C) 2016 - 2017, Lefteris Zafiris <zaf@fastmail.com>

	This program is free software, distributed under the terms of
	the BSD 3-Clause License. See the LICENSE file
	at the top of the source tree.

	Package g711 implements encoding and decoding of G711 PCM sound data.
	G.711 is an ITU-T standard for audio companding.
*/

package g711

const (
	uLawBias = 0x84
	uLawClip = 0x7F7B
)

var (
	// u-law quantization segment lookup table
	ulawSegment = [256]uint8{
		0, 0, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3,
		4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
		5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
		5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7, 7,
	}
	// u-law to LPCM conversion lookup table
	ulaw2lpcm = [256]int16{
		-32124, -31100, -30076, -29052, -28028, -27004, -25980, -24956,
		-23932, -22908, -21884, -20860, -19836, -18812, -17788, -16764,
		-15996, -15484, -14972, -14460, -13948, -13436, -12924, -12412,
		-11900, -11388, -10876, -10364, -9852, -9340, -8828, -8316,
		-7932, -7676, -7420, -7164, -6908, -6652, -6396, -6140,
		-5884, -5628, -5372, -5116, -4860, -4604, -4348, -4092,
		-3900, -3772, -3644, -3516, -3388, -3260, -3132, -3004,
		-2876, -2748, -2620, -2492, -2364, -2236, -2108, -1980,
		-1884, -1820, -1756, -1692, -1628, -1564, -1500, -1436,
		-1372, -1308, -1244, -1180, -1116, -1052, -988, -924,
		-876, -844, -812, -780, -748, -716, -684, -652,
		-620, -588, -556, -524, -492, -460, -428, -396,
		-372, -356, -340, -324, -308, -292, -276, -260,
		-244, -228, -212, -196, -180, -164, -148, -132,
		-120, -112, -104, -96, -88, -80, -72, -64,
		-56, -48, -40, -32, -24, -16, -8, 0,
		32124, 31100, 30076, 29052, 28028, 27004, 25980, 24956,
		23932, 22908, 21884, 20860, 19836, 18812, 17788, 16764,
		15996, 15484, 14972, 14460, 13948, 13436, 12924, 12412,
		11900, 11388, 10876, 10364, 9852, 9340, 8828, 8316,
		7932, 7676, 7420, 7164, 6908, 6652, 6396, 6140,
		5884, 5628, 5372, 5116, 4860, 4604, 4348, 4092,
		3900, 3772, 3644, 3516, 3388, 3260, 3132, 3004,
		2876, 2748, 2620, 2492, 2364, 2236, 2108, 1980,
		1884, 1820, 1756, 1692, 1628, 1564, 1500, 1436,
		1372, 1308, 1244, 1180, 1116, 1052, 988, 924,
		876, 844, 812, 780, 748, 716, 684, 652,
		620, 588, 556, 524, 492, 460, 428, 396,
		372, 356, 340, 324, 308, 292, 276, 260,
		244, 228, 212, 196, 180, 164, 148, 132,
		120, 112, 104, 96, 88, 80, 72, 64,
		56, 48, 40, 32, 24, 16, 8, 0,
	}
	// u-law to A-law conversion lookup table based on the ITU-T G.711 specification
	ulaw2alaw = [256]uint8{
		42, 43, 40, 41, 46, 47, 44, 45, 34, 35, 32, 33, 38, 39, 36, 37,
		58, 59, 56, 57, 62, 63, 60, 61, 50, 51, 48, 49, 54, 55, 52, 53,
		10, 11, 8, 9, 14, 15, 12, 13, 2, 3, 0, 1, 6, 7, 4, 26,
		27, 24, 25, 30, 31, 28, 29, 18, 19, 16, 17, 22, 23, 20, 21, 106,
		104, 105, 110, 111, 108, 109, 98, 99, 96, 97, 102, 103, 100, 101, 122, 120,
		126, 127, 124, 125, 114, 115, 112, 113, 118, 119, 116, 117, 75, 73, 79, 77,
		66, 67, 64, 65, 70, 71, 68, 69, 90, 91, 88, 89, 94, 95, 92, 93,
		82, 82, 83, 83, 80, 80, 81, 81, 86, 86, 87, 87, 84, 84, 85, 85,
		170, 171, 168, 169, 174, 175, 172, 173, 162, 163, 160, 161, 166, 167, 164, 165,
		186, 187, 184, 185, 190, 191, 188, 189, 178, 179, 176, 177, 182, 183, 180, 181,
		138, 139, 136, 137, 142, 143, 140, 141, 130, 131, 128, 129, 134, 135, 132, 154,
		155, 152, 153, 158, 159, 156, 157, 146, 147, 144, 145, 150, 151, 148, 149, 234,
		232, 233, 238, 239, 236, 237, 226, 227, 224, 225, 230, 231, 228, 229, 250, 248,
		254, 255, 252, 253, 242, 243, 240, 241, 246, 247, 244, 245, 203, 201, 207, 205,
		194, 195, 192, 193, 198, 199, 196, 197, 218, 219, 216, 217, 222, 223, 220, 221,
		210, 210, 211, 211, 208, 208, 209, 209, 214, 214, 215, 215, 212, 212, 213, 213,
	}
)

// EncodeUlaw encodes 16bit LPCM data to G711 u-law PCM
func EncodeUlaw(lpcm []byte) []byte {
	if len(lpcm) < 2 {
		return []byte{}
	}
	ulaw := make([]byte, len(lpcm)/2)
	for i, j := 0, 0; j <= len(lpcm)-2; i, j = i+1, j+2 {
		ulaw[i] = EncodeUlawFrame(int16(lpcm[j]) | int16(lpcm[j+1])<<8)
	}
	return ulaw
}

// EncodeUlawFrame encodes a 16bit LPCM frame to G711 u-law PCM
func EncodeUlawFrame(frame int16) uint8 {
	/*
		The algorithm first stores off the sign. It then adds in a bias value
		which (due to wrapping) will cause high valued samples to lose precision.
		The top five most significant bits are pulled out of the sample.
		Then, the bottom three bits of the compressed byte are generated using the
		segment look-up table, based on the biased value of the source sample.
		The 8-bit compressed sample is then finally created by logically OR'ing together
		the 5 most important bits, the 3 lower bits, and the sign when applicable. The bits
		are then logically NOT'ed for transmission.
	*/
	sign := (frame >> 8) & 0x80
	if sign != 0 {
		frame = -frame
	}
	if frame > uLawClip {
		frame = uLawClip
	}
	frame += uLawBias
	segment := ulawSegment[(frame>>7)&0xFF]
	bottom := (frame >> (segment + 3)) & 0x0F
	return uint8(^(sign | (int16(segment) << 4) | bottom))
}

// DecodeUlaw decodes u-law PCM data to 16bit LPCM
func DecodeUlaw(pcm []byte) []byte {
	lpcm := make([]byte, len(pcm)*2)
	for i, j := 0, 0; i < len(pcm); i, j = i+1, j+2 {
		frame := ulaw2lpcm[pcm[i]]
		lpcm[j] = byte(frame)
		lpcm[j+1] = byte(frame >> 8)
	}
	return lpcm
}

// DecodeUlawFrame decodes a u-law PCM frame to 16bit LPCM
func DecodeUlawFrame(frame uint8) int16 {
	return ulaw2lpcm[frame]
}

// Ulaw2Alaw performs direct u-law to A-law data conversion
func Ulaw2Alaw(ulaw []byte) []byte {
	alaw := make([]byte, len(ulaw))
	for i := 0; i < len(alaw); i++ {
		alaw[i] = ulaw2alaw[ulaw[i]]
	}
	return ulaw
}

// Ulaw2AlawFrame directly converts a u-law frame to A-law
func Ulaw2AlawFrame(frame uint8) uint8 {
	return ulaw2alaw[frame]
}
