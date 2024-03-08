// This file is necessary because "github.com/status-im/migrate/v4"
// can't handle files starting with a prefix. At least that's the case
// for go-bindata.
// If go-bindata is called from the same directory, asset names
// have no prefix and "github.com/status-im/migrate/v4" works as expected.

// NOTE: The naming of the file needs to follow the unix timestamp convention
// unlike the two initial files.
// That's because currently the encryption migrations and protocol migrations
// share the same table, which means that the latest migration in terms of filename
// is from the encryption namespace files, which uses a unix timestamp.
// If we add 00003_... for example, this will be ignored in existing clients as
// older than the latest encryption migration.

package sqlite

//go:generate go-bindata -modtime=1700000000 -pkg migrations -o ../migrations.go .
