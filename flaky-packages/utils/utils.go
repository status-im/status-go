package utils

import (
	"fmt"
	"time"
)

func Foo() {
	fmt.Println("fake coverage 1")
	fmt.Println("fake coverage 2")
	fmt.Println("fake coverage 3")
}

func Sleep() {
	time.Sleep(3 * time.Second)
}

func LogFlakiness() {
	fmt.Println("flaky")
}
