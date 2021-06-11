package anonmetrics

import "testing"

func TestFibonacciIntervalIncrementer_Next(t *testing.T) {
	fii := FibonacciIntervalIncrementer{
		Last:    0,
		Current: 1,
	}

	expected := []int64{
		1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377, 610, 987, 1597, 2584, 4181, 6765,
	}

	for i := 0; i < 19; i++ {
		next := fii.Next()
		if next != expected[i] {
			t.Errorf("expected '%d', received '%d'", expected[i], next)
		}
	}
}
