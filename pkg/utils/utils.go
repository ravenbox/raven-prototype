package utils

import (
	"cmp"
)

func MapKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for id := range m {
		keys = append(keys, id)
	}
	return keys
}
