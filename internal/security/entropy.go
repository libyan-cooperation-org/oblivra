package security

import (
	"math"
)

// CalculateEntropy computes the Shannon entropy of a byte slice.
// Returns a value between 0.0 and 8.0.
// Higher values (approaching 8.0) indicate higher randomness/potential encryption.
func CalculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	counts := make(map[byte]int)
	for _, b := range data {
		counts[b]++
	}

	var entropy float64
	for _, count := range counts {
		p := float64(count) / float64(len(data))
		entropy -= p * math.Log2(p)
	}

	return entropy
}

// IsLowEntropy returns true if the data is likely plain text or highly structured (e.g. zeros).
func IsLowEntropy(data []byte) bool {
	return CalculateEntropy(data) < 4.0
}

// IsHighEntropy returns true if the data is likely encrypted or compressed.
// Compressed data typically scores > 7.5, encrypted > 7.9.
func IsHighEntropy(data []byte) bool {
	return CalculateEntropy(data) > 7.5
}
