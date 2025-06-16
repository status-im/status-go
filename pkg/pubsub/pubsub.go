package pubsub

// Publish will send the value val of type T on the specified Publisher in a
// non-blocking manner. Subscribers for which the channel buffer is full will
// not receive the event.
func Publish[T any](p *Publisher, val T) {
	subs := getTypePublisher[T](p)
	subs.Publish(val)
}

// Subscribe creates a channel with the specified buffer size to listen for events of
// type T published by the Publisher.
// The returned channel will be closed by the Publisher.
// UnsubFn must be called by the subscriber before it becomes unable to process events.
func Subscribe[T any](p *Publisher, bufferSize uint) (chan T, UnsubFn) {
	subs := getTypePublisher[T](p)
	ch := subs.Subscribe(bufferSize)

	unsub := func() {
		subs.Unsubscribe(ch)
	}

	return ch, unsub
}
