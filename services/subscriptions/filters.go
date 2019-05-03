package subscriptions

type filter interface {
	getId() string
	getChanges() ([]interface{}, error)
	uninstall() error
}
