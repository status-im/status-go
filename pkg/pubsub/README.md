# pubsub

A type-safe, generic publish-subscribe pattern implementation for Go.

## Overview

The `pubsub` package provides a thread-safe, type-safe implementation of the publish-subscribe pattern using Go generics. It allows different parts of your application to communicate through typed events without direct coupling.

## Key Features

- **Type Safety**: Uses Go generics to ensure type safety at compile time
- **Thread Safe**: All operations are protected by appropriate mutexes
- **Non-blocking**: Publishers never block, events are dropped if subscribers can't keep up
- **Automatic Cleanup**: Channels are automatically closed when unsubscribing
- **Multiple Event Types**: Single publisher can handle multiple different event types
- **Buffer Control**: Configurable buffer sizes for subscriber channels

## API Reference

### Core Functions

#### `NewPublisher() *Publisher`
Creates a new publisher instance that can handle multiple event types.

#### `Publish[T any](p *Publisher, val T)`
Publishes a value of type `T` to all subscribers of that type. This operation is non-blocking.

#### `Subscribe[T any](p *Publisher, bufferSize uint) (chan T, UnsubFn)`
Subscribes to events of type `T` and returns a channel to receive events and an unsubscribe function.

### Publisher Methods

#### `Close()`
Closes the publisher and all its subscriptions, cleaning up resources.

## Usage Examples

### Basic Usage

```go
package main

import (
    "fmt"
    "time"
    "github.com/status-im/status-go/pkg/pubsub"
)

func main() {
    // Create a publisher
    publisher := pubsub.NewPublisher()
    defer publisher.Close()

    // Define event types
    type UserEvent struct {
        UserID   string
        Action   string
        Timestamp time.Time
    }

    type SystemEvent struct {
        Level   string
        Message string
    }

    // Subscribe to UserEvent
    userCh, unsubUser := pubsub.Subscribe[UserEvent](publisher, 10)
    defer unsubUser()

    // Subscribe to SystemEvent
    systemCh, unsubSystem := pubsub.Subscribe[SystemEvent](publisher, 5)
    defer unsubSystem()

    // Start listening for events
    go func() {
        for event := range userCh {
            fmt.Printf("User event: %+v\n", event)
        }
    }()

    go func() {
        for event := range systemCh {
            fmt.Printf("System event: %+v\n", event)
        }
    }()

    // Publish events
    pubsub.Publish(publisher, UserEvent{
        UserID:    "user123",
        Action:    "login",
        Timestamp: time.Now(),
    })

    pubsub.Publish(publisher, SystemEvent{
        Level:   "info",
        Message: "Server started",
    })
}
```

### Working with Different Types

```go
// Integer events
intCh, unsubInt := pubsub.Subscribe[int](publisher, 100)
defer unsubInt()

// String events
stringCh, unsubString := pubsub.Subscribe[string](publisher, 50)
defer unsubString()

// Custom struct events
type CustomEvent struct {
    ID   int
    Data string
}

customCh, unsubCustom := pubsub.Subscribe[CustomEvent](publisher, 25)
defer unsubCustom()

// Publish different types
pubsub.Publish(publisher, 42)
pubsub.Publish(publisher, "hello world")
pubsub.Publish(publisher, CustomEvent{ID: 1, Data: "test"})
```

### Multiple Subscribers

```go
// Multiple subscribers for the same event type
ch1, unsub1 := pubsub.Subscribe[UserEvent](publisher, 10)
ch2, unsub2 := pubsub.Subscribe[UserEvent](publisher, 10)
ch3, unsub3 := pubsub.Subscribe[UserEvent](publisher, 10)

defer unsub1()
defer unsub2()
defer unsub3()

// All three subscribers will receive the same event
pubsub.Publish(publisher, UserEvent{UserID: "user1", Action: "login"})
```

### Buffer Management

```go
// Unbuffered channel (bufferSize = 0)
// Events will be dropped if subscriber can't keep up
ch, unsub := pubsub.Subscribe[UserEvent](publisher, 0)
defer unsub()

// Large buffer for high-throughput scenarios
highThroughputCh, unsubHigh := pubsub.Subscribe[UserEvent](publisher, 1000)
defer unsubHigh()
```

## Best Practices

### 1. Always Clean Up
Always call the unsubscribe function returned by `Subscribe` to prevent memory leaks:

```go
ch, unsub := pubsub.Subscribe[MyEvent](publisher, 10)
defer unsub() // Important!
```

### 2. Choose Appropriate Buffer Sizes
- Use smaller buffers for low-frequency events or events that are processed quickly
- Use larger buffers for high-frequency events or events that take longer to process
- Use unbuffered channels (0) when you want to drop events if subscribers can't keep up
- For events that require heavy processing, consider doing the work in a separate goroutine to release the channel quickly:

```go
ch, unsub := pubsub.Subscribe[HeavyEvent](publisher, 10)
defer unsub()

for event := range ch {
    // Process the event in a separate goroutine to avoid blocking the channel
    go func(e HeavyEvent) {
        // Do heavy processing here
        processHeavyEvent(e)
    }(event)
}
```

### 3. Handle Channel Closure
Always check if channels are closed in your event processing loops:

```go
for event := range ch {
    // Process event
}
// Channel is closed when loop exits
```

### 4. Publisher Lifecycle
Always close the publisher when you're done with it:

```go
publisher := pubsub.NewPublisher()
defer publisher.Close()
```

## Thread Safety

All operations in the pubsub package are thread-safe:
- Multiple goroutines can publish events simultaneously
- Multiple goroutines can subscribe/unsubscribe simultaneously
- Event delivery is safe across concurrent operations

## Error Handling

The pubsub package is designed to be robust:
- Publishing to a closed publisher is safe (events are ignored)
- Subscribing to a closed publisher is safe (returns closed channel)
- Unsubscribing multiple times is safe (idempotent)
- Events are dropped if subscriber channels are full (non-blocking behavior)

## Examples in Tests

For more comprehensive examples, see the test file `pubsub_test.go` which demonstrates:
- Working with different data types (int, string, struct, slice, map, channel, function, interface)
- Multiple subscribers
- Unsubscription behavior
- Concurrent operations