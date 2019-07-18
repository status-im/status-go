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

var __0001_accounts_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x8e\x4d\x4e\xf3\x30\x10\x40\xf7\x3e\xc5\x2c\x1b\x29\x37\xf8\x56\x8e\x33\xf9\x6a\x11\x6c\x98\x4c\x68\xbb\x8a\x4c\x6a\x50\x24\x70\x8a\x1d\x83\xb8\x3d\x0a\x50\x75\x53\xb1\x9d\x9f\xf7\x9e\x22\x94\x8c\xc0\xb2\x6a\x11\x74\x03\xc6\x32\xe0\x5e\x77\xdc\x81\x1b\xc7\x39\x87\x25\xc1\x46\xb8\xe3\x31\xfa\x94\xe0\x41\x92\xda\x4a\x82\x3b\xd2\xb7\x92\x0e\x70\x83\x87\x52\x04\xf7\xea\x81\x71\xcf\xdf\xcf\xa6\x6f\xdb\x52\x14\xb0\xd3\xbc\xb5\x3d\x03\xd9\x9d\xae\xff\x09\xf1\x87\x68\x9c\xc3\xd3\xf4\x9c\xa3\x5b\xa6\x39\x5c\xd3\x5d\xb8\xcb\xe7\xc9\x5f\x19\xbf\xbb\x97\xec\xa1\x6a\x6d\x55\x8a\xc6\x12\xea\xff\x66\x6d\xdb\xfc\x82\x0a\x20\x6c\x90\xd0\x28\xec\xe0\x31\xce\x1f\xc9\xc7\x74\x59\x5a\x03\x35\xb6\xc8\x08\x4a\x76\x4a\xd6\x58\x0a\x65\x4d\xc7\x24\xb5\x61\xc8\x61\x7a\xcb\x7e\xf8\x89\x1c\xd6\x80\xe1\xe4\xe3\x70\x6e\xec\x8d\xbe\xef\x11\xce\xb4\x72\x3d\x28\x44\x21\xbe\x02\x00\x00\xff\xff\x99\x43\xfd\x7a\x59\x01\x00\x00")

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
