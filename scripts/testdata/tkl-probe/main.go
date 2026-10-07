// A minimal consumer for checking the final shared-library export policy.
package main

import "C"

import "github.com/status-im/nim-token-lists/go/tkl"

//export TokenLibraryProbe
func TokenLibraryProbe() C.uint { return C.uint(tkl.ABIVersion()) }

func main() {}
