package pubsub_test

import (
	"testing"
	"time"

	"github.com/status-im/status-go/pkg/pubsub"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/stretchr/testify/assert"
)

func TestPubSub_Int(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	testingCh, unsub := pubsub.Subscribe[int](publisher, 10)
	defer unsub()

	val := gofakeit.Int()
	pubsub.Publish(publisher, val)

	incVal, ok := <-testingCh

	assert.True(t, ok)
	assert.Equal(t, val, incVal)
}

func TestPubSub_Ptr(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	testingCh, unsub := pubsub.Subscribe[*string](publisher, 10)
	defer unsub()

	str := gofakeit.Word()
	val := &str
	pubsub.Publish(publisher, val)

	incVal, ok := <-testingCh

	assert.True(t, ok)
	assert.Equal(t, val, incVal)
}

func TestPubSub_Struct(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	type testStruct struct {
		Foo int
		Bar string
	}

	testingCh, unsub := pubsub.Subscribe[testStruct](publisher, 10)
	defer unsub()

	var val testStruct
	err := gofakeit.Struct(&val)
	assert.NoError(t, err)

	pubsub.Publish(publisher, val)

	incVal, ok := <-testingCh

	assert.True(t, ok)
	assert.Equal(t, val, incVal)
}

func TestPubSub_Slice(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	type testSlice []int

	testingCh, unsub := pubsub.Subscribe[testSlice](publisher, 10)
	defer unsub()

	val := make(testSlice, 5)
	gofakeit.Slice(&val)

	pubsub.Publish(publisher, val)

	incVal, ok := <-testingCh

	assert.True(t, ok)
	assert.Equal(t, val, incVal)
}

func TestPubSub_Map(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	type testMap map[string]int

	testingCh, unsub := pubsub.Subscribe[testMap](publisher, 10)
	defer unsub()

	val := make(testMap, 5)
	for range 5 {
		val[gofakeit.Word()] = gofakeit.Int()
	}

	pubsub.Publish(publisher, val)

	incVal, ok := <-testingCh

	assert.True(t, ok)
	assert.Equal(t, val, incVal)
}

func TestPubSub_Chan(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	type testChan chan any

	testingCh, unsub := pubsub.Subscribe[testChan](publisher, 10)
	defer unsub()

	val := make(testChan)

	pubsub.Publish(publisher, val)

	incVal, ok := <-testingCh

	assert.True(t, ok)
	assert.Equal(t, val, incVal)
}

func TestPubSub_Fn(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	type testFunc func() int

	testingCh, unsub := pubsub.Subscribe[testFunc](publisher, 10)
	defer unsub()

	count := 0
	val := func() int {
		count++
		return count
	}

	countVal := val()
	assert.Equal(t, 1, countVal)
	assert.Equal(t, count, countVal)

	pubsub.Publish[testFunc](publisher, val)

	incVal, ok := <-testingCh

	assert.True(t, ok)
	countVal = incVal()
	assert.Equal(t, 2, countVal)
	assert.Equal(t, count, countVal)
}

type testInterface interface {
	Testing()
}

type testImpl struct {
}

func (t testImpl) Testing() {}

func TestPubSub_Intf(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	testingCh, unsub := pubsub.Subscribe[testInterface](publisher, 10)
	defer unsub()

	val := testImpl{}
	pubsub.Publish[testInterface](publisher, val)

	incVal, ok := <-testingCh
	assert.True(t, ok)
	assert.Equal(t, val, incVal)
}

func TestPubSub_Unsub(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	testingCh, unsub := pubsub.Subscribe[int](publisher, 10)

	unsub()

	_, ok := <-testingCh
	assert.False(t, ok)
}

// This test only fails if it panics
func TestPubSub_NoSub(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	assert.NotPanics(t, func() {
		pubsub.Publish(publisher, gofakeit.Int())
	})
}

func TestPubSub_Multi(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	const pubIntCount = 50
	sentInt := make([]int, pubIntCount)
	gofakeit.Slice(&sentInt)
	const pubStringCount = 20
	sentString := make([]string, pubStringCount)
	gofakeit.Slice(&sentString)

	chInt1, unsubInt1 := pubsub.Subscribe[int](publisher, pubIntCount)
	recvInt1 := make([]int, 0, pubIntCount)
	go func() {
		for val := range chInt1 {
			recvInt1 = append(recvInt1, val)
		}
	}()
	defer unsubInt1()

	chInt2, unsubInt2 := pubsub.Subscribe[int](publisher, pubIntCount)
	recvInt2 := make([]int, 0, pubIntCount)
	go func() {
		for val := range chInt2 {
			recvInt2 = append(recvInt2, val)
		}
	}()
	defer unsubInt2()

	chString1, unsubString1 := pubsub.Subscribe[string](publisher, pubStringCount)
	recvString1 := make([]string, 0, pubStringCount)
	go func() {
		for val := range chString1 {
			recvString1 = append(recvString1, val)
		}
	}()
	defer unsubString1()

	for _, val := range sentInt {
		pubsub.Publish(publisher, val)
	}
	for _, val := range sentString {
		pubsub.Publish(publisher, val)
	}

	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		assert.Equal(collect, sentInt, recvInt1)
		assert.Equal(collect, sentInt, recvInt2)
		assert.Equal(collect, sentString, recvString1)
	}, 1*time.Second, 100*time.Millisecond)
}

func TestPubSub_Buffer(t *testing.T) {
	publisher := pubsub.NewPublisher()
	defer publisher.Close()

	const pubIntCount = 10
	sentInt := make([]int, pubIntCount)
	gofakeit.Slice(&sentInt)

	chInt, unsubInt := pubsub.Subscribe[int](publisher, 0) // Unbuffered channel
	recvInt1 := make([]int, 0, pubIntCount)

	// Channel not ready to receive, so the events are dropped.
	for _, val := range sentInt {
		pubsub.Publish(publisher, val)
	}

	quitCh := make(chan struct{})
	defer close(quitCh)

	const sendDelay = 50 * time.Millisecond
	go func() {
		for {
			select {
			case <-quitCh:
				return
			case val := <-chInt:
				recvInt1 = append(recvInt1, val)
				// Keep channel busy
				time.Sleep(2 * sendDelay) // Take twice as long to process as the events are sent, about half events will be dropped.
			}
		}
	}()
	defer unsubInt()

	for _, val := range sentInt {
		pubsub.Publish(publisher, val)
		time.Sleep(sendDelay)
	}

	assert.Subset(t, sentInt, recvInt1)
	assert.Less(t, len(recvInt1), len(sentInt))
}
