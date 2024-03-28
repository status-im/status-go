# Known Issues

## Golang version mismatch

If the go compilation run in error with a version mismatch, unset the variable `GOROOT`

```
compile: version "go1.20.13" does not match go tool version "go1.19.9"
# golang.org/x/text/internal/utf8internal
```
