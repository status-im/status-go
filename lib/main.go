package main

import "github.com/status-im/status-go/api"

var statusBackend = api.NewStatusBackend()

// main; Technically this package supposed to be a lib for
// cross-compilation and usage with Android/iOS, but
// without main it produces cryptic errors.
// TODO(divan): investigate the cause of the errors
// and change this package to be a library if possible.
func main() {}
