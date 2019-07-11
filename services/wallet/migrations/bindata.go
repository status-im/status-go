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

var __0001_transfers_down_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x29\x4a\xcc\x2b\x4e\x4b\x2d\x2a\xb6\xe6\x42\x12\x4d\xca\xc9\x4f\xce\x46\x15\x4a\x4c\x4e\xce\x2f\xcd\x2b\x29\x8e\x2f\xc9\x8f\x87\x49\x03\x02\x00\x00\xff\xff\xe1\x80\x1c\xac\x48\x00\x00\x00")

func _0001_transfers_down_db_sql() ([]byte, error) {
	return bindata_read(
		__0001_transfers_down_db_sql,
		"0001_transfers.down.db.sql",
	)
}

var __0001_transfers_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\xcd\xae\x9b\x30\x10\x85\xf7\x7e\x8a\x59\x06\x89\x37\xe8\xca\xc0\x90\x58\x75\xed\xd6\x98\xa6\x59\x21\x02\x6e\x82\x12\x0c\xc5\x20\x35\x6f\x5f\x11\x20\x3f\xbd\x51\x74\x77\xd6\xcc\x19\x9f\x6f\xe6\x84\x0a\xa9\x46\xd0\x34\xe0\x08\x2c\x06\x21\x35\xe0\x2f\x96\xe8\x04\xfa\x2e\xb7\xee\xb7\xe9\x1c\xac\xc8\x31\x77\x47\xf8\x49\x55\xb8\xa1\x0a\x52\xc1\x7e\xa4\xe8\x93\xbc\x2c\x3b\xe3\xdc\xad\x3e\xce\x8a\x94\x73\x9f\xec\xcf\xa7\xec\x69\xe4\xde\xea\xff\x42\xc0\x65\xe0\x13\x67\x6c\x69\xba\x17\x8a\xce\x14\xa6\x6a\xfb\x59\x76\x6e\x0e\xf3\xab\xbf\xb4\xe6\x85\x3c\x96\x0a\xd9\x5a\xc0\x57\xdc\xad\x16\x5f\x0f\x14\xc6\xa8\x50\x84\x98\xc0\xfe\xdc\x14\x27\xb7\x9a\xea\x52\x40\x84\x1c\x35\x42\x48\x93\x90\x46\xe8\x93\x50\x8a\x44\x2b\xca\x84\x86\xc1\x56\x7f\x06\x93\x2d\x9b\x67\x8d\xbd\x7e\x97\x2d\x9b\x4e\x9b\xc3\xf5\x2f\x7f\x2e\x7a\xc4\xfb\x42\xc8\x9b\x3b\x4e\xfe\xff\x1f\xf1\xbb\x62\xdf\xa8\xda\x8d\xd8\x3e\xb1\x43\xbd\x37\x1d\x04\x6c\x3d\x52\xcc\x2e\x0f\x37\xab\x6a\xe3\xfa\xbc\x6e\x21\x15\x09\x5b\x0b\x8c\x16\xe9\x5d\x73\x34\x79\x09\x81\x94\x1c\x22\x8c\x69\xca\x35\xc4\x94\x27\x48\x3c\xd8\x32\xbd\x91\xa9\x06\x25\xb7\x2c\x7a\x8f\x9a\x17\x45\x33\xd8\xde\x65\x7d\x93\xdd\xb0\xdf\xc7\xfc\x8c\x7e\xef\xb9\x8b\x2d\x80\x09\xfd\x31\xa0\x69\xe2\x55\x44\x4b\xe7\x53\x21\xd5\x79\xdb\x56\xf6\x30\x66\x34\x13\x4e\xc8\x0b\xd1\x92\xd5\xdc\xf4\x1f\xac\xc7\xc4\xfe\x05\x00\x00\xff\xff\x69\x7c\xa5\x4c\xf9\x02\x00\x00")

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
