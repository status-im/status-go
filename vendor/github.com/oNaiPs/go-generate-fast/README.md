# go-generate-fast

[![build](https://github.com/oNaiPs/go-generate-fast/actions/workflows/build-test.yml/badge.svg)](https://github.com/oNaiPs/go-generate-fast/actions?query=branch%main)
[![Go Report
Card](https://goreportcard.com/badge/github.com/oNaiPs/go-generate-fast)](https://goreportcard.com/report/github.com/oNaiPs/go-generate-fast)

🚀 Shave off minutes and turn them into seconds for your go generation step 🚀.

`go-generate-fast` serves as a drop-in replacement for [go
generate](https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source).

Smart enough to understand if your generated files have changed or not,
`go-generate-fast` can circumvent running unaltered scripts, offering a
significant speed boost by harnessing smart caching mechanisms.

<https://github.com/oNaiPs/go-generate-fast/assets/374130/396a0160-90f8-46d0-a05c-783d127a384e>

## Features

- **Effortless Integration**: Seamlessly integrate it with your workflow through
  [comment directives](#usage).
- **Tool-awareness**: Executes tools only when necessary, based on changes
  identified in input files.
- **Comprehensive Tool Compatibility**: Works with a variety of `go:generate`
  tools, including
  [stringer](https://godoc.org/golang.org/x/tools/cmd/stringer),
  [mockgen](https://github.com/golang/mock/tree/master/mockgen),
  [esc](https://github.com/mjibson/esc), and [more](#supported-tools).
- **Support for Custom Scripts**: Manually specify inputs and outputs if needed.

## Install

```bash
go install github.com/oNaiPs/go-generate-fast
```

## Usage

Replace the traditional `go generate` command with `go-generate-fast` in your
scripts to leverage its benefits.

Typical command invocation is as follows:

```bash
go-generate-fast [file.go... | packages]
```

## Supported Tools

`go-generate-fast` automatically detects the input/output files for the
following tools:

- [stringer](https://godoc.org/golang.org/x/tools/cmd/stringer): Automated
  generation of methods satisfying the `fmt.Stringer` interface.
- [protobuf](https://developers.google.com/protocol-buffers): Go code generation
  for protobuf files
- [gqlgen](https://gqlgen.com/): Go code generation for GraphQL APIs
- [mockgen](https://github.com/uber-go/mock): Mock class generation for
  interfaces
- [moq](https://github.com/matryer/moq): Interface mocking tool for go generate
- [esc](https://github.com/mjibson/esc): Embedding static files in Go binaries
- [go-bindata](https://github.com/go-bindata/go-bindata): Turn data file into go
  code.
- [genny](https://github.com/cheekybits/genny): Elegant generics for Go.
- [controller-gen](https://book.kubebuilder.io/reference/controller-gen) Generates utility code and Kubernetes YAML.

Above commands can be called in binary form, or with go run. E.g.:

```go
//go:generate stringer

OR

//go:generate go run golang.org/x/tools/cmd/stringer[@latest]
```

### Custom Input/Output Files

If you are using a custom or currently unsupported script/tool, you can manually
add files as demonstrated below:

```go
//go:generate_input doc/gen_docs.go commands/**/*.go
//go:generate_output doc/man/*.1 doc/md/*.md
//go:generate go run doc/gen_docs.go

//go:generate_input images/*.png
//go:generate_output images/*.jpeg
//go:generate bash convert_images.sh
```

Just before your `go:generate` command, you can add any `go:generate_input` or
`go:generate_output` directives accurately. The input files influence the
tool/script (re)execution, while output files contain the results. If one or
more input files change, the command reruns and stores the output files in the
[cache directory](#configuration).

## Configuration

Various environment variables are available for configuration:

- `GO_GENERATE_FAST_DIR`: Sets the base directory for configurations and
  caching. Default locations differ by OS:
  - Linux: `$HOME/.config/go-generate-fast`
  - Darwin: `$HOME/Library/Application Support/go-generate-fast`
  - Windows: `%AppData%\go-generate-fast`

- `GO_GENERATE_FAST_CACHE_DIR`: Defines the cache files location. Default is
  `$GO_GENERATE_FAST_DIR/cache/`.
- `GO_GENERATE_FAST_DEBUG`: Enables debugging logs.
- `GO_GENERATE_FAST_DISABLE`: Completely ignores caching.
- `GO_GENERATE_FAST_READ_ONLY`: Uses the existing cache but prevents any new
  cache entries.
- `GO_GENERATE_FAST_FORCE_USE_CACHE`: Forceably uses cache. If it does not exist, the command fails.
- `GO_GENERATE_FAST_RECACHE`: Sets the cache to overwrite existing entries. The
  new results will be cached.

## How it Works

`go-generate-fast` makes regeneration faster by reusing previously outputs when
the same input files are used again. This approach is identified via the
creation of a unique hash that combines inputs,outputs and other metadata. The
underlying concept is similar to the C/C++ compiler cache
[ccache](https://ccache.dev/).

## Contributing

We highly appreciate community contributions! Please create an issue or submit a
pull request for any suggestions or issues.

## License

`go-generate-fast` is licensed under the [MIT License](LICENSE).
