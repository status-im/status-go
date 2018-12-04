# Mobile

Package mobile implements [gomobile](https://github.com/golang/mobile) bindings for status-go. Current implementation servers as a drop-in replacement for `lib` package.

The framework name is generated from the package name, hence these things are done intentionally:
(1) this package's name isn't equal to the directory name (`statusgo` vs `mobile` respectively);
(2) this package name is `statusgo` and not `status` which produces the right framework name.

# Usage

For properly using this package, please refer to Makefile in the root of `status-go` directory.

To manually build library, run following commands:

### iOS

```
gomobile bind -v -target=ios -ldflags="-s -w" github.com/status-im/status-go/mobile
```
This will produce `Statusgo.framework` file in the current directory, which can be used in iOS project.

### Android

```
gomobile bind -v -target=android -ldflags="-s -w" github.com/status-im/status-go/mobile
```
This will generate `Statusgo.aar` file in the current dir.

# Notes

See [https://github.com/golang/go/wiki/Mobile](https://github.com/golang/go/wiki/Mobile) for more information on `gomobile` usage.
