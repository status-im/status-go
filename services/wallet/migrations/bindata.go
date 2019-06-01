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

var __0001_transfers_down_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x29\x4a\xcc\x2b\x4e\x4b\x2d\x2a\xb6\xe6\x42\x12\x4d\xca\xc9\x4f\xce\x2e\xb6\xe6\x02\x04\x00\x00\xff\xff\x27\x4d\x7a\xa1\x29\x00\x00\x00")

func _0001_transfers_down_db_sql() ([]byte, error) {
	return bindata_read(
		__0001_transfers_down_db_sql,
		"0001_transfers.down.db.sql",
	)
}

var __0001_transfers_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x8f\x4d\x4e\xc3\x30\x10\x85\xf7\x3e\xc5\x5b\x36\x52\x6e\xd0\x95\xe3\x4c\xda\x11\xc6\x06\x67\x42\xe9\x0a\x25\xc1\x28\xa8\x25\x54\x71\x90\xe8\xed\x51\x5b\x90\x10\x02\x96\x33\xef\x9b\x9f\xcf\x04\xd2\x42\x10\x5d\x58\x02\x57\x70\x5e\x40\xf7\x5c\x4b\x8d\x79\x6a\xc7\xf4\x14\xa7\x84\x85\x1a\xda\x34\xe0\x4e\x07\xb3\xd6\xe1\xcc\xb8\xc6\xda\x5c\x75\xfb\xdd\xc3\x1f\xd1\xfc\x8e\xc2\xfa\x22\x57\x53\xec\xe3\xf3\x61\xfe\xac\xe6\xe3\x21\xfe\x42\x57\x3e\x10\xaf\x1c\xae\x68\xbb\xf8\x5a\x9a\x21\x50\x45\x81\x9c\xa1\x1a\xdd\xfe\xb5\xdf\xa5\xc5\xa5\xef\x1d\x4a\xb2\x24\x04\xa3\x6b\xa3\x4b\x52\xd9\x52\xa9\x7f\x54\x2e\xd3\x3f\x3d\x6e\x02\x5f\xeb\xb0\x3d\x1d\xcd\xd5\xf8\xf6\xd2\xc5\x09\x05\xaf\xd8\x09\x1a\xc7\xb7\x0d\x7d\x7b\x70\x88\xed\xe3\x29\x3e\x4b\xa4\xe3\xd8\x83\x9d\xa8\x0c\x1b\x96\xb5\x6f\x04\xc1\x6f\xb8\x5c\xaa\x8f\x00\x00\x00\xff\xff\x99\xd3\x62\x18\x4f\x01\x00\x00")

func _0001_transfers_up_db_sql() ([]byte, error) {
	return bindata_read(
		__0001_transfers_up_db_sql,
		"0001_transfers.up.db.sql",
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
	"0001_transfers.down.db.sql": _0001_transfers_down_db_sql,
	"0001_transfers.up.db.sql": _0001_transfers_up_db_sql,
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
	"0001_transfers.down.db.sql": &_bintree_t{_0001_transfers_down_db_sql, map[string]*_bintree_t{
	}},
	"0001_transfers.up.db.sql": &_bintree_t{_0001_transfers_up_db_sql, map[string]*_bintree_t{
	}},
	"doc.go": &_bintree_t{doc_go, map[string]*_bintree_t{
	}},
}}
