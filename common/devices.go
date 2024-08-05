package common

import "runtime"

const (
	AndroidPlatform = "android"
	IosPlatform     = "ios"
	WindowsPlatform = "windows"
)

var IsMobilePlatform = func() bool {
	return OperatingSystemIs(AndroidPlatform) || OperatingSystemIs(IosPlatform)
}

func OperatingSystemIs(targetOS string) bool {
	return runtime.GOOS == targetOS
}
