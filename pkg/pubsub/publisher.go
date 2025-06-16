package pubsub

import (
	"reflect"
	"sync"
)

// Publisher manages subscriptions to events of different types.
type Publisher struct {
	subsPerType *sync.Map // map[reflect.Type]*TypePublisher[T]
}

func NewPublisher() *Publisher {
	return &Publisher{
		subsPerType: &sync.Map{},
	}
}

type closeable interface {
	Close()
}

func (p *Publisher) Close() {
	p.subsPerType.Range(func(key any, value any) bool {
		value.(closeable).Close()
		return true
	})
	p.subsPerType = &sync.Map{}
}

func getTypePublisher[T any](p *Publisher) *TypePublisher[T] {
	typeOfT := reflect.TypeFor[T]()
	subs, _ := p.subsPerType.LoadOrStore(typeOfT, NewTypePublisher[T]())
	return subs.(*TypePublisher[T])
}
