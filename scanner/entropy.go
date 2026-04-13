package scanner

import (
	"math"
)

// CalculateShannonEntropy calculates the Shannon entropy of a given string.
// Higher entropy indicates more randomness (potential secret).
func CalculateShannonEntropy(data string) float64 {
	if len(data) == 0 {
		return 0.0
	}

	frequencies := make(map[rune]float64)
	for _, char := range data {
		frequencies[char]++
	}

	var entropy float64
	length := float64(len(data))

	for _, count := range frequencies {
		probability := count / length
		entropy -= probability * math.Log2(probability)
	}

	return entropy
}
