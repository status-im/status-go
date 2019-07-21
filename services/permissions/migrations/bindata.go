package migrations

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var __0001_permissions_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\x49\x2c\x28\x28\xb6\xe6\x42\x12\x29\x48\x2d\xca\xcd\x2c\x2e\xce\xcc\xcf\x2b\xb6\xe6\x02\x04\x00\x00\xff\xff\xeb\x21\xe7\xd0\x2a\x00\x00\x00")

func _0001_permissions_down_sql() ([]byte, error) {
	return bindata_read(
		__0001_permissions_down_sql,
		"0001_permissions.down.sql",
	)
}

var __0001_permissions_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\xce\x31\x0f\x82\x30\x10\x05\xe0\xbd\xbf\xe2\x8d\x90\xf8\x0f\x9c\x6a\x79\x68\x63\x6d\x4d\x39\x02\x4c\x86\x44\x06\x06\x90\xc8\xff\x4f\x4c\xa3\x91\xc4\xc1\xf5\xee\xdd\x77\xcf\x44\x6a\x21\x44\x1f\x1c\x61\x4b\xf8\x20\x60\x6b\x2b\xa9\x70\xef\x97\x65\x45\xa6\xe6\x7e\x1a\x20\x6c\x05\xd7\x68\x2f\x3a\x76\x38\xb3\x53\x39\x1a\x2b\xa7\x50\x0b\x62\x68\x6c\xb1\x57\xea\x0f\xb5\x0c\xcf\x69\x5c\xd7\xf1\x31\x27\x30\xc1\xb7\x4d\x4d\x39\x5f\x3b\xb7\x53\x5b\xec\x77\x53\x86\x48\x7b\xf4\xe9\x73\xf6\x3d\xcf\x11\x59\x32\xd2\x1b\x7e\xda\x66\xef\x71\xf0\x28\xe8\x28\x84\xd1\x95\xd1\x05\x55\xfe\x0a\x00\x00\xff\xff\x9e\x9a\xc6\xf0\xe8\x00\x00\x00")

func _0001_permissions_up_sql() ([]byte, error) {
	return bindata_read(
		__0001_permissions_up_sql,
		"0001_permissions.up.sql",
	)
}

var _doc_go = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2c\xc9\xb1\x0d\xc4\x20\x0c\x05\xd0\x9e\x29\xfe\x02\xd8\xfd\x6d\xe3\x4b\xac\x2f\x44\x82\x09\x78\x7f\xa5\x49\xfd\xa6\x1d\xdd\xe8\xd8\xcf\x55\x8a\x2a\xe3\x47\x1f\xbe\x2c\x1d\x8c\xfa\x6f\xe3\xb4\x34\xd4\xd9\x89\xbb\x71\x59\xb6\x18\x1b\x35\x20\xa2\x9f\x0a\x03\xa2\xe5\x0d\x00\x00\xff\xff\x60\xcd\x06\xbe\x4a\x00\x00\x00")

func doc_go() ([]byte, error) {
	return bindata_read(
		_doc_go,
		"doc.go",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"0001_permissions.down.sql": _0001_permissions_down_sql,
	"0001_permissions.up.sql": _0001_permissions_up_sql,
	"doc.go": doc_go,
}
// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func func() ([]byte, error)
	Children map[string]*_bintree_t
}
var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"0001_permissions.down.sql": &_bintree_t{_0001_permissions_down_sql, map[string]*_bintree_t{
	}},
	"0001_permissions.up.sql": &_bintree_t{_0001_permissions_up_sql, map[string]*_bintree_t{
	}},
	"doc.go": &_bintree_t{doc_go, map[string]*_bintree_t{
	}},
}}
