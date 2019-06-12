package adapters

import "math/rand"

func randomItem(items []string) string {
	l := len(items)
	return items[rand.Intn(l)]
}
