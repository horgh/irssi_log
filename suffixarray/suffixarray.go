/*
 * suffixarray provides a simple suffix array implementation.
 *
 * I use it for text generation.
 *
 * Note there is a suffixarray in the core library (index/suffixarray).
 */

package suffixarray

import (
	"sort"
)

// Build takes a text and generates a suffix array.
func Build(text string) ([]string, error) {
	var suffixes []string
	suffixes = append(suffixes, text[0:])

	for i, c := range text {
		if c == ' ' {
			suffixes = append(suffixes, text[i+1:])
			continue
		}
	}

	return suffixes, nil
}

// Sort sorts the suffix array.
func Sort(suffixArray []string) ([]string, error) {
	sort.Strings(suffixArray)
	return suffixArray, nil
}

// Store writes a suffix array to disk. This is so it can be restored later.
//
// The reason this could be useful is to mean loading and sorting the array
// is not needed on restore.
func Store(file string) error {
}
