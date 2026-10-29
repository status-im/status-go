# status-go as a nimble package

The installed package is status-go's full source, so a dependent builds libstatus from its store
entry: `make -C <status_go> statusgo-shared-library STATUS_GO_BUILD_DIR=<out>` writes only under
`<out>`. `srcDir` is the empty `internal/nimble/src`, so no status-go directory reaches a
dependent's Nim module path. Install it in local-deps mode (`nimbledeps/`): nimble 0.26 global
installs flatten `srcDir` and drop the root files.

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
