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

var __0001_transfers_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\xcf\x8e\x9b\x30\x10\xc6\xef\x7e\x8a\x39\x06\x89\x37\xe8\xc9\xc0\x90\x58\x75\xed\xd6\x98\xa6\x39\x21\x02\x6e\x83\x12\x0c\xc5\x20\x6d\xde\x7e\x45\x80\xfc\xd9\x8d\xa2\xbd\xce\x7c\x33\xdf\x6f\xe6\x0b\x15\x52\x8d\xa0\x69\xc0\x11\x58\x0c\x42\x6a\xc0\x3f\x2c\xd1\x09\xf4\x5d\x6e\xdd\x5f\xd3\x39\x58\x91\x43\xee\x0e\xf0\x9b\xaa\x70\x43\x15\xa4\x82\xfd\x4a\xd1\x27\x79\x59\x76\xc6\xb9\x6b\x7d\x9c\x15\x29\xe7\x3e\xd9\x9f\x8e\xd9\xc3\xc8\xad\xd5\xbf\x41\xc0\x65\xe0\x13\x67\x6c\x69\xba\x27\x8a\xce\x14\xa6\x6a\xfb\x59\xd6\x9f\x5b\xf3\x44\x14\x4b\x85\x6c\x2d\xe0\x3b\xee\x56\x8b\x9b\x07\x0a\x63\x54\x28\x42\x4c\x60\x7f\x6a\x8a\xa3\x5b\x4d\x75\x29\x20\x42\x8e\x1a\x21\xa4\x49\x48\x23\xf4\x49\x28\x45\xa2\x15\x65\x42\xc3\x60\xab\xff\x83\xc9\x96\x7b\xb3\xc6\x5e\xd6\x65\xcb\x7d\xd3\xbd\x70\xd9\xe5\xcf\x45\x8f\x78\xdf\x08\x79\xf1\xbd\xc9\xff\xe3\xeb\x7e\x2a\xf6\x83\xaa\xdd\x88\xed\x13\x3b\xd4\x7b\xd3\x41\xc0\xd6\x23\xc5\xec\x72\xf7\xa9\xaa\x36\xae\xcf\xeb\x16\x52\x91\xb0\xb5\xc0\x68\x91\xde\x34\x07\x93\x97\x10\x48\xc9\x21\xc2\x98\xa6\x5c\x43\x4c\x79\x82\xc4\x83\x2d\xd3\x1b\x99\x6a\x50\x72\xcb\xa2\xd7\xa8\x79\x51\x34\x83\xed\x5d\xd6\x37\xd9\x15\xfb\x75\xb8\x8f\xe8\xb7\x9e\x3b\xdb\x02\x98\xd0\x9f\x03\x9a\x26\x9e\x45\xb4\x74\xbe\x14\x52\x9d\xb7\x6d\x65\xff\x8d\x19\xcd\x84\x13\xf2\x42\xb4\x64\x35\x37\xfd\x3b\xeb\x31\xb1\xf7\x00\x00\x00\xff\xff\x76\x37\x2b\x31\xef\x02\x00\x00")

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
