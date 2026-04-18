package ueba

import (
	"math/rand"
	"testing"
)

func TestIsolationForest_AnomalyDetection(t *testing.T) {
	// 1. Setup deterministic forest with standard high-integrity parameters
	forest := NewIsolationForest(200, 256)
	forest.SetSeed(42)

	// 2. Generate "normal" baseline: 300 users with event_frequency ~ 1.0
	var baseline []*EntityProfile
	rng := rand.New(rand.NewSource(101))
	for i := 0; i < 300; i++ {
		p := &EntityProfile{
			ID:             "normal_user",
			FeatureVectors: map[string]float64{"event_frequency": 1.0 + rng.Float64()*0.1},
		}
		baseline = append(baseline, p)
	}

	// 3. Train forest
	forest.Train(baseline)

	// 4. Test normal scoring
	normal := &EntityProfile{
		ID:             "normal_test",
		FeatureVectors: map[string]float64{"event_frequency": 1.05},
	}
	normalScore := forest.Score(normal)
	if normalScore > 0.55 {
		t.Fatalf("Normal profile score too high: %v (expected < 0.55)", normalScore)
	}

	// 5. Test anomalous scoring (Clear Outlier)
	anomaly := &EntityProfile{
		ID:             "attacker",
		FeatureVectors: map[string]float64{"event_frequency": 100.0},
	}
	anomalyScore := forest.Score(anomaly)
	if anomalyScore < 0.65 {
		t.Fatalf("Anomaly profile score too low: %v (expected > 0.65)", anomalyScore)
	}

	t.Logf("Detection success: Normal(%v) vs Anomaly(%v)", normalScore, anomalyScore)
}

func TestIsolationForest_MultiFeature(t *testing.T) {
	forest := NewIsolationForest(200, 256)
	forest.SetSeed(123)

	var baseline []*EntityProfile
	rng := rand.New(rand.NewSource(101))
	for i := 0; i < 300; i++ {
		baseline = append(baseline, &EntityProfile{
			FeatureVectors: map[string]float64{
				"event_frequency": 1.0 + rng.Float64()*0.01,
				"failed_logins":   0.0 + rng.Float64()*0.01,
			},
		})
	}
	forest.Train(baseline)

	// Anomaly in ONE feature (Significantly higher than baseline variance)
	anomaly := &EntityProfile{
		FeatureVectors: map[string]float64{
			"event_frequency": 1.0,
			"failed_logins":   100.0,
		},
	}
	score := forest.Score(anomaly)
	if score < 0.6 {
		t.Fatalf("Failed to detect single-feature anomaly: %v (expected > 0.6)", score)
	}
}
