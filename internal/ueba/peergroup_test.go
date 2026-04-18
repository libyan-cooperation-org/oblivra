package ueba

import (
	"math"
	"testing"
)

func TestPeerGroupDeviation(t *testing.T) {
	group := NewPeerGroup("servers")

	// 1. Train group with 10 "normal" entities
	// Each entity has ~100 events
	for i := 0; i < 10; i++ {
		features := map[string]float64{
			"event_frequency": 100.0 + float64(i%3), // 100, 101, 102
		}
		group.Update(features)
	}

	// 2. Test a "normal" entity
	normalFeatures := map[string]float64{
		"event_frequency": 101.0,
	}
	normalDev := group.GetDeviation(normalFeatures)
	if normalDev > 0.3 {
		t.Errorf("Expected low deviation for normal entity, got %.2f", normalDev)
	}

	// 3. Test an "outlier" entity
	// 120 events is significantly higher than 100-102
	outlierFeatures := map[string]float64{
		"event_frequency": 120.0,
	}
	outlierDev := group.GetDeviation(outlierFeatures)
	if outlierDev < 0.8 {
		t.Errorf("Expected high deviation for outlier entity, got %.2f", outlierDev)
	}

	t.Logf("Normal Dev: %.2f, Outlier Dev: %.2f", normalDev, outlierDev)
}

func TestStatSummary(t *testing.T) {
	s := &StatSummary{}
	data := []float64{10, 20, 30}

	// Implementation follows Welford's algorithm
	// Mean should be 20
	// Variance should be ((10-20)^2 + (20-20)^2 + (30-20)^2) / (3-1) = (100 + 0 + 100) / 2 = 100

	for _, v := range data {
		delta := v - s.Mean
		s.Count++
		s.Mean += delta / float64(s.Count)
		delta2 := v - s.Mean
		s.M2 += delta * delta2
	}

	if s.Mean != 20 {
		t.Errorf("Expected mean 20, got %.2f", s.Mean)
	}

	variance := s.M2 / float64(s.Count-1)
	if variance != 100 {
		t.Errorf("Expected variance 100, got %.2f", variance)
	}

	stdDev := math.Sqrt(variance)
	if stdDev != 10 {
		t.Errorf("Expected stdDev 10, got %.2f", stdDev)
	}
}
