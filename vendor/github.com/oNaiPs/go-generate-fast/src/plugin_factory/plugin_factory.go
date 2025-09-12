package plugin_factory

import (
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/controller-gen"
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/esc"
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/genny"
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/go-bindata"
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/gqlgen"
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/mockgen"
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/moq"
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/protoc"
	_ "github.com/oNaiPs/go-generate-fast/src/plugins/stringer"
)

func Init() {}
