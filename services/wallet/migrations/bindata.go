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

var __0001_transfers_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x8f\x4d\x4e\xc3\x30\x10\x85\xf7\x3e\xc5\x5b\x26\x52\x6e\xd0\x95\xed\x4c\xda\x11\xc6\x86\x89\x43\xe9\x0a\x25\xc5\x28\xa8\xa5\x54\x49\x90\xe0\xf6\xa8\x04\x84\xf8\x51\x97\x33\xef\x9b\x79\xfa\xac\x90\x8e\x84\xa8\x8d\x23\x70\x05\x1f\x22\xe8\x96\xeb\x58\x63\x1a\xda\xc3\xf8\x90\x86\x11\x99\xea\xdb\xb1\xc7\x8d\x16\xbb\xd2\x82\xc6\xf3\x75\x43\x1f\xa8\x6f\x9c\x2b\x54\xb7\xdf\xdd\xfd\x20\xbe\xa3\xe9\x15\xc6\x05\x53\xa8\x21\x6d\xd3\xe3\x71\xfa\x9c\xa6\xb7\x63\xfa\x87\xae\x82\x10\x2f\x3d\x2e\x68\x93\x7d\x3d\xcd\x21\x54\x91\x90\xb7\x54\xa3\xdb\x3f\x6f\x77\x63\x36\xef\x83\x47\x49\x8e\x22\xc1\xea\xda\xea\x92\x54\xbe\x50\xea\x8c\xd1\x7c\xfd\x5b\xe7\x4a\xf8\x52\xcb\xe6\x54\x5a\xa8\xc3\xcb\x53\x97\x06\x18\x5e\xb2\x8f\x7f\x4d\xfb\xd4\xde\x9f\x62\x17\x8c\xca\xb1\xe6\xb8\x0a\x4d\x84\x84\x35\x97\x0b\xf5\x1e\x00\x00\xff\xff\x59\x89\x24\x2f\x4c\x01\x00\x00")

func _0001_transfers_up_db_sql() ([]byte, error) {
	return bindata_read(
		__0001_transfers_up_db_sql,
		"0001_transfers.up.db.sql",
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
}}
