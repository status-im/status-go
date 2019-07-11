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

var __0001_transfers_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\xcd\x8e\x9b\x30\x14\x46\xf7\x7e\x8a\xbb\x0c\x92\xdf\xa0\x2b\x03\x97\x8c\x55\xd7\x6e\x8d\xe9\x74\x56\x88\x80\xdb\xa0\x19\x0c\xc5\x20\x4d\xde\xbe\x22\x40\x48\xa2\x28\xea\xd6\xf7\xef\xf8\x3b\x91\x46\x66\x10\x0c\x0b\x05\x02\x4f\x40\x2a\x03\xf8\x8b\xa7\x26\x85\xa1\x2f\x9c\xff\x6d\x7b\x0f\x3b\x72\x2c\xfc\x11\x7e\x32\x1d\xbd\x30\x0d\x99\xe4\x3f\x32\xa4\xa4\xa8\xaa\xde\x7a\x7f\x79\x9f\x66\x65\x26\x04\x25\x87\x8f\xf7\xfc\x66\x64\x2b\x0d\x9f\x10\x0a\x15\x52\xe2\xad\xab\x6c\xff\xa0\xa3\xb7\xa5\xad\xbb\x61\x69\x5b\x29\xb8\xab\xec\x27\x64\x32\xe5\x7b\x89\x31\x70\x69\x28\x19\x4e\x9d\x7d\xb0\x20\x51\x1a\xf9\x5e\xc2\x57\x7c\xdb\xad\x24\x01\x68\x4c\x50\xa3\x8c\x30\x85\xc3\x47\x5b\xbe\xfb\xdd\xfc\xae\x24\xc4\x28\xd0\x20\x44\x2c\x8d\x58\x8c\x94\x44\x4a\xa6\x46\x33\x2e\x0d\x8c\xae\xfe\x3b\xda\x7c\xa5\xc8\x5b\x77\x5e\x97\xaf\x7f\x9f\xb3\x80\xf3\x2e\xba\x3c\x06\x24\xf8\x42\xc8\x93\x64\xe7\xfb\xf7\xb1\x7e\xd7\xfc\x1b\xd3\x6f\x13\x36\x25\x6e\x6c\x0e\xb6\x87\x90\xef\x27\x8a\xe5\xca\x55\x8a\x75\x63\xfd\x50\x34\xdd\x96\xc8\xd2\xba\xf5\x1c\x6d\x51\x41\xa8\x94\x80\x18\x13\x96\x09\x03\x09\x13\x29\x92\x00\x5e\xb9\x79\x51\x99\x01\xad\x5e\x79\xfc\x1c\xb5\x28\xcb\x76\x74\x83\xcf\x87\x36\xbf\x60\x3f\x17\x7f\x8b\xbe\xd5\xfc\xc9\x95\xb3\xb7\x7b\x41\xf3\xc4\x23\x45\x6b\xe5\xbf\x24\x35\x45\xd7\xd5\xee\xcf\xe4\x68\x21\x9c\x91\x57\xa2\xd5\xd5\x52\xa4\x57\xa7\x27\x63\xff\x02\x00\x00\xff\xff\x97\x7c\x79\x34\x0b\x03\x00\x00")

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
