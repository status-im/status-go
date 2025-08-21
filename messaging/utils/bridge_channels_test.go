package utils

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBridgeChannels(t *testing.T) {
	intChan := make(chan int, 3)
	intChan <- 1
	intChan <- 2
	intChan <- 3
	close(intChan)

	convert := func(i int) string {
		return fmt.Sprintf("%d", i)
	}

	resultChan := BridgeChannels(intChan, convert)

	expected := []string{"1", "2", "3"}
	var result []string
	for v := range resultChan {
		result = append(result, v)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestBridgeChannelsSlice(t *testing.T) {
	intChan := make(chan []int, 3)
	intChan <- []int{1, 2, 3}
	intChan <- []int{4, 5, 6}
	intChan <- []int{7, 8, 9}
	close(intChan)

	convert := func(i int) string {
		return fmt.Sprintf("%d", i)
	}

	resultChan := BridgeChannelsSlice(intChan, convert)

	expected := [][]string{
		{"1", "2", "3"},
		{"4", "5", "6"},
		{"7", "8", "9"},
	}
	var result [][]string
	for v := range resultChan {
		result = append(result, v)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
