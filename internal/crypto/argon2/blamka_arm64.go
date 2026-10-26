// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build arm64 && gc && !purego

package argon2

//go:noescape
func mixBlocksNEON(out, a, b, c *block)

//go:noescape
func xorBlocksNEON(out, a, b, c *block)

//go:noescape
func blamkaNEON(b *block)

func processBlockNEON(out, in1, in2 *block, xor bool) {
	var t block
	mixBlocksNEON(&t, in1, in2, &t)
	blamkaNEON(&t)
	if xor {
		xorBlocksNEON(out, in1, in2, &t)
	} else {
		mixBlocksNEON(out, in1, in2, &t)
	}
}

func processBlock(out, in1, in2 *block) {
	processBlockNEON(out, in1, in2, false)
}

func processBlockXOR(out, in1, in2 *block) {
	processBlockNEON(out, in1, in2, true)
}
