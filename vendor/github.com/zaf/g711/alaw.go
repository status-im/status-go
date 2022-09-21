/*
	Copyright (C) 2016 - 2017, Lefteris Zafiris <zaf@fastmail.com>

	This program is free software, distributed under the terms of
	the BSD 3-Clause License. See the LICENSE file
	at the top of the source tree.

	Package g711 implements encoding and decoding of G711 PCM sound data.
	G.711 is an ITU-T standard for audio companding.
*/

package g711

const alawClip = 0x7F7B

var (
	// A-law quantization segment lookup table
	alawSegment = [128]uint8{
		1, 1, 2, 2, 3, 3, 3, 3,
		4, 4, 4, 4, 4, 4, 4, 4,
		5, 5, 5, 5, 5, 5, 5, 5,
		5, 5, 5, 5, 5, 5, 5, 5,
		6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6,
		6, 6, 6, 6, 6, 6, 6, 6,
		7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7,
		7, 7, 7, 7, 7, 7, 7, 7,
	}
	// A-law to LPCM conversion lookup table
	alaw2lpcm = [256]int16{
		-5504, -5248, -6016, -5760, -4480, -4224, -4992, -4736,
		-7552, -7296, -8064, -7808, -6528, -6272, -7040, -6784,
		-2752, -2624, -3008, -2880, -2240, -2112, -2496, -2368,
		-3776, -3648, -4032, -3904, -3264, -3136, -3520, -3392,
		-22016, -20992, -24064, -23040, -17920, -16896, -19968, -18944,
		-30208, -29184, -32256, -31232, -26112, -25088, -28160, -27136,
		-11008, -10496, -12032, -11520, -8960, -8448, -9984, -9472,
		-15104, -14592, -16128, -15616, -13056, -12544, -14080, -13568,
		-344, -328, -376, -360, -280, -264, -312, -296,
		-472, -456, -504, -488, -408, -392, -440, -424,
		-88, -72, -120, -104, -24, -8, -56, -40,
		-216, -200, -248, -232, -152, -136, -184, -168,
		-1376, -1312, -1504, -1440, -1120, -1056, -1248, -1184,
		-1888, -1824, -2016, -1952, -1632, -1568, -1760, -1696,
		-688, -656, -752, -720, -560, -528, -624, -592,
		-944, -912, -1008, -976, -816, -784, -880, -848,
		5504, 5248, 6016, 5760, 4480, 4224, 4992, 4736,
		7552, 7296, 8064, 7808, 6528, 6272, 7040, 6784,
		2752, 2624, 3008, 2880, 2240, 2112, 2496, 2368,
		3776, 3648, 4032, 3904, 3264, 3136, 3520, 3392,
		22016, 20992, 24064, 23040, 17920, 16896, 19968, 18944,
		30208, 29184, 32256, 31232, 26112, 25088, 28160, 27136,
		11008, 10496, 12032, 11520, 8960, 8448, 9984, 9472,
		15104, 14592, 16128, 15616, 13056, 12544, 14080, 13568,
		344, 328, 376, 360, 280, 264, 312, 296,
		472, 456, 504, 488, 408, 392, 440, 424,
		88, 72, 120, 104, 24, 8, 56, 40,
		216, 200, 248, 232, 152, 136, 184, 168,
		1376, 1312, 1504, 1440, 1120, 1056, 1248, 1184,
		1888, 1824, 2016, 1952, 1632, 1568, 1760, 1696,
		688, 656, 752, 720, 560, 528, 624, 592,
		944, 912, 1008, 976, 816, 784, 880, 848,
	}
	// A-law to u-law conversion lookup table based on the ITU-T G.711 specification
	alaw2ulaw = [256]uint8{
		42, 43, 40, 41, 46, 47, 44, 45, 34, 35, 32, 33, 38, 39, 36, 37,
		57, 58, 55, 56, 61, 62, 59, 60, 49, 50, 47, 48, 53, 54, 51, 52,
		10, 11, 8, 9, 14, 15, 12, 13, 2, 3, 0, 1, 6, 7, 4, 5,
		26, 27, 24, 25, 30, 31, 28, 29, 18, 19, 16, 17, 22, 23, 20, 21,
		98, 99, 96, 97, 102, 103, 100, 101, 93, 93, 92, 92, 95, 95, 94, 94,
		116, 118, 112, 114, 124, 126, 120, 122, 106, 107, 104, 105, 110, 111, 108, 109,
		72, 73, 70, 71, 76, 77, 74, 75, 64, 65, 63, 63, 68, 69, 66, 67,
		86, 87, 84, 85, 90, 91, 88, 89, 79, 79, 78, 78, 82, 83, 80, 81,
		170, 171, 168, 169, 174, 175, 172, 173, 162, 163, 160, 161, 166, 167, 164, 165,
		185, 186, 183, 184, 189, 190, 187, 188, 177, 178, 175, 176, 181, 182, 179, 180,
		138, 139, 136, 137, 142, 143, 140, 141, 130, 131, 128, 129, 134, 135, 132, 133,
		154, 155, 152, 153, 158, 159, 156, 157, 146, 147, 144, 145, 150, 151, 148, 149,
		226, 227, 224, 225, 230, 231, 228, 229, 221, 221, 220, 220, 223, 223, 222, 222,
		244, 246, 240, 242, 252, 254, 248, 250, 234, 235, 232, 233, 238, 239, 236, 237,
		200, 201, 198, 199, 204, 205, 202, 203, 192, 193, 191, 191, 196, 197, 194, 195,
		214, 215, 212, 213, 218, 219, 216, 217, 207, 207, 206, 206, 210, 211, 208, 209,
	}
)

// EncodeAlaw encodes 16bit LPCM data to G711 A-law PCM
func EncodeAlaw(lpcm []byte) []byte {
	if len(lpcm) < 2 {
		return []byte{}
	}
	alaw := make([]byte, len(lpcm)/2)
	for i, j := 0, 0; j <= len(lpcm)-2; i, j = i+1, j+2 {
		alaw[i] = EncodeAlawFrame(int16(lpcm[j]) | int16(lpcm[j+1])<<8)
	}
	return alaw
}

// EncodeAlawFrame encodes a 16bit LPCM frame to G711 A-law PCM
func EncodeAlawFrame(frame int16) uint8 {
	/*
		The algorithm first stores off the sign. Then the code branches.
		If the absolute value of the source sample is less than 256, the 16-bit sample is simply
		shifted down 4 bits and converted to an 8-bit value, thus losing the top 4 bits in the process.
		However, if it is more than 256, a logarithmic algorithm is applied to the sample to
		determine the precision to keep. In that case, the sample is shifted down to access the
		seven most significant bits of the sample. Those seven bits are then used to determine the
		precision of the bottom 4 bits (segment). Finally, the top seven bits are shifted back up four bits
		to make room for the bottom 4 bits. The two are then logically OR'd together to create the
		eight bit compressed sample. The sign is then applied, and the entire compressed sample
		is logically XOR'd for transmission.
	*/
	sign := ((^frame) >> 8) & 0x80
	if sign == 0 {
		frame = -frame
	}
	if frame > alawClip {
		frame = alawClip
	}
	var compressedByte uint8
	if frame >= 256 {
		segment := alawSegment[(frame>>8)&0x7F]
		bottom := (frame >> (segment + 3)) & 0x0F
		compressedByte = uint8(((int16(segment) << 4) | bottom))
	} else {
		compressedByte = uint8(frame >> 4)
	}
	return compressedByte ^ uint8(sign^0x55)
}

// DecodeAlaw decodes A-law PCM data to 16bit LPCM
func DecodeAlaw(pcm []byte) []byte {
	lpcm := make([]byte, len(pcm)*2)
	for i, j := 0, 0; i < len(pcm); i, j = i+1, j+2 {
		frame := alaw2lpcm[pcm[i]]
		lpcm[j] = byte(frame)
		lpcm[j+1] = byte(frame >> 8)
	}
	return lpcm
}

// DecodeAlawFrame decodes an A-law PCM frame to 16bit LPCM
func DecodeAlawFrame(frame uint8) int16 {
	return alaw2lpcm[frame]
}

// Alaw2Ulaw performs direct A-law to u-law data conversion
func Alaw2Ulaw(alaw []byte) []byte {
	ulaw := make([]byte, len(alaw))
	for i := 0; i < len(alaw); i++ {
		ulaw[i] = alaw2ulaw[alaw[i]]
	}
	return ulaw
}

// Alaw2UlawFrame directly converts an A-law frame to u-law
func Alaw2UlawFrame(frame uint8) uint8 {
	return alaw2ulaw[frame]
}
