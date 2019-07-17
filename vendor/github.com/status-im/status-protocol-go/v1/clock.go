package statusproto

import "time"

const clockBumpInMs = int64(time.Minute / time.Millisecond)

// CalcMessageClock calculates a new clock value for Message.
// It is used to properly sort messages and accomodate the fact
// that time might be different on each device.
func CalcMessageClock(lastObservedValue int64, timeInMs TimestampInMs) int64 {
	clock := lastObservedValue
	if clock < int64(timeInMs) {
		// Added time should be larger than time skew tollerance for a message.
		// Here, we use 1 minute which is larger than accepted message time skew by Whisper.
		clock = int64(timeInMs) + clockBumpInMs
	} else {
		clock++
	}
	return clock
}
