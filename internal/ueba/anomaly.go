package ueba

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

// IsolationForest represents the ensemble of isolation trees.
type IsolationForest struct {
	mu        sync.RWMutex
	Trees     []*IsolationTree
	Subsample int
	rng       *rand.Rand
}

// IsolationTree represents a single tree in the forest.
type IsolationTree struct {
	Root *Node
}

type Node struct {
	Left      *Node
	Right     *Node
	SplitAttr string
	SplitVal  float64
	Size      int
	IsLeaf    bool
}

func NewIsolationForest(numTrees int, subsample int) *IsolationForest {
	return &IsolationForest{
		Trees:     make([]*IsolationTree, numTrees),
		Subsample: subsample,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())), // Default to time-based but deterministic per instance
	}
}

// SetSeed allows deterministic forest construction for testing or audit.
func (f *IsolationForest) SetSeed(seed int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rng = rand.New(rand.NewSource(seed))
}

func (f *IsolationForest) Train(profiles []*EntityProfile) {
	if len(profiles) == 0 {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.Trees {
		// Subsample data using the forest's local RNG
		sample := f.sampleProfiles(profiles, f.Subsample)
		f.Trees[i] = &IsolationTree{
			Root: f.buildTree(sample, 0, int(math.Ceil(math.Log2(float64(f.Subsample))))),
		}
	}
}

// Score calculates the abnormality score for a profile.
// Score near 1 indicates anomaly, near 0 indicates normal.
func (f *IsolationForest) Score(p *EntityProfile) float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if len(f.Trees) == 0 {
		return 0.5
	}

	avgPathLen := 0.0
	for _, t := range f.Trees {
		avgPathLen += pathLength(p, t.Root, 0)
	}
	avgPathLen /= float64(len(f.Trees))

	return math.Pow(2, -avgPathLen/c(f.Subsample))
}

func (f *IsolationForest) sampleProfiles(profiles []*EntityProfile, n int) []*EntityProfile {
	if len(profiles) <= n {
		return profiles
	}
	sample := make([]*EntityProfile, n)
	perm := f.rng.Perm(len(profiles))
	for i := 0; i < n; i++ {
		sample[i] = profiles[perm[i]]
	}
	return sample
}

func (f *IsolationForest) buildTree(profiles []*EntityProfile, depth int, maxDepth int) *Node {
	if depth >= maxDepth || len(profiles) <= 1 {
		return &Node{Size: len(profiles), IsLeaf: true}
	}

	// Randomly select a feature that exists in the subset
	features := f.getAvailableFeatures(profiles)
	if len(features) == 0 {
		return &Node{Size: len(profiles), IsLeaf: true}
	}
	attr := features[f.rng.Intn(len(features))]

	// Randomly select a split value between min and max
	min, max := getMinMax(profiles, attr)
	if min == max {
		return &Node{Size: len(profiles), IsLeaf: true}
	}
	splitVal := min + f.rng.Float64()*(max-min)

	// Partition data
	var left, right []*EntityProfile
	for _, p := range profiles {
		p.mu.RLock()
		val := p.FeatureVectors[attr]
		p.mu.RUnlock()

		if val < splitVal {
			left = append(left, p)
		} else {
			right = append(right, p)
		}
	}

	return &Node{
		SplitAttr: attr,
		SplitVal:  splitVal,
		Left:      f.buildTree(left, depth+1, maxDepth),
		Right:     f.buildTree(right, depth+1, maxDepth),
		IsLeaf:    false,
	}
}

func pathLength(p *EntityProfile, n *Node, currentDepth int) float64 {
	if n.IsLeaf {
		if n.Size <= 1 {
			return float64(currentDepth)
		}
		return float64(currentDepth) + c(n.Size)
	}

	val := p.FeatureVectors[n.SplitAttr]
	if val < n.SplitVal {
		return pathLength(p, n.Left, currentDepth+1)
	}
	return pathLength(p, n.Right, currentDepth+1)
}

// c(n) is the average path length of unsuccessful search in Binary Search Tree.
func c(n int) float64 {
	if n <= 1 {
		return 0
	}
	if n == 2 {
		return 1
	}
	return 2*(math.Log(float64(n-1))+0.5772156649) - (2*float64(n-1))/float64(n)
}

func (f *IsolationForest) getAvailableFeatures(profiles []*EntityProfile) []string {
	if len(profiles) == 0 {
		return nil
	}
	// Use a map to find the union of all features in the sample
	featureSet := make(map[string]struct{})
	for _, p := range profiles {
		p.mu.RLock()
		for k := range p.FeatureVectors {
			featureSet[k] = struct{}{}
		}
		p.mu.RUnlock()
	}

	var features []string
	for k := range featureSet {
		features = append(features, k)
	}
	return features
}

func getMinMax(profiles []*EntityProfile, attr string) (float64, float64) {
	min := math.MaxFloat64
	max := -math.MaxFloat64
	for _, p := range profiles {
		p.mu.RLock()
		val := p.FeatureVectors[attr]
		p.mu.RUnlock()
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}
	return min, max
}
