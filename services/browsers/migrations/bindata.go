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

var __0001_browsers_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\x2a\xca\x2f\x2f\x4e\x2d\x2a\xb6\xe6\xc2\x22\x18\x9f\x91\x59\x5c\x92\x5f\x54\x69\xcd\x05\x08\x00\x00\xff\xff\x5b\xe2\x78\xbe\x32\x00\x00\x00")

func _0001_browsers_down_sql() ([]byte, error) {
	return bindata_read(
		__0001_browsers_down_sql,
		"0001_browsers.down.sql",
	)
}

var __0001_browsers_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x8f\xc1\x6a\x83\x40\x10\x86\xef\xf3\x14\xff\x51\xc1\x37\xe8\x69\xd5\xd1\x0e\xdd\xee\x96\x75\x24\xc9\x29\x58\xb4\x54\xa8\x89\xa8\xd0\xf6\xed\xcb\x92\x04\xaf\xbd\x7e\xcc\x37\xfc\x5f\x11\xd8\x28\x43\x4d\x6e\x19\x52\xc1\x79\x05\x1f\xa5\xd1\x06\xef\xcb\xf5\x7b\x1d\x96\x15\x09\x8d\x3d\x94\x8f\x8a\xb7\x20\xaf\x26\x9c\xf0\xc2\xa7\x8c\x2e\xdd\x34\xdc\x70\x94\x5c\x6b\x6d\x46\xdb\x38\x0d\xeb\xd6\x4d\x33\xda\xa6\x96\xda\x71\x89\x5c\x6a\x71\x9a\x51\xdf\xcd\x33\x72\xef\x2d\x1b\x87\x92\x2b\xd3\x5a\xc5\x47\xf7\xb5\x0e\x19\x7d\x8e\xeb\x76\x5d\x7e\xe5\xd2\x0f\x3f\x68\x5d\x73\x33\xc5\x29\xa5\x38\x88\x3e\xfb\x56\x11\xfc\x41\xca\x27\xa2\x7f\x2c\x3e\xdf\xff\x21\xa1\x3b\x3a\x3f\x0a\xf6\xa9\x8f\x9b\x88\x33\xaa\x7c\x60\xa9\x5d\x2c\x4b\x76\x27\x45\xe0\x8a\x03\xbb\x82\xf7\xef\x49\xe4\x3e\x36\x58\x56\x46\x61\x9a\xc2\x94\x4c\x29\xfd\x05\x00\x00\xff\xff\x69\xa0\xeb\xdb\x4c\x01\x00\x00")

func _0001_browsers_up_sql() ([]byte, error) {
	return bindata_read(
		__0001_browsers_up_sql,
		"0001_browsers.up.sql",
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
	"0001_browsers.down.sql": _0001_browsers_down_sql,
	"0001_browsers.up.sql": _0001_browsers_up_sql,
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
	"0001_browsers.down.sql": &_bintree_t{_0001_browsers_down_sql, map[string]*_bintree_t{
	}},
	"0001_browsers.up.sql": &_bintree_t{_0001_browsers_up_sql, map[string]*_bintree_t{
	}},
	"doc.go": &_bintree_t{doc_go, map[string]*_bintree_t{
	}},
}}
