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

var __0001_settings_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x4e\x2d\x29\xc9\xcc\x4b\x2f\xb6\xe6\x42\x12\x4c\x4c\x4e\xce\x2f\xcd\x2b\x29\xb6\xe6\x02\x04\x00\x00\xff\xff\x2c\x0a\x12\xf1\x2a\x00\x00\x00")

func _0001_settings_down_sql() ([]byte, error) {
	return bindata_read(
		__0001_settings_down_sql,
		"0001_settings.down.sql",
	)
}

var __0001_settings_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\xce\x4f\x4b\xc3\x30\x1c\xc6\xf1\x7b\x5e\xc5\x73\x74\xb0\x77\xd0\x53\xba\xfe\x4a\x83\x35\xd1\x2c\xb1\xdb\x69\x84\x2e\x68\x21\xc6\x3f\x49\x14\xdf\xbd\x94\x4d\xf1\xd0\xed\xf8\xf0\xc0\x87\xef\x46\x13\x37\x04\xc3\xeb\x9e\x20\x5a\x48\x65\x40\x3b\xb1\x35\x5b\x24\x9f\xf3\x14\x9f\x12\x6e\x58\xfe\x7e\xf3\x78\xe4\x7a\xd3\x71\x8d\x7b\x2d\xee\xb8\xde\xe3\x96\xf6\x6b\xf6\xe9\x42\xf1\xa8\x7b\x55\xb3\x15\x06\x61\x3a\x65\x0d\xb4\x1a\x44\x53\x31\x76\x05\x77\xe3\xf8\x5a\x62\x9e\x71\x77\x3c\x7e\xf8\x94\x96\xfd\x17\x37\x45\xd4\x4a\xf5\xc4\x25\x1a\x6a\xb9\xed\x0d\x5a\x17\x92\x5f\xb3\x2f\x17\x82\xcf\x97\xdf\x3c\x3e\x2f\x9f\x97\x53\xad\x14\x0f\x96\x20\x64\x43\x3b\x94\x38\xbd\x17\x7f\x98\x13\x0e\xbf\x91\x4a\xfe\x4b\x9f\x9f\x15\x86\x8e\x34\x9d\x47\x75\x0d\x3a\xf5\x2e\x53\xa7\xef\x0f\x3b\xcf\x8a\xfd\x04\x00\x00\xff\xff\x74\x9d\x28\x7c\xa0\x01\x00\x00")

func _0001_settings_up_sql() ([]byte, error) {
	return bindata_read(
		__0001_settings_up_sql,
		"0001_settings.up.sql",
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
	"0001_settings.down.sql": _0001_settings_down_sql,
	"0001_settings.up.sql": _0001_settings_up_sql,
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
	"0001_settings.down.sql": &_bintree_t{_0001_settings_down_sql, map[string]*_bintree_t{
	}},
	"0001_settings.up.sql": &_bintree_t{_0001_settings_up_sql, map[string]*_bintree_t{
	}},
	"doc.go": &_bintree_t{doc_go, map[string]*_bintree_t{
	}},
}}
