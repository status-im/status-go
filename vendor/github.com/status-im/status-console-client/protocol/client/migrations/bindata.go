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

var __000001_add_messages_contacts_down_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x09\xf2\x0f\x50\x08\x71\x74\xf2\x71\x55\x28\x2d\x4e\x2d\x8a\xcf\x4d\x2d\x2e\x4e\x4c\x4f\x2d\xb6\xe6\x42\x97\x49\xce\xcf\x2b\x49\x4c\x2e\x41\x95\xc9\xc8\x2c\x2e\xc9\x2f\xaa\x8c\x47\x56\x11\x5f\x92\x5f\x90\x99\x6c\xcd\x05\x08\x00\x00\xff\xff\x1b\x57\x14\x62\x5b\x00\x00\x00")

func _000001_add_messages_contacts_down_db_sql() ([]byte, error) {
	return bindata_read(
		__000001_add_messages_contacts_down_db_sql,
		"000001_add_messages_contacts.down.db.sql",
	)
}

var __000001_add_messages_contacts_up_db_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x92\xcf\x6e\xb3\x30\x10\xc4\xef\x7e\x8a\x3d\xf2\x49\x1c\xbe\x7b\x4f\x06\x96\xc4\x2a\xb5\x5b\xc7\x34\xc9\x09\x51\xc7\x6a\x50\xc2\x1f\xc5\x8e\x54\xde\xbe\x82\x42\xe3\xa6\x69\xae\x33\xf6\x7a\xe6\xb7\x8e\x25\x52\x85\xa0\x68\x94\x21\xb0\x14\xb8\x50\x80\x1b\xb6\x52\x2b\x38\x5b\x73\x2a\x6a\x63\x6d\xf9\x6e\x2c\x04\xa4\xda\x41\x94\x89\x08\x72\xce\x5e\x72\x1c\x4f\xf2\x3c\xcb\x42\xa2\xdb\xc6\x95\xda\x15\xd5\x0e\x5e\xa9\x8c\x97\x54\x5e\x99\xa6\x71\x85\xeb\x3b\x33\xdb\x21\x99\xc6\x5e\xa9\xce\x7c\x38\x50\xb8\x51\x21\xd1\xc7\x56\x1f\x20\x62\x0b\xc6\x55\x48\x5c\x55\x1b\xeb\xca\xba\xfb\x56\xe6\xb1\x7a\x5f\x8e\x0f\x4f\xb7\xe6\xc7\x2e\x83\xba\xf3\xdb\xb1\xd2\xc5\xc1\xf4\x63\x7a\xf2\xef\x81\x90\xa9\x34\xe3\x09\x6e\xe0\x92\xde\x82\xe0\x3f\x5b\x07\x17\xd3\xbb\xf7\x27\xac\xe9\xf4\x04\x6b\x66\xf1\x2c\xd9\x13\x95\x5b\x78\xc4\xad\xc7\xa5\x29\x6b\x73\x03\x97\x6b\xbb\x4a\x8f\xd1\x7d\x71\xa0\xc4\xb8\x2f\x59\x57\xba\x51\xbb\xd1\x10\xd6\x4c\x2d\x45\xae\x40\x8a\x35\x4b\xee\xe7\xde\x57\xd6\xb5\xa7\xbe\xf0\xf3\x17\x5f\x21\x02\x62\xfb\x46\x9b\xdd\xc4\x1c\x12\x4c\x69\x9e\x29\xf8\x7f\x7f\xf5\xbf\xbe\x47\x2a\x24\xb2\x05\x1f\xfa\xfb\x3c\x41\x62\x8a\x12\x79\x8c\x57\xf4\x82\xc1\x14\x1c\x12\xcc\x50\x21\xc4\x74\x15\xd3\x04\x87\xc5\x7d\x06\x00\x00\xff\xff\x28\x6d\x42\x61\xad\x02\x00\x00")

func _000001_add_messages_contacts_up_db_sql() ([]byte, error) {
	return bindata_read(
		__000001_add_messages_contacts_up_db_sql,
		"000001_add_messages_contacts.up.db.sql",
	)
}

var __000002_add_unread_to_user_messages_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xd2\xd5\x55\x48\x4e\xcc\xcb\xcb\x2f\x51\x48\x29\xca\x2f\x50\x48\xce\xcf\x29\xcd\xcd\x53\x28\xcf\x2c\xc9\xc8\x2f\x2d\x51\x48\xce\x2f\xa8\xcc\xcc\x4b\x57\x48\x54\x28\x49\x4c\xca\x49\xe5\x02\x04\x00\x00\xff\xff\xf3\xa7\xbd\xe9\x2e\x00\x00\x00")

func _000002_add_unread_to_user_messages_down_sql() ([]byte, error) {
	return bindata_read(
		__000002_add_unread_to_user_messages_down_sql,
		"000002_add_unread_to_user_messages.down.sql",
	)
}

var __000002_add_unread_to_user_messages_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x2c\xcc\x41\x0a\xc2\x40\x0c\x46\xe1\x7d\x4f\xf1\x5f\xa0\xe8\xde\xd5\xe8\x44\x11\x62\x0a\x25\xb3\x96\x80\xb1\x14\xab\xc2\x84\xf1\xfc\x52\xe8\xea\xad\xde\x97\x58\x69\x84\xa6\x23\x13\x5a\x78\xbd\xbf\x3d\xc2\x26\x8f\x0e\x00\x52\xce\x38\x0d\x5c\x6e\x82\xe7\x62\x53\xe0\x2a\x4a\x17\x1a\x21\x83\x42\x0a\x33\x32\x9d\x53\x61\xc5\xfe\x80\xbe\xc7\xcf\xea\xfc\x6d\x81\x0d\xd9\xa6\x65\x7e\x39\xaa\xdb\x63\xd7\x3e\x6b\xba\x7f\x00\x00\x00\xff\xff\xe2\xba\xfc\x8b\x75\x00\x00\x00")

func _000002_add_unread_to_user_messages_up_sql() ([]byte, error) {
	return bindata_read(
		__000002_add_unread_to_user_messages_up_sql,
		"000002_add_unread_to_user_messages.up.sql",
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
	"000001_add_messages_contacts.down.db.sql": _000001_add_messages_contacts_down_db_sql,
	"000001_add_messages_contacts.up.db.sql": _000001_add_messages_contacts_up_db_sql,
	"000002_add_unread_to_user_messages.down.sql": _000002_add_unread_to_user_messages_down_sql,
	"000002_add_unread_to_user_messages.up.sql": _000002_add_unread_to_user_messages_up_sql,
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
	"000001_add_messages_contacts.down.db.sql": &_bintree_t{_000001_add_messages_contacts_down_db_sql, map[string]*_bintree_t{
	}},
	"000001_add_messages_contacts.up.db.sql": &_bintree_t{_000001_add_messages_contacts_up_db_sql, map[string]*_bintree_t{
	}},
	"000002_add_unread_to_user_messages.down.sql": &_bintree_t{_000002_add_unread_to_user_messages_down_sql, map[string]*_bintree_t{
	}},
	"000002_add_unread_to_user_messages.up.sql": &_bintree_t{_000002_add_unread_to_user_messages_up_sql, map[string]*_bintree_t{
	}},
}}
