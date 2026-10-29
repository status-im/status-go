# Building the Nim libraries

The tasks `libsds`, `libsdsAndroid` and `libsdsIOS` build the pinned nim-sds into `LIB_OUT_DIR`
(one per library and platform; default `build/nim/sds/<os>-<cpu>`), with `NIM_PARAMS` = the
caller's flags + `nim/lib_params` (behaviour-neutral; ORC is a later change) + the resolution's
`--path`s, per the [Nim library build contract](https://github.com/status-im/status-app/issues/22982).
An embedder runs them with plain `nim` (`NIM_PARAMS` is POSIX shell syntax):

```sh
printf 'include "%s"\n' <status_go>/status_go.nimble > "$LIB_OUT_DIR/status_go.nims"
nim --skipUserCfg --skipParentCfg libsds "$LIB_OUT_DIR/status_go.nims"
```

`make build-libsds` (and `-android`, `-ios`) run these tasks and copy the result to
`NIM_SDS_LIB_DIR`, as before.
