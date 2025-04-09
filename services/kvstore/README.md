# Status key value storage

## Overview

The service provides a simple key-value store by leveraging the existing sqlite database. Key value pairs are stored in a table called `kv_store` with the following schema:

```sql
CREATE TABLE IF NOT EXISTS kv_store (
    key TEXT PRIMARY KEY,
    value BLOB
);
```

This solution could be used to store any kind of data, such as application settings, simple states, or any other key-value pairs that need to be persisted.


## Design Considerations

The key is a string, and the value is a byte array (BLOB in SQLite). The users of the key-value store have full control over how the value should be encoded. This allows for flexibility in storing different types of data.

The keys need to be prefixed properly such as `config/rln-rate-limit-enabled`, and maintained in one place which is `database.go` file. The prefix is used to avoid key collisions and to provide a clear structure for the keys. The prefix should be descriptive enough to indicate the purpose of the key.

`DeprecatedKeys` will be removed as part of database migration, also maintained in `database.go` file. The deprecated keys are keys that are no longer used or needed in the application. They are kept in the code for backward compatibility and will be removed in future versions. The duration for which the deprecated keys will be kept in the code is still undecided.


## API Usage

The database API provides a simple interface for setting and getting key-value pairs. The following methods are available:
- `Set(key string, value []byte) error`: Sets the value for the given key. If the key already exists, it will be updated with the new value.
- `Get(key string) ([]byte, error)`: Gets the value for the given key. If the key does not exist, it will return `nil`.
- `Delete(key string) error`: Deletes the key-value pair for the given key.
- `SetBool(key string, value bool) error`: Helper function is save a boolean value.
- `GetBool(key string) (bool, error)`: Helper function to get a boolean value.
