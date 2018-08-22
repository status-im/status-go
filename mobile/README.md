# Mobile

Package mobile implements [gomobile](https://github.com/golang/mobile) bindings for status-go. Current implementation servers as a drop-in replacement for `lib` package.

# Usage

For properly using this package, please refer to Makefile in the root of `status-go` directory.

To manually build library, run following commands:

### iOS

```
gomobile bind -v -target=ios -ldflags="-s -w" github.com/status-im/status-go/mobile
```
This will produce `Status.framework` file in the current directory, which can be used in iOS project.

### Android

```
gomobile bind -v -target=android -ldflags="-s -w" github.com/status-im/status-go/mobile
```
This will generate `Status.aar` file in the current dir.

# Notes

See [https://github.com/golang/go/wiki/Mobile](https://github.com/golang/go/wiki/Mobile) for more information on `gomobile` usage.
