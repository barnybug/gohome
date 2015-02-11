package util

import "fmt"

func ExampleSortedKeys() {
	keys := SortedKeys(map[string]interface{}{
		"a": 1,
		"d": 2,
		"b": 3,
		"c": 4,
	})
	fmt.Println(keys)
	// Output:
	// [a b c d]
}
