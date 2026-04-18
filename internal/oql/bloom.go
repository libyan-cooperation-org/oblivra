package oql

import (
	"hash/fnv"
	"math"
)

// BloomFilter is a probabilistic data structure for set membership.
type BloomFilter struct {
	bits   []uint64
	m      uint64 // number of bits
	k      uint32 // number of hash functions
	count  uint64
}

// NewBloomFilter creates a filter optimized for the given capacity and false positive rate.
func NewBloomFilter(n uint64, p float64) *BloomFilter {
	if n == 0 { n = 1000 }
	if p == 0 { p = 0.01 }
	
	m := uint64(math.Ceil(-float64(n) * math.Log(p) / math.Pow(math.Log(2), 2)))
	k := uint32(math.Ceil(math.Log(2) * float64(m) / float64(n)))
	
	// Round m up to nearest 64 for uint64 slice
	size := (m + 63) / 64
	return &BloomFilter{
		bits: make([]uint64, size),
		m:    size * 64,
		k:    k,
	}
}

// Add inserts a string into the filter.
func (f *BloomFilter) Add(s string) {
	h1, h2 := hashString(s)
	for i := uint32(0); i < f.k; i++ {
		idx := (h1 + uint64(i)*h2) % f.m
		f.bits[idx/64] |= (1 << (idx % 64))
	}
	f.count++
}

// Test checks if a string might be in the filter.
func (f *BloomFilter) Test(s string) bool {
	h1, h2 := hashString(s)
	for i := uint32(0); i < f.k; i++ {
		idx := (h1 + uint64(i)*h2) % f.m
		if f.bits[idx/64]&(1<<(idx%64)) == 0 {
			return false
		}
	}
	return true
}

func hashString(s string) (uint64, uint64) {
	h := fnv.New64a()
	h.Write([]byte(s))
	v := h.Sum64()
	return v, v >> 32 // Simplified double hashing
}
