//go:build tools

package main

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/cheekybits/genny"
	_ "github.com/go-bindata/go-bindata"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/matryer/moq"
	_ "github.com/mjibson/esc"
	_ "go.uber.org/mock/mockgen"
	_ "golang.org/x/tools/cmd/stringer"
	_ "gotest.tools/gotestsum"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
