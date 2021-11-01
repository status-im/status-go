package utils

import "time"

func GetUnixEpochFrom(now func() time.Time) float64 {
	return float64(now().UnixNano()) / float64(time.Second)
}

func GetUnixEpoch() float64 {
	return GetUnixEpochFrom(time.Now)
}
