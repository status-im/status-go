package protocol

import (
	"context"
)

func (m *Messenger) AddBrowser(ctx context.Context, browser Browser) error {
	return m.persistence.AddBrowser(browser)
}

func (m *Messenger) GetBrowsers(ctx context.Context) (browsers []*Browser, err error) {
	return m.persistence.GetBrowsers()
}

func (m *Messenger) DeleteBrowser(ctx context.Context, id string) error {
	return m.persistence.DeleteBrowser(id)
}
