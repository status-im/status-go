package utils

import "time"

func GetUnixEpoch() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Second)
}
