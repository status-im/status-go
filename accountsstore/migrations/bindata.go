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

var __0001_accounts_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x48\x4c\x4e\xce\x2f\xcd\x2b\x29\xb6\xe6\x42\x12\x4c\xce\xcf\x4b\xcb\x4c\x2f\x2d\x4a\x2c\xc9\xcc\xcf\x2b\xb6\xe6\x02\x04\x00\x00\xff\xff\x66\xfd\xe8\xb9\x30\x00\x00\x00")

func _0001_accounts_down_sql() ([]byte, error) {
	return bindata_read(
		__0001_accounts_down_sql,
		"0001_accounts.down.sql",
	)
}

var __0001_accounts_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\xcf\x41\x6e\xab\x30\x10\x06\xe0\xbd\x4f\x31\xcb\x20\x71\x83\xb7\x32\x66\x48\xac\x47\xed\x76\x18\x9a\x64\x85\x2c\xe2\x46\x48\xc1\x50\xc0\x95\x7a\xfb\x8a\xb6\x11\x9b\xa8\x5b\x7b\xe6\xff\xbf\x51\x84\x92\x11\x58\x66\x25\x82\x2e\xc0\x58\x06\x3c\xe9\x8a\x2b\x70\x6d\x3b\xc4\xb0\xcc\xb0\x13\xee\x72\x99\xfc\x3c\xc3\xab\x24\x75\x90\x04\xcf\xa4\x9f\x24\x9d\xe1\x3f\x9e\x53\x11\x5c\xef\x81\xf1\xc4\xdf\xcb\xa6\x2e\xcb\x54\xdc\x86\x6b\x17\xb8\xeb\xfd\xbc\xb8\x7e\x84\x4c\xef\x41\x1b\x16\x09\x1c\x35\x1f\x6c\xcd\x40\xf6\xa8\xf3\x7f\x42\xfc\xd1\xdf\x0e\xe1\xad\xbb\xc6\xc9\x2d\xdd\x10\x1e\x29\xb6\xba\xe5\x73\xf4\x0f\x9e\x3f\xdc\x2d\x7a\xc8\x4a\x9b\xa5\xa2\xb0\x84\x7a\x6f\x56\xf2\xee\x37\x28\x01\xc2\x02\x09\x8d\xc2\xed\xda\xed\xd3\x1a\xc8\xb1\x44\x46\x50\xb2\x52\x32\xc7\x54\x28\x6b\x2a\x26\xa9\x0d\x43\x0c\xdd\x7b\xf4\xcd\x0f\xb2\x59\x01\xcd\xe8\xa7\xe6\x6e\xac\x8d\x7e\xa9\x11\xee\x69\xe9\x3a\x90\x88\x44\x7c\x05\x00\x00\xff\xff\xdc\x73\x1e\xbc\x70\x01\x00\x00")

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
