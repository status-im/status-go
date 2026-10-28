// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build arm64 && gc && !purego

#include "textflag.h"

// The 16 words of a BlaMka round sit in V0-V7, two words per register, so every
// step below runs four G functions: (V0,V2,V4,V6) and (V1,V3,V5,V7) side by side.
// V8-V13 are scratch, V14/V15 hold the byte shuffles for the 24/16-bit rotations.

// UMLAL Vd.2D, V8.2S, V9.2S and UMLAL Vd.2D, V11.2S, V12.2S.
#define UMLAL_0(d) WORD $(0x2ea98100 | d)
#define UMLAL_1(d) WORD $(0x2eac8160 | d)

// a += b + 2*lo32(a)*lo32(b); na0/na1 are the register numbers of a0/a1.
#define FBLAMKA(a0, b0, a1, b1, na0, na1) \
	VUZP1 a0.S4, a0.S4, V8.S4;  \
	VUZP1 b0.S4, b0.S4, V9.S4;  \
	VUZP1 a1.S4, a1.S4, V11.S4; \
	VUZP1 b1.S4, b1.S4, V12.S4; \
	VADD  b0.D2, a0.D2, a0.D2;  \
	VADD  b1.D2, a1.D2, a1.D2;  \
	UMLAL_0(na0);               \
	UMLAL_1(na1);               \
	UMLAL_0(na0);               \
	UMLAL_1(na1)

// x = (x ^ y) >>> 32
#define XOR_ROTR32(x0, y0, x1, y1) \
	VEOR   y0.B16, x0.B16, x0.B16; \
	VEOR   y1.B16, x1.B16, x1.B16; \
	VREV64 x0.S4, x0.S4;           \
	VREV64 x1.S4, x1.S4

// x = (x ^ y) >>> 24 or 16, picked by the shuffle in tbl
#define XOR_ROTR_TBL(x0, y0, x1, y1, tbl) \
	VEOR y0.B16, x0.B16, x0.B16;          \
	VEOR y1.B16, x1.B16, x1.B16;          \
	VTBL tbl.B16, [x0.B16], x0.B16;       \
	VTBL tbl.B16, [x1.B16], x1.B16

// x = (x ^ y) >>> 63
#define XOR_ROTR63(x0, y0, x1, y1) \
	VEOR  y0.B16, x0.B16, x0.B16;  \
	VEOR  y1.B16, x1.B16, x1.B16;  \
	VUSHR $63, x0.D2, V8.D2;       \
	VUSHR $63, x1.D2, V11.D2;      \
	VADD  x0.D2, x0.D2, x0.D2;     \
	VADD  x1.D2, x1.D2, x1.D2;     \
	VEOR  V8.B16, x0.B16, x0.B16;  \
	VEOR  V11.B16, x1.B16, x1.B16

#define G(a0, b0, c0, d0, a1, b1, c1, d1, nc0, nc1) \
	FBLAMKA(a0, b0, a1, b1, 0, 1);                  \
	XOR_ROTR32(d0, a0, d1, a1);                     \
	FBLAMKA(c0, d0, c1, d1, nc0, nc1);              \
	XOR_ROTR_TBL(b0, c0, b1, c1, V14);              \
	FBLAMKA(a0, b0, a1, b1, 0, 1);                  \
	XOR_ROTR_TBL(d0, a0, d1, a1, V15);              \
	FBLAMKA(c0, d0, c1, d1, nc0, nc1);              \
	XOR_ROTR63(b0, c0, b1, c1)

// Columns, then diagonals: V16-V19 hold (v5,v6), (v7,v4), (v15,v12), (v13,v14).
#define ROUND \
	G(V0, V2, V4, V6, V1, V3, V5, V7, 4, 5);     \
	VEXT $8, V3.B16, V2.B16, V16.B16;      \
	VEXT $8, V2.B16, V3.B16, V17.B16;      \
	VEXT $8, V6.B16, V7.B16, V18.B16;      \
	VEXT $8, V7.B16, V6.B16, V19.B16;      \
	G(V0, V16, V5, V18, V1, V17, V4, V19, 5, 4); \
	VEXT $8, V16.B16, V17.B16, V2.B16;     \
	VEXT $8, V17.B16, V16.B16, V3.B16;     \
	VEXT $8, V19.B16, V18.B16, V6.B16;     \
	VEXT $8, V18.B16, V19.B16, V7.B16

// func blamkaNEON(b *block)
TEXT ·blamkaNEON(SB), NOSPLIT, $0-8
	MOVD b+0(FP), R0
	MOVD $·rotr24<>(SB), R3
	VLD1 (R3), [V14.B16]
	MOVD $·rotr16<>(SB), R3
	VLD1 (R3), [V15.B16]

	// Rows: 16 consecutive words each.
	MOVD R0, R1
	MOVD $8, R2

rows:
	ADD  $64, R1, R3
	VLD1 (R1), [V0.D2, V1.D2, V2.D2, V3.D2]
	VLD1 (R3), [V4.D2, V5.D2, V6.D2, V7.D2]
	ROUND
	VST1.P [V0.D2, V1.D2, V2.D2, V3.D2], 64(R1)
	VST1.P [V4.D2, V5.D2, V6.D2, V7.D2], 64(R1)
	SUB  $1, R2
	CBNZ R2, rows

	// Columns: word pairs 2i, 2i+1 of every row.
	MOVD R0, R1
	MOVD $8, R2

columns:
	FMOVQ (R1), F0
	FMOVQ 128(R1), F1
	FMOVQ 256(R1), F2
	FMOVQ 384(R1), F3
	FMOVQ 512(R1), F4
	FMOVQ 640(R1), F5
	FMOVQ 768(R1), F6
	FMOVQ 896(R1), F7
	ROUND
	FMOVQ F0, (R1)
	FMOVQ F1, 128(R1)
	FMOVQ F2, 256(R1)
	FMOVQ F3, 384(R1)
	FMOVQ F4, 512(R1)
	FMOVQ F5, 640(R1)
	FMOVQ F6, 768(R1)
	FMOVQ F7, 896(R1)
	ADD   $16, R1
	SUB   $1, R2
	CBNZ  R2, columns
	RET

// func mixBlocksNEON(out, a, b, c *block)
TEXT ·mixBlocksNEON(SB), NOSPLIT, $0-32
	MOVD out+0(FP), R0
	MOVD a+8(FP), R1
	MOVD b+16(FP), R2
	MOVD c+24(FP), R3
	MOVD $16, R4

mix:
	VLD1.P 64(R1), [V0.B16, V1.B16, V2.B16, V3.B16]
	VLD1.P 64(R2), [V4.B16, V5.B16, V6.B16, V7.B16]
	VLD1.P 64(R3), [V8.B16, V9.B16, V10.B16, V11.B16]
	VEOR   V4.B16, V0.B16, V0.B16
	VEOR   V5.B16, V1.B16, V1.B16
	VEOR   V6.B16, V2.B16, V2.B16
	VEOR   V7.B16, V3.B16, V3.B16
	VEOR   V8.B16, V0.B16, V0.B16
	VEOR   V9.B16, V1.B16, V1.B16
	VEOR   V10.B16, V2.B16, V2.B16
	VEOR   V11.B16, V3.B16, V3.B16
	VST1.P [V0.B16, V1.B16, V2.B16, V3.B16], 64(R0)
	SUB    $1, R4
	CBNZ   R4, mix
	RET

// func xorBlocksNEON(out, a, b, c *block)
TEXT ·xorBlocksNEON(SB), NOSPLIT, $0-32
	MOVD out+0(FP), R0
	MOVD a+8(FP), R1
	MOVD b+16(FP), R2
	MOVD c+24(FP), R3
	MOVD $16, R4

xor:
	VLD1   (R0), [V12.B16, V13.B16, V14.B16, V15.B16]
	VLD1.P 64(R1), [V0.B16, V1.B16, V2.B16, V3.B16]
	VLD1.P 64(R2), [V4.B16, V5.B16, V6.B16, V7.B16]
	VLD1.P 64(R3), [V8.B16, V9.B16, V10.B16, V11.B16]
	VEOR   V4.B16, V0.B16, V0.B16
	VEOR   V5.B16, V1.B16, V1.B16
	VEOR   V6.B16, V2.B16, V2.B16
	VEOR   V7.B16, V3.B16, V3.B16
	VEOR   V8.B16, V12.B16, V12.B16
	VEOR   V9.B16, V13.B16, V13.B16
	VEOR   V10.B16, V14.B16, V14.B16
	VEOR   V11.B16, V15.B16, V15.B16
	VEOR   V0.B16, V12.B16, V12.B16
	VEOR   V1.B16, V13.B16, V13.B16
	VEOR   V2.B16, V14.B16, V14.B16
	VEOR   V3.B16, V15.B16, V15.B16
	VST1.P [V12.B16, V13.B16, V14.B16, V15.B16], 64(R0)
	SUB    $1, R4
	CBNZ   R4, xor
	RET

DATA ·rotr24<>+0(SB)/8, $0x0201000706050403
DATA ·rotr24<>+8(SB)/8, $0x0a09080f0e0d0c0b
GLOBL ·rotr24<>(SB), RODATA|NOPTR, $16

DATA ·rotr16<>+0(SB)/8, $0x0100070605040302
DATA ·rotr16<>+8(SB)/8, $0x09080f0e0d0c0b0a
GLOBL ·rotr16<>(SB), RODATA|NOPTR, $16
