## v4

This is meant to satisfy go module versioning behavior; see
https://stackoverflow.com/a/52058348/329700 for more.

## 3.24.0

Remove uses of io/ioutil; you must use Go 1.18 or higher with this version of
go-bindata and its generated asset files.

Update generated doc comments for compatibility with Go's updated doc comment
guidelines.

## 3.21.0

Replace "Debug" with "AssetDebug" to reduce the likelihood of conflicts.

## 3.20.0

Add the "Debug" constant if assets have been generated using the `--debug` flag
at the command line.
