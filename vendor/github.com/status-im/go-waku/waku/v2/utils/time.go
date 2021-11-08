package utils

import "time"

// GetUnixEpoch converts a time into a unix timestamp with the integer part
// representing seconds and the decimal part representing subseconds
func GetUnixEpochFrom(now time.Time) float64 {
	return float64(now.UnixNano()) / float64(time.Second)
}

// GetUnixEpoch returns the current time in unix timestamp with the integer part
// representing seconds and the decimal part representing subseconds
func GetUnixEpoch() float64 {
	return GetUnixEpochFrom(time.Now())
}
