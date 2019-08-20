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

var __0001_accounts_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x1c\xcb\x41\x0e\x82\x30\x10\x05\xd0\x7d\x4f\xf1\x97\x9a\x70\x03\x57\x05\xab\x4c\x44\x20\xc3\x20\xb0\x6c\x80\x08\x89\xb4\xc4\xd6\xfb\x9b\x70\x80\x97\xb1\xd1\x62\x20\x3a\x2d\x0c\xe8\x86\xb2\x12\x98\x9e\x1a\x69\x60\xc7\xd1\xff\x5c\x0c\x38\x29\x3b\x4d\xdf\x39\x04\xbc\x34\x67\xb9\x66\xd4\x4c\x4f\xcd\x03\x1e\x66\x48\x94\xb3\xdb\x0c\x31\xbd\x1c\xb8\x6c\x8b\x22\x51\x1f\xff\x5e\x9d\xac\xdb\x1c\xa2\xdd\x76\xa4\x74\x07\x95\x92\xa8\x7d\xf1\xd1\xd7\x36\x2e\x07\x50\x67\x74\x24\x79\xd5\x0a\xb8\xea\xe8\x7a\x51\xff\x00\x00\x00\xff\xff\x14\xa8\x25\xe1\x8f\x00\x00\x00")

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
