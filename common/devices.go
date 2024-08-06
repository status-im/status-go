package common

import "runtime"

const (
	AndroidPlatform = "android"
	IOSPlatform     = "ios"
	WindowsPlatform = "windows"
)

var IsMobilePlatform = func() bool {
	return OperatingSystemIs(AndroidPlatform) || OperatingSystemIs(IOSPlatform)
}

func OperatingSystemIs(targetOS string) bool {
	return runtime.GOOS == targetOS
}
