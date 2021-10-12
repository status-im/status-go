# SQL Datastore

[![CircleCI](https://circleci.com/gh/ipfs/go-ds-sql.svg?style=shield)](https://circleci.com/gh/ipfs/go-ds-sql)
[![Coverage](https://codecov.io/gh/ipfs/go-ds-sql/branch/master/graph/badge.svg)](https://codecov.io/gh/ipfs/go-ds-sql)
[![Standard README](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg)](https://github.com/RichardLitt/standard-readme)
[![GoDoc](http://img.shields.io/badge/godoc-reference-5272B4.svg)](https://godoc.org/github.com/ipfs/go-ds-sql)
[![golang version](https://img.shields.io/badge/golang-%3E%3D1.14.0-orange.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/ipfs/go-ds-sql)](https://goreportcard.com/report/github.com/ipfs/go-ds-sql)

An implementation of [the datastore interface](https://github.com/ipfs/go-datastore)
that can be backed by any sql database.

## Install

```sh
go get github.com/ipfs/go-ds-sql
```

## Usage

Ensure a database is created and a table exists with `key` and `data` columns. For example, in PostgreSQL you can create a table with the following structure (replacing `table_name` with the name of the table the datastore will use - by default this is `blocks`):

```sql
CREATE TABLE IF NOT EXISTS table_name (key TEXT NOT NULL UNIQUE, data BYTEA)
```

It's recommended to create an index on the `key` column that is optimised for prefix scans. For example, in PostgreSQL you can create a `text_pattern_ops` index on the table:

```sql
CREATE INDEX IF NOT EXISTS table_name_key_text_pattern_ops_idx ON table_name (key text_pattern_ops)
```

Import and use in your application:

```go
import (
	"database/sql"
	"github.com/ipfs/go-ds-sql"
	pg "github.com/ipfs/go-ds-sql/postgres"
)

mydb, _ := sql.Open("yourdb", "yourdbparameters")

// Implement the Queries interface for your SQL impl.
// ...or use the provided PostgreSQL queries
queries := pg.NewQueries("blocks")

ds := sqlds.NewDatastore(mydb, queries)
```

## API

[GoDoc Reference](https://godoc.org/github.com/ipfs/go-ds-sql)

## Contribute

Feel free to dive in! [Open an issue](https://github.com/ipfs/go-ds-sql/issues/new) or submit PRs.

## License

[MIT](LICENSE)
