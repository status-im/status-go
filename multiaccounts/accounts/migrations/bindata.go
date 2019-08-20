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

var __0001_settings_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x8f\xb1\x4e\xc3\x30\x10\x40\xf7\xfb\x8a\x1b\xa9\x94\x3f\xc8\xe4\xb4\x87\x62\x11\x6c\x70\x1d\x92\x4e\x95\x49\xad\xb6\x22\x24\x21\xb6\x41\xfd\x7b\x14\x97\x54\x1d\x4a\x37\x3f\xdf\xe9\xe9\xdd\x52\x11\xd3\x84\x9a\x65\x05\x21\x7f\x44\x21\x35\x52\xcd\xd7\x7a\x8d\xce\x7a\x7f\xec\xf6\x0e\x1f\xc0\x9f\x06\x8b\x6f\x4c\x2d\x73\xa6\xf0\x45\xf1\x67\xa6\x36\xf8\x44\x9b\x04\xbe\x4d\x1b\x2c\x66\x85\xcc\x60\x81\x15\xd7\xb9\x2c\x35\x2a\x59\xf1\x55\x0a\x70\x47\x6e\x9a\xa6\x0f\x9d\x9f\xe4\x66\xb7\x1b\xad\x73\xb7\xfd\x3f\xa6\x6d\xad\xc7\x4c\xca\x82\x98\x48\xa0\x39\x98\x2b\x8a\x5d\x9a\x6a\x9d\x80\xf3\xfd\x68\xf6\x33\x0d\xe1\xfd\xc3\x9e\x62\x57\x02\x83\xf1\x87\xbf\xff\xce\x7c\xce\x2b\x4d\xdf\xf6\x63\x7c\xff\x5f\x5e\x0a\xfe\x5a\x12\x72\xb1\xa2\x1a\x43\x77\xfc\x0a\x76\x7b\x2e\xda\xce\xd5\x52\x5c\xdd\x72\x9e\x2d\xb0\xca\x49\xd1\x05\xd3\x7b\xba\xe9\xa0\xdb\xb2\x69\x72\x51\x45\x48\xe1\x37\x00\x00\xff\xff\xfe\xcd\xfe\xc2\xaf\x01\x00\x00")

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
