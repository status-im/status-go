// +build gofuzz

package whisperv6

func Fuzz(data []byte) int {
	if len(data) < 2 {
		return -1
	}

	msg := &ReceivedMessage{Raw: data}
	msg.ValidateAndParse()

	return 0
}
