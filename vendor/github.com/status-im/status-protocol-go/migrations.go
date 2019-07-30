package statusproto

import (
	"strings"

	"github.com/pkg/errors"

	encryptmigrations "github.com/status-im/status-protocol-go/encryption/migrations"
	appmigrations "github.com/status-im/status-protocol-go/migrations"
	transpmigrations "github.com/status-im/status-protocol-go/transport/whisper/migrations"
)

type getter func(string) ([]byte, error)

type migrationsWithGetter struct {
	Names  []string
	Getter getter
}

var defaultMigrations = []migrationsWithGetter{
	{
		Names:  transpmigrations.AssetNames(),
		Getter: transpmigrations.Asset,
	},
	{
		Names:  encryptmigrations.AssetNames(),
		Getter: encryptmigrations.Asset,
	},
	{
		Names:  appmigrations.AssetNames(),
		Getter: appmigrations.Asset,
	},
}

func prepareMigrations(migrations []migrationsWithGetter) ([]string, getter, error) {
	var allNames []string
	nameToGetter := make(map[string]getter)

	for _, m := range migrations {
		for _, name := range m.Names {
			if !validateName(name) {
				continue
			}

			if _, ok := nameToGetter[name]; ok {
				return nil, nil, errors.Errorf("migration with name %s already exists", name)
			}
			allNames = append(allNames, name)
			nameToGetter[name] = m.Getter
		}
	}

	return allNames, func(name string) ([]byte, error) {
		getter, ok := nameToGetter[name]
		if !ok {
			return nil, errors.Errorf("no migration for name %s", name)
		}
		return getter(name)
	}, nil
}

// validateName verifies that only *.sql files are taken into consideration.
func validateName(name string) bool {
	return strings.HasSuffix(name, ".sql")
}
