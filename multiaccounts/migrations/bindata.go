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

var __0001_accounts_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\x4c\x4e\xce\x2f\xcd\x2b\x29\xb6\xe6\x02\x04\x00\x00\xff\xff\x96\x1e\x13\xa1\x15\x00\x00\x00")

func _0001_accounts_down_sql() ([]byte, error) {
	return bindata_read(
		__0001_accounts_down_sql,
		"0001_accounts.down.sql",
	)
}

var __0001_accounts_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x54\xcb\x31\x8e\x84\x20\x14\x00\xd0\x9e\x53\xfc\x72\x37\xe1\x06\x5b\xa1\xcb\xae\x44\x47\x0d\x7e\x47\x2d\x7f\x84\x28\x99\x11\x8c\x30\x85\xb7\x9f\xc4\x6e\xda\x97\xbc\x5c\x4b\x81\x12\x50\x64\x95\x04\xf5\x07\x75\x83\x20\x47\xd5\x61\x07\x34\xcf\xe1\xe5\x53\x84\x2f\x46\xc6\x1c\x36\x46\xb8\x0b\x9d\x17\x42\x43\xab\xd5\x4d\xe8\x09\x4a\x39\x71\xe6\x69\xb3\x80\x72\xc4\x2b\xd7\x7d\x55\x71\xf6\x0c\x8b\xf3\xe8\x36\x1b\x13\x6d\x3b\x64\xea\x1f\x54\x8d\x9c\xed\x6b\x48\xa1\xa5\xb4\x5e\x81\xb3\x87\x3d\x67\x3a\x4c\x4b\xee\x70\x7e\xf9\xc4\xd2\x9e\xbd\x33\x97\xb1\x6f\x18\x14\x16\x4d\x8f\xa0\x9b\x41\xfd\xfe\xb0\x77\x00\x00\x00\xff\xff\x96\x21\x96\xe1\xb8\x00\x00\x00")

func _0001_accounts_up_sql() ([]byte, error) {
	return bindata_read(
		__0001_accounts_up_sql,
		"0001_accounts.up.sql",
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
	"0001_accounts.down.sql": _0001_accounts_down_sql,
	"0001_accounts.up.sql": _0001_accounts_up_sql,
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
	"0001_accounts.down.sql": &_bintree_t{_0001_accounts_down_sql, map[string]*_bintree_t{
	}},
	"0001_accounts.up.sql": &_bintree_t{_0001_accounts_up_sql, map[string]*_bintree_t{
	}},
	"doc.go": &_bintree_t{doc_go, map[string]*_bintree_t{
	}},
}}
