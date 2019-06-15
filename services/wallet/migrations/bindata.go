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

var __0001_transfers_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x91\xd1\x72\xaa\x30\x10\x86\xef\xf3\x14\x7b\x29\x33\x79\x83\x73\x15\x60\xd1\xcc\xc9\x49\x4e\x43\xa8\xf5\x8a\x41\x4c\xab\xa3\x06\x4a\x60\xa6\xbe\x7d\x07\x01\xad\xad\xe3\xf4\x32\xbb\xd9\xdd\xef\xff\xff\x48\x23\x33\x08\x86\x85\x02\x81\x27\x20\x95\x01\x7c\xe1\xa9\x49\xa1\x6d\x0a\xe7\x5f\x6d\xe3\x61\x46\xb6\x85\xdf\xc2\x33\xd3\xd1\x82\x69\xc8\x24\x7f\xca\x90\x92\x62\xb3\x69\xac\xf7\x97\x7a\x3f\x2b\x33\x21\x28\x59\x1f\xf6\xf9\xcd\xc8\xb5\xd5\x7e\x40\x28\x54\x48\x49\x63\x4b\xbb\xab\xdb\xf1\xd5\x9e\x6a\x7b\xe7\x77\xa2\x34\xf2\xb9\x84\xbf\xb8\x9a\x4d\x4b\x03\xd0\x98\xa0\x46\x19\x61\x0a\xeb\x43\x55\xee\xfd\x6c\xa8\x2b\x09\x31\x0a\x34\x08\x11\x4b\x23\x16\x23\x25\x91\x92\xa9\xd1\x8c\x4b\x03\x9d\xdb\xbd\x77\x36\x9f\x64\xe5\x95\x3b\xaf\xcb\x27\x19\x83\x2c\x38\xef\xa2\x63\x31\x20\xc1\x1f\x42\x1e\x98\x34\xdc\xff\xee\xd0\x7f\xcd\xff\x31\xbd\xea\xb1\x29\x71\xdd\x71\x6d\x1b\x08\xf9\xbc\xa7\x18\xaf\x5c\x25\x6e\x6d\xb1\x81\x50\x29\x01\x31\x26\x2c\x13\x06\x12\x26\x52\x24\x01\x2c\xb9\x59\xa8\xcc\x80\x56\x4b\x1e\x3f\xc6\x28\xca\xb2\xea\x5c\xeb\xf3\xb6\xca\x2f\x48\x8f\xf3\xb9\xc5\xba\xf6\xfc\xc9\x95\xc0\xa5\xf9\x69\xfe\x30\x71\xcf\xfe\xa9\xf3\xab\x00\x8e\x45\x5d\xef\xdc\x5b\xef\xff\x48\x38\x20\x4f\x44\x53\x0e\x63\x93\x7e\x39\xdd\xa7\xf1\x19\x00\x00\xff\xff\xfc\x91\xad\x60\xb2\x02\x00\x00")

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
