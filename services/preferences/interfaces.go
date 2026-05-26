package preferences

//go:generate go tool mockgen -package=mock_preferences -source=interfaces.go -destination=mock/interfaces.go

// PreferenceStore is the persistence seam for per-account opaque preferences.
type PreferenceStore interface {
	Set(category, key, value string) error
	SetMany(category string, kvs map[string]string) error
	Get(category, key string) (value string, found bool, err error)
	GetAll(category string) (map[string]string, error)
	ListCategories() ([]string, error)
	ListKeys(category string) ([]string, error)
	Delete(category, key string) error
	DeleteCategory(category string) (removed int, err error)
	PurgeUnknown(category string, validKeys []string) (removed int, err error)
	LoadAndPurgeUnknown(category string, validKeys []string) (values map[string]string, err error)
}
