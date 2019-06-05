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

var __0001_transfers_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\x51\x6f\x82\x30\x14\x85\xdf\xfb\x2b\xee\xa3\x24\xfd\x07\x7b\x2a\x78\xd5\x66\xac\xdd\x4a\x99\xf3\x89\x20\x76\xd3\xa8\x85\x51\x48\xe6\xbf\x5f\x10\xaa\x73\x33\x64\x8f\xdc\xc3\xbd\xe7\x9c\xaf\x91\x42\xa6\x11\x34\x0b\x63\x04\x3e\x03\x21\x35\xe0\x1b\x4f\x74\x02\x4d\x9d\x5b\xf7\x6e\x6a\x07\x13\xb2\xcd\xdd\x16\x5e\x99\x8a\x16\x4c\x41\x2a\xf8\x4b\x8a\x94\xe4\x9b\x4d\x6d\x9c\xbb\xcc\xbb\x5d\x91\xc6\x31\x25\xeb\xc3\x3e\xbb\x59\xb9\x4a\xcd\x17\x84\xb1\x0c\x29\xa9\x4d\x61\x76\x55\x33\x7c\x35\xa7\xca\xdc\xf9\x7b\x26\x15\xf2\xb9\x80\x47\x5c\x4d\xfc\xd1\x00\x14\xce\x50\xa1\x88\x30\x81\xf5\xa1\x2c\xf6\x6e\xd2\xcf\xa5\x80\x29\xc6\xa8\x11\x22\x96\x44\x6c\x8a\x94\x44\x52\x24\x5a\x31\x2e\x34\xb4\x76\xf7\xd9\x9a\xcc\xd7\xca\x4a\x7b\x3e\x97\xf9\x1a\x7d\x2d\x38\xdf\xa2\xc3\x30\x20\xc1\x03\x21\x23\x90\x7a\xff\xdf\x84\x9e\x15\x7f\x62\x6a\xd5\xc5\xa6\xc4\xb6\xc7\xb5\xa9\x21\xe4\xf3\x2e\xc5\xe0\xe2\x2b\x92\x00\x96\x5c\x2f\x64\xaa\x41\xc9\x25\x9f\x8e\xbb\xe5\x45\x51\xb6\xb6\x71\x59\x53\x66\x17\xe7\xf1\x67\xb8\x75\xbf\x6a\xee\x64\x0b\xe0\x42\xff\x65\xdc\x6f\xdc\xa3\xec\x95\x7f\x71\x3e\xe6\x55\xb5\xb3\x1f\x1d\xe6\x21\x61\x1f\xd9\x27\xf2\xb8\x07\x91\xfe\xb0\xee\xa0\x7f\x07\x00\x00\xff\xff\xd1\xf0\x46\xeb\x99\x02\x00\x00")

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
