package history

import "sync"

type Emitter[T any] struct {
	sync.Mutex
	subscriptions []chan T
}

func NewEmitter[T any]() *Emitter[T] {
	return &Emitter[T]{}
}

func (s *Emitter[T]) Subscribe() <-chan T {
	s.Lock()
	defer s.Unlock()
	c := make(chan T)
	s.subscriptions = append(s.subscriptions, c)
	return c
}

func (s *Emitter[T]) Emit(value T) {
	s.Lock()
	defer s.Unlock()

	for _, sub := range s.subscriptions {
		sub <- value
	}
}

type OneShotEmitter[T any] struct {
	Emitter[T]
}

func NewOneshotEmitter[T any]() *OneShotEmitter[T] {
	return &OneShotEmitter[T]{}
}

func (s *OneShotEmitter[T]) Emit(value T) {
	s.Lock()
	defer s.Unlock()

	for _, subs := range s.subscriptions {
		subs <- value
		close(subs)
	}
	s.subscriptions = nil
}
