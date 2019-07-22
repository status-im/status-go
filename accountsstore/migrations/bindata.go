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

var __0001_accounts_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x8f\xcd\x4e\xeb\x30\x10\x85\xf7\xf3\x14\xb3\x6c\xa4\xbc\xc1\x5d\x39\xce\xe4\xd6\x22\xd8\x30\x99\xd0\x76\x15\x59\xa9\x41\x91\xc0\x29\x49\x8c\xc4\xdb\xa3\x00\x51\x37\x15\xdb\xf9\xf9\xce\x77\x34\x93\x12\x42\x51\x45\x4d\x68\x2a\xb4\x4e\x90\x8e\xa6\x91\x06\x7d\xdf\x8f\x29\x2e\x33\xee\xc0\x9f\xcf\x53\x98\x67\x7c\x52\xac\xf7\x8a\xf1\x81\xcd\xbd\xe2\x13\xde\xd1\x29\x87\xe8\xdf\x02\x0a\x1d\xe5\xfb\xd9\xb6\x75\x0d\x19\x1e\x8c\xec\x5d\x2b\xc8\xee\x60\xca\x7f\x00\x7f\xe4\xf4\x63\x7c\x1e\x5e\xd2\xe4\x97\x61\x8c\xb7\xd2\x36\x6c\x0e\xcb\xe7\x25\xdc\x18\x7f\xf8\xd7\x14\xb0\xa8\x5d\x91\x43\xe5\x98\xcc\x7f\xbb\xaa\xed\x7e\x41\x19\x32\x55\xc4\x64\x35\x5d\x5b\x5d\x97\xce\x62\x49\x35\x09\xa1\x56\x8d\x56\x25\xe5\xa0\x9d\x6d\x84\x95\xb1\x82\x29\x0e\xef\x29\x74\x3f\x92\xdd\x2a\xd0\x5d\xc2\xd4\x6d\x8e\xad\x35\x8f\x2d\xe1\x46\xcb\xd7\x83\x0c\x32\xf8\x0a\x00\x00\xff\xff\x5b\x93\x16\x33\x58\x01\x00\x00")

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
