package logutils

// DefaultTruncateLength is the number of characters kept by TruncateWithDot.
const DefaultTruncateLength = 8

// TruncateWithDotN truncates a string by keeping n characters (excluding "..") from start and end,
// replacing middle part with "..".
// The total length of returned string will be n + 2 (where 2 is the length of "..").
// For n characters total (excluding dots), characters are distributed evenly between start and end.
// If n is odd, start gets one extra character.
// Examples:
//
//	TruncateWithDotN("0x123456789", 4) // returns "0x1..89" (total length 6)
//	TruncateWithDotN("0x123456789", 3) // returns "0x..9" (total length 5)
//	TruncateWithDotN("0x123456789", 5) // returns "0x1..89" (total length 7)
//	TruncateWithDotN("acdkef099", 3) // returns "ac..9" (total length 5)
//	TruncateWithDotN("ab", 4)        // returns "ab" (no truncation needed as len <= n)
//	TruncateWithDotN("test", 0)      // returns "test" (no truncation for n <= 0)
func TruncateWithDotN(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}

	if n < 2 { // need at least 2 chars (1 at start + 1 at end)
		n = 2
	}

	const dots = ".."
	// For n characters total (excluding dots), we want to keep:
	// - ceil(n/2) characters from the start
	// - floor(n/2) characters from the end
	startChars := (n + 1) / 2 // ceil(n/2)
	endChars := n / 2         // floor(n/2)

	return s[:startChars] + dots + s[len(s)-endChars:]
}

func TruncateWithDot(s string) string {
	return TruncateWithDotN(s, DefaultTruncateLength)
}
