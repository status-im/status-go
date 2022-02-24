package utils

import "time"

// GetUnixEpoch converts a time into a unix timestamp with nanoseconds
func GetUnixEpochFrom(now time.Time) int64 {
	return now.UnixNano()
}

// GetUnixEpoch returns the current time in unix timestamp with the integer part
// representing seconds and the decimal part representing subseconds
func GetUnixEpoch() int64 {
	return GetUnixEpochFrom(time.Now())
}
