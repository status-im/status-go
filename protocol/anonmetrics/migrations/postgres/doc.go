// This file is necessary because "github.com/status-im/migrate/v4"
// can't handle files starting with a prefix. At least that's the case
// for go-bindata.
// If go-bindata is called from the same directory, asset names
// have no prefix and "github.com/status-im/migrate/v4" works as expected.

package migrations

//go:generate go-bindata -pkg migrations -o ../migrations.go ./
